"""Commit-bound Cloud Run release orchestration. External calls are injectable."""

import argparse
import copy
import hashlib
import socket
import ssl
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path
from urllib.error import HTTPError
from urllib.parse import urlencode, urlsplit
from urllib.request import Request, urlopen

from smoke_test import smoke


class ReleaseError(RuntimeError):
    pass


class ConflictError(ReleaseError):
    def __init__(self, *kinds):
        self.kinds = set(kinds)
        super().__init__("Resource was changed by another actor")


class ApiError(ReleaseError):
    def __init__(self, status):
        self.status = status
        super().__init__("Remote API returned HTTP " + str(status))


def command(args):
    try:
        result = subprocess.run(args, capture_output=True, text=True, check=False, timeout=240)
    except subprocess.TimeoutExpired:
        raise ReleaseError("Command timed out: " + " ".join(args[:3])) from None
    if result.returncode:
        # CLI output can contain runtime configuration; never forward it to logs.
        raise ReleaseError("Command failed: " + " ".join(args[:3]))
    return result.stdout.strip()


def request(method, url, token, data=None):
    encoded = json.dumps(data).encode() if data is not None else None
    req = Request(url, data=encoded, method=method, headers={
        "Authorization": "Bearer " + token,
        "Content-Type": "application/json",
        "Accept": "application/json",
        "X-GitHub-Api-Version": "2026-03-10",
    })
    try:
        with urlopen(req, timeout=30) as response:
            body = response.read()
            return json.loads(body) if body else {}
    except HTTPError as error:
        status = error.code
        error.close()
        raise ApiError(status) from None
    except (OSError, ValueError):
        raise ReleaseError("Remote API request failed") from None


TARGETS = json.loads(Path(__file__).with_name("release-targets.json").read_text(encoding="utf-8"))
TAG = "release-preview"


class Config:
    def __init__(self, branch, commit, run_id, attempt):
        if branch not in TARGETS:
            raise ReleaseError("Unsupported release branch")
        if not re.fullmatch(r"[a-f0-9]{40}", commit):
            raise ReleaseError("A full source commit SHA is required")
        if not re.fullmatch(r"[1-9][0-9]{0,19}", run_id) or not re.fullmatch(r"[1-9][0-9]{0,9}", attempt):
            raise ReleaseError("Invalid Actions run identity")
        self.branch, self.commit = branch, commit
        self.release_id = f"gha-{run_id}-{attempt}"
        for name, value in TARGETS[branch].items():
            setattr(self, name, value)
        self.revision = self.service + "-" + self.release_id
        if len(self.revision) > 63:
            raise ReleaseError("Revision name exceeds Cloud Run limit")
        self.run_url = f"https://github.com/armada-ths/ArmadaCMS/actions/runs/{run_id}/attempts/{attempt}"

    @classmethod
    def environment(cls):
        if os.environ.get("GITHUB_REPOSITORY") != "armada-ths/ArmadaCMS" or os.environ.get("GITHUB_EVENT_NAME") != "push":
            raise ReleaseError("Release requires a push in the expected repository")
        return cls(*(os.environ.get(key, "") for key in
                     ("GITHUB_REF_NAME", "GITHUB_SHA", "GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT")))

    def image(self, value):
        if not re.fullmatch(re.escape(self.repository) + r"@sha256:[a-f0-9]{64}", value):
            raise ReleaseError("A verified image digest from the target repository is required")
        return value

    def identity(self):
        return {"release_id": self.release_id, "commit": self.commit, "branch": self.branch,
                "service": self.service, "project": self.project, "region": self.region,
                "revision": self.revision, "run_url": self.run_url}


class GitHub:
    def __init__(self, config, token, *, api=request):
        self.config, self.token, self.api = config, token, api

    def current(self):
        if not self.token:
            raise ReleaseError("GitHub token is missing")
        head = self.api("GET", "https://api.github.com/repos/armada-ths/ArmadaCMS/commits/" +
                        self.config.branch, self.token)
        if head.get("sha") != self.config.commit:
            raise ReleaseError("Release commit is no longer the branch head")


def fingerprint(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def write_json(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    temporary.replace(path)


def preview_tls(url):
    """Verify DNS, SNI and certificate before any production service mutation."""
    parsed = urlsplit(url)
    if parsed.scheme != "https" or not parsed.hostname or parsed.port not in (None, 443):
        raise ReleaseError("Invalid preview URL")
    try:
        with socket.create_connection((parsed.hostname, 443), timeout=10) as connection:
            with ssl.create_default_context().wrap_socket(connection, server_hostname=parsed.hostname):
                pass
    except OSError:
        raise ReleaseError("Preview DNS or TLS is not ready") from None


class Cloud:
    SERVICE_FIELDS = ("name", "description", "labels", "annotations", "client", "clientVersion",
                      "ingress", "launchStage", "binaryAuthorization", "template", "scaling",
                      "invokerIamDisabled", "defaultUriDisabled", "iapEnabled", "multiRegionSettings",
                      "customAudiences", "threatDetectionEnabled", "buildConfig", "sshEnabled")
    JOB_FIELDS = ("name", "labels", "annotations", "client", "clientVersion",
                  "launchStage", "binaryAuthorization", "template")

    def __init__(self, config, *, run=command, api=request, clock=time.monotonic, sleep=time.sleep):
        self.config, self.run, self.api = config, run, api
        self.clock, self.sleep = clock, sleep
        self.parent = f"projects/{config.project}/locations/{config.region}"

    def call(self, method, url, data=None):
        return self.api(method, url, self.run(["gcloud", "auth", "print-access-token"]), data)

    def resource(self, kind):
        name = self.config.service if kind == "services" else self.config.worker
        return f"https://run.googleapis.com/v2/{self.parent}/{kind}/{name}"

    def service(self):
        return self.call("GET", self.resource("services"))

    def worker(self):
        return self.call("GET", self.resource("jobs"))

    def stable(self, kind):
        deadline = self.clock() + 240
        while True:
            value = self.call("GET", self.resource(kind))
            if not value.get("reconciling"):
                if not value.get("etag"):
                    raise ReleaseError("Resource etag is missing")
                return value
            if self.clock() >= deadline:
                raise ReleaseError("Resource is still reconciling; recovery requires review")
            self.sleep(2)

    @staticmethod
    def traffic(service, *, desired=False):
        result = {}
        for target in service.get("traffic" if desired else "trafficStatuses", []):
            percent = target.get("percent", 0)
            if not percent:
                continue
            revision = target.get("revision", "").rsplit("/", 1)[-1]
            if desired and target.get("type") == "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST":
                revision = service.get("latestReadyRevision", "").rsplit("/", 1)[-1]
            if not re.fullmatch(r"[a-z0-9-]+", revision) or not isinstance(percent, int) or percent < 0:
                raise ReleaseError("Traffic is not bound to explicit revisions")
            result[revision] = result.get(revision, 0) + percent
        if sum(result.values()) != 100:
            raise ReleaseError("A complete traffic distribution is required")
        return result

    @staticmethod
    def tags(service):
        targets = service.get("traffic", [])
        result = {}
        for target in targets:
            if target.get("tag"):
                revision = target.get("revision", "").rsplit("/", 1)[-1]
                if not re.fullmatch(r"[a-z0-9-]+", revision):
                    raise ReleaseError("Tag is not bound to an explicit revision")
                if target["tag"] in result:
                    raise ReleaseError("Duplicate traffic tag")
                result[target["tag"]] = revision
        return result

    @staticmethod
    def routes(traffic, tags):
        return ([{"type": "TRAFFIC_TARGET_ALLOCATION_TYPE_REVISION", "revision": revision, "percent": percent}
                 for revision, percent in sorted(traffic.items())] +
                [{"type": "TRAFFIC_TARGET_ALLOCATION_TYPE_REVISION", "revision": revision, "tag": tag, "percent": 0}
                 for tag, revision in sorted(tags.items())])

    @staticmethod
    def worker_image(worker):
        return worker["template"]["template"]["containers"][0]["image"]

    @classmethod
    def config_hash(cls, value, kind):
        fields = cls.SERVICE_FIELDS if kind == "services" else cls.JOB_FIELDS
        selected = copy.deepcopy({key: value[key] for key in fields if key in value})
        template = selected["template"] if kind == "services" else selected["template"]["template"]
        template["containers"][0].pop("image", None)
        if kind == "services":
            template.pop("revision", None)
        return fingerprint(selected)

    def permissions(self, service, worker):
        checks = [
            (self.resource("services"), ["run.services.get", "run.services.update", "run.revisions.get"]),
            (self.resource("jobs"), ["run.jobs.get", "run.jobs.update"]),
        ]
        identities = {service["template"].get("serviceAccount"),
                      worker["template"]["template"].get("serviceAccount")}
        if None in identities or "" in identities:
            raise ReleaseError("Runtime service account is missing")
        for identity in identities:
            checks.append(("https://iam.googleapis.com/v1/projects/-/serviceAccounts/" + identity,
                           ["iam.serviceAccounts.actAs"]))
        for resource, permissions in checks:
            result = self.call("POST", resource + ":testIamPermissions", {"permissions": permissions})
            if not set(permissions).issubset(result.get("permissions", [])):
                raise ReleaseError("Deployment permissions are incomplete")

    def resolve_image(self, image):
        digest = self.run(["gcloud", "artifacts", "docker", "images", "describe", image,
                           "--project=" + self.config.project, "--format=value(image_summary.digest)"])
        if not re.fullmatch(r"sha256:[a-f0-9]{64}", digest):
            raise ReleaseError("Previous image digest is unavailable")
        if "@" in image:
            repository, expected_digest = image.rsplit("@", 1)
            if digest != expected_digest:
                raise ReleaseError("Previous image digest does not match the registry")
        else:
            repository = image.rsplit(":", 1)[0]
        return repository + "@" + digest

    def operation(self, operation):
        deadline = self.clock() + 240
        while not operation.get("done"):
            if self.clock() >= deadline:
                raise ReleaseError("Cloud Run operation timed out")
            self.sleep(2)
            operation = self.call("GET", "https://run.googleapis.com/v2/" + operation["name"])
        if operation.get("error"):
            raise ReleaseError("Cloud Run operation failed")

    def patch(self, kind, expected, changes):
        current = self.stable(kind)
        if current["etag"] != expected["etag"]:
            raise ConflictError(kind)
        if kind == "services":
            payload = {"name": current["name"], "etag": current["etag"], **changes}
            url = self.resource(kind) + "?" + urlencode({"updateMask": ",".join(changes)})
        else:
            # The Job API has no updateMask. Copy only mutable fields, never execution tokens.
            payload = {key: current[key] for key in self.JOB_FIELDS if key in current}
            payload.update(etag=current["etag"], **changes)
            url = self.resource(kind)
        try:
            operation = self.call("PATCH", url, payload)
        except ApiError as error:
            if error.status in (409, 412):
                raise ConflictError(kind) from None
            raise
        self.operation(operation)
        updated = self.stable(kind)
        if self.config_hash(updated, kind) != self.config_hash(expected, kind):
            raise ReleaseError("Configuration changed during the Cloud Run operation")
        return updated

    def set_worker(self, image, expected):
        template = copy.deepcopy(expected["template"])
        template["template"]["containers"][0]["image"] = image
        updated = self.patch("jobs", expected, {"template": template})
        if self.worker_image(updated) != image:
            raise ReleaseError("Worker image did not converge")
        return updated

    def set_routes(self, traffic, tags, expected):
        updated = self.patch("services", expected, {"traffic": self.routes(traffic, tags)})
        self.verify_routes(updated, traffic, tags)
        return updated

    def verify_routes(self, service, traffic, tags):
        if (self.traffic(service) != traffic or self.traffic(service, desired=True) != traffic
                or self.tags(service) != tags):
            raise ReleaseError("Cloud Run routing did not converge")
        observed = {target["tag"]: target.get("revision", "").rsplit("/", 1)[-1]
                    for target in service.get("trafficStatuses", []) if target.get("tag")}
        if observed != tags:
            raise ReleaseError("Observed preview tags did not converge")

    def deploy(self, image, previous, snapshot):
        template = copy.deepcopy(previous["template"])
        template["revision"] = self.config.revision
        template["containers"][0]["image"] = image
        tags = {**snapshot["previous_tags"], TAG: self.config.revision}
        service = self.patch("services", previous, {
            "template": template, "traffic": self.routes(snapshot["previous_traffic"], tags)})
        self.verify_routes(service, snapshot["previous_traffic"], tags)
        revision = self.call("GET", self.resource("services") + "/revisions/" + self.config.revision)
        if revision["containers"][0]["image"] != image:
            raise ReleaseError("Preview revision has an unexpected image digest")
        if self.config.preview_url:
            url = self.config.preview_url
        else:
            target = next((entry for entry in service.get("trafficStatuses", []) if entry.get("tag") == TAG), {})
            url = target.get("uri", "")
            parsed = urlsplit(url)
            if parsed.scheme != "https" or not (parsed.hostname or "").endswith(".run.app"):
                raise ReleaseError("Tagged preview URL is missing")
        return service, url


class Release:
    def __init__(self, config, github, cloud, *, check=smoke, tls=preview_tls, save=lambda value: None):
        self.config, self.github, self.cloud = config, github, cloud
        self.check, self.tls, self.save = check, tls, save

    def prepare(self, image):
        self.config.image(image)
        self.github.current()
        service, worker = self.cloud.stable("services"), self.cloud.stable("jobs")
        self.cloud.permissions(service, worker)
        ingress = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER" if self.config.preview_url else "INGRESS_TRAFFIC_ALL"
        if service.get("ingress") != ingress or (not self.config.preview_url and service.get("defaultUriDisabled")):
            raise ReleaseError("Preview ingress does not match the target; ingress is unchanged")
        if self.config.preview_url:
            self.tls(self.config.preview_url)
        traffic = self.cloud.traffic(service)
        if self.cloud.traffic(service, desired=True) != traffic:
            raise ReleaseError("Existing traffic is not stable")
        if len(service["template"]["containers"]) != 1 or len(worker["template"]["template"]["containers"]) != 1:
            raise ReleaseError("Release expects a single-container service and worker")
        snapshot = {**self.config.identity(), "image": image,
                    "previous_traffic": traffic, "previous_tags": self.cloud.tags(service),
                    "previous_worker_image": self.cloud.resolve_image(self.cloud.worker_image(worker)),
                    "previous_worker_reference": self.cloud.worker_image(worker),
                    "previous_service_image": service["template"]["containers"][0]["image"],
                    "previous_template_revision": service["template"].get("revision"),
                    "service_etag": service["etag"], "worker_etag": worker["etag"],
                    "service_config_hash": self.cloud.config_hash(service, "services"),
                    "worker_config_hash": self.cloud.config_hash(worker, "jobs")}
        return snapshot

    def verify_snapshot(self, snapshot):
        if any(snapshot.get(key) != value for key, value in self.config.identity().items()):
            raise ReleaseError("Snapshot belongs to a different release")
        self.config.image(snapshot["image"])
        service, worker = self.cloud.stable("services"), self.cloud.stable("jobs")
        if (service["etag"] != snapshot["service_etag"] or worker["etag"] != snapshot["worker_etag"]
                or self.cloud.resolve_image(self.cloud.worker_image(worker)) != snapshot["previous_worker_image"]):
            raise ReleaseError("Prepared resources were changed by another actor")
        return service, worker

    def recover(self, snapshot, *, worker_attempted, traffic_attempted, blocked=()):
        errors = []
        try:
            if "services" in blocked:
                raise ConflictError("services")
            service = self.cloud.stable("services")
            allowed_traffic = (snapshot["previous_traffic"], {self.config.revision: 100})
            tags = self.cloud.tags(service)
            other_tags = {key: value for key, value in tags.items() if key != TAG}
            old_other = {key: value for key, value in snapshot["previous_tags"].items() if key != TAG}
            if (self.cloud.config_hash(service, "services") != snapshot["service_config_hash"]
                    or service["template"].get("revision") not in
                    (snapshot["previous_template_revision"], self.config.revision)
                    or service["template"]["containers"][0]["image"] not in
                    (snapshot["previous_service_image"], snapshot["image"])
                    or self.cloud.traffic(service) not in allowed_traffic
                    or self.cloud.traffic(service, desired=True) not in allowed_traffic
                    or other_tags != old_other
                    or tags.get(TAG) not in (snapshot["previous_tags"].get(TAG), self.config.revision)):
                raise ConflictError("services")
            if (self.cloud.traffic(service) != snapshot["previous_traffic"]
                    or self.cloud.traffic(service, desired=True) != snapshot["previous_traffic"]
                    or tags != snapshot["previous_tags"]):
                self.cloud.set_routes(snapshot["previous_traffic"], snapshot["previous_tags"], service)
        except Exception:
            errors.append("service_restore_failed_or_external_change")
        if worker_attempted:
            try:
                if "jobs" in blocked:
                    raise ConflictError("jobs")
                worker = self.cloud.stable("jobs")
                image = self.cloud.worker_image(worker)
                if (self.cloud.config_hash(worker, "jobs") != snapshot["worker_config_hash"]
                        or image not in (snapshot["previous_worker_reference"], snapshot["previous_worker_image"], snapshot["image"])):
                    raise ConflictError("jobs")
                if self.cloud.resolve_image(image) != snapshot["previous_worker_image"]:
                    self.cloud.set_worker(snapshot["previous_worker_image"], worker)
            except Exception:
                errors.append("worker_restore_failed_or_external_change")
        if not errors:
            try:
                service, worker = self.cloud.stable("services"), self.cloud.stable("jobs")
                self.cloud.verify_routes(service, snapshot["previous_traffic"], snapshot["previous_tags"])
                if self.cloud.resolve_image(self.cloud.worker_image(worker)) != snapshot["previous_worker_image"]:
                    raise ReleaseError("Worker recovery did not converge")
                if traffic_attempted:
                    # Legacy revisions may have no identity headers. Routing is verified above.
                    self.check(self.config.live_url, accepted_revisions=tuple(snapshot["previous_traffic"]))
            except Exception:
                errors.append("recovery_verification_failed")
        return errors

    def execute(self, snapshot):
        result = {**self.config.identity(), "image": snapshot.get("image"),
                  "phase": "starting", "rollback": "not_needed"}
        def record(phase):
            result["phase"] = phase
            self.save(result)
        worker_attempted = traffic_attempted = preview_attempted = False
        try:
            record("preflight")
            self.github.current()
            previous, worker = self.verify_snapshot(snapshot)
            if self.config.preview_url:
                self.tls(self.config.preview_url)
            record("deploying_preview")
            preview_attempted = True
            service, preview = self.cloud.deploy(snapshot["image"], previous, snapshot)
            result["preview_url"] = preview
            record("verifying_preview")
            self.check(preview, self.config.commit, self.config.revision)
            self.github.current()
            # Re-read both resources before activating; never accept a concurrent edit.
            if self.cloud.stable("services")["etag"] != service["etag"]:
                raise ConflictError("services")
            if self.cloud.stable("jobs")["etag"] != worker["etag"]:
                raise ConflictError("jobs")
            record("updating_worker")
            worker_attempted = True
            worker = self.cloud.set_worker(snapshot["image"], worker)
            self.github.current()
            record("promoting")
            traffic_attempted = True
            service = self.cloud.set_routes({self.config.revision: 100},
                                            self.cloud.tags(service), service)
            record("verifying_live")
            self.check(self.config.live_url, self.config.commit, self.config.revision)
            final_service, final_worker = self.cloud.stable("services"), self.cloud.stable("jobs")
            conflicts = []
            if final_service["etag"] != service["etag"]:
                conflicts.append("services")
            if final_worker["etag"] != worker["etag"]:
                conflicts.append("jobs")
            if conflicts:
                raise ConflictError(*conflicts)
            self.cloud.verify_routes(final_service, {self.config.revision: 100}, self.cloud.tags(service))
            if self.cloud.worker_image(final_worker) != snapshot["image"]:
                raise ReleaseError("Worker changed during live verification")
            record("verified")
            return result
        except Exception as error:
            # Exception text can contain URLs/configuration from an injected client.
            result["error"] = str(error) if isinstance(error, ReleaseError) else type(error).__name__
            if preview_attempted:
                try:
                    record("restoring")
                except OSError:
                    pass  # A local logging failure must never prevent recovery.
                errors = self.recover(snapshot, worker_attempted=worker_attempted,
                                      traffic_attempted=traffic_attempted,
                                      blocked=error.kinds if isinstance(error, ConflictError) else ())
                result["rollback"] = "failed" if errors else "verified"
                result["recovery_errors"] = errors
            record("failed")
            raise ReleaseError("Release failed; rollback " + result["rollback"]) from None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("phase", choices=("target", "prepare", "deploy"))
    parser.add_argument("--image")
    parser.add_argument("--snapshot")
    parser.add_argument("--result")
    args = parser.parse_args()
    try:
        config = Config.environment()
        github = GitHub(config, os.environ.get("GH_TOKEN", ""))
        if args.phase == "target":
            github.current()
            output = {**TARGETS[config.branch], **config.identity(),
                      "image_tag": config.commit + "-" + config.release_id,
                      "registry": config.repository.split("/")[0]}
            with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as stream:
                for key, value in output.items():
                    stream.write(f"{key}={value}\n")
            if args.result:
                write_json(args.result, {**config.identity(), "phase": "awaiting_build", "rollback": "not_needed"})
            return 0
        if not args.snapshot:
            raise ReleaseError("Snapshot path is required")
        cloud = Cloud(config)
        if args.phase == "prepare":
            if not args.image:
                raise ReleaseError("Image digest is required")
            snapshot = Release(config, github, cloud).prepare(args.image)
            write_json(args.snapshot, snapshot)
            if args.result:
                write_json(args.result, {**config.identity(), "image": args.image,
                                        "phase": "prepared", "rollback": "not_needed"})
        else:
            if not args.result:
                raise ReleaseError("Result path is required")
            snapshot = json.loads(Path(args.snapshot).read_text(encoding="utf-8"))
            Release(config, github, cloud, save=lambda value: write_json(args.result, value)).execute(snapshot)
        return 0
    except Exception as error:
        # No raw CLI/API responses or tracebacks in release logs.
        print(str(error) if isinstance(error, ReleaseError) else type(error).__name__, file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
