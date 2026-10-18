import copy
import json
import sys
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from release import ApiError, Cloud, Config, GitHub, Release, ReleaseError, TAG, preview_tls

COMMIT = "a" * 40


def config(branch="main", attempt="1"):
    return Config(branch, COMMIT, "123456", attempt)


REPO = config().repository
OLD = REPO + "@sha256:" + "1" * 64
NEW = REPO + "@sha256:" + "2" * 64


def registry_command(args):
    if args[:3] == ["gcloud", "artifacts", "docker"]:
        return args[5].split("@", 1)[1] if "@" in args[5] else "sha256:" + "1" * 64
    return json.dumps({"credential": {"access_token": "token", "token_expiry": "2100-01-01T00:00:00Z"}})


class Clock:
    def __init__(self):
        self.now = 0

    def __call__(self):
        return self.now

    def sleep(self, duration):
        self.now += duration


class API:
    """Stateful Cloud Run API fake, including CAS, pending traffic and ambiguous writes."""
    def __init__(self, cfg):
        self.config = cfg
        self.events = []
        self.calls = []
        self.sequence = 0
        self.failure = None
        self.deny = False
        self.bad_digest = False
        self.async_operations = False
        self.deny_operations = False
        self.service = {
            "name": f"projects/{cfg.project}/locations/{cfg.region}/services/{cfg.service}",
            "etag": "service-0", "uid": "output-only", "latestReadyRevision": "old",
            "ingress": "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER" if cfg.preview_url else "INGRESS_TRAFFIC_ALL",
            "template": {"revision": "old", "serviceAccount": "runtime@example.iam.gserviceaccount.com",
                         "containers": [{"image": OLD, "env": [{"name": "SECRET", "value": "do-not-persist"}]}]},
            "traffic": [{"type": "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST", "percent": 100}],
            "trafficStatuses": [{"revision": "old", "percent": 100}],
        }
        self.worker = {
            "name": f"projects/{cfg.project}/locations/{cfg.region}/jobs/{cfg.worker}",
            "etag": "worker-0", "runExecutionToken": "never-replay", "generation": "1",
            "template": {"taskCount": 1, "template": {"serviceAccount": "runtime@example.iam.gserviceaccount.com",
                         "containers": [{"image": OLD, "env": [{"name": "SECRET", "value": "do-not-persist"}]}]}},
        }

    def __call__(self, method, url, token, data=None):
        self.calls.append((method, url, copy.deepcopy(data)))
        if ":testIamPermissions" in url:
            if self.deny_operations and "cloudresourcemanager.googleapis.com" in url:
                return {"permissions": []}
            return {"permissions": [] if self.deny else data["permissions"]}
        if "/operations/" in url:
            if self.deny_operations:
                raise ApiError(403, method, url)
            return {"done": True}
        if "/revisions/" in url:
            image = OLD if self.bad_digest else self.service["template"]["containers"][0]["image"]
            return {"containers": [{"image": image}]}
        value = self.worker if "/jobs/" in url else self.service
        if method == "GET":
            return copy.deepcopy(value)
        if method != "PATCH":
            raise AssertionError((method, url))
        if data.get("etag") != value["etag"]:
            raise ApiError(409)
        is_worker = value is self.worker
        if is_worker and "runExecutionToken" in data:
            raise AssertionError("Execution token replayed")
        if not is_worker and "updateMask=" not in url:
            raise AssertionError("Missing service update mask")
        if not is_worker and "template" in data:
            event = "preview"
        elif is_worker:
            event = "worker-new" if data["template"]["template"]["containers"][0]["image"] == NEW else "worker-old"
        else:
            traffic = data["traffic"]
            event = "traffic-new" if any(t.get("revision") == self.config.revision and t.get("percent") == 100 for t in traffic) else "routes-old"
        self.events.append(event)
        if self.failure == "denied-preview" and event == "preview":
            raise ApiError(403)
        if self.failure == "restore" and event == "routes-old":
            raise ApiError(500)
        value.update(copy.deepcopy(data))
        self.sequence += 1
        value["etag"] = f"changed-{self.sequence}"
        if not is_worker:
            if "template" in data:
                value["latestReadyRevision"] = self.config.revision
            if self.failure != "pending-traffic" or event != "traffic-new":
                value["trafficStatuses"] = copy.deepcopy(value["traffic"])
                for target in value["trafficStatuses"]:
                    if target.get("tag"):
                        target["uri"] = "https://release-preview-example.run.app"
        if self.failure == event or self.failure == "pending-traffic" and event == "traffic-new":
            self.failure = None
            raise ApiError(500)  # The server accepted the write; the client did not get success.
        if self.async_operations:
            return {"name": f"projects/{self.config.project}/locations/{self.config.region}/operations/test"}
        return {"done": True}


class ReleaseTests(unittest.TestCase):
    def flow(self, branch="main"):
        cfg = config(branch)
        api = API(cfg)
        clock = Clock()
        cloud = Cloud(cfg, api=api, run=Mock(side_effect=registry_command),
                      clock=clock, sleep=clock.sleep)
        github, check, tls = Mock(), Mock(), Mock()
        records = []
        release = Release(cfg, github, cloud, check=check, tls=tls,
                          save=lambda record: records.append(copy.deepcopy(record)))
        snapshot = release.prepare(NEW)
        api.events.clear()
        return release, snapshot, api, check, github, records

    def test_success_preserves_live_traffic_until_preview_passes(self):
        release, snapshot, api, check, github, records = self.flow()
        def verify(url, *identity):
            if url == release.config.preview_url:
                self.assertEqual(Cloud.traffic(api.service), {"old": 100})
                self.assertEqual(Cloud.traffic(api.service, desired=True), {"old": 100})
                self.assertEqual(Cloud.worker_image(api.worker), OLD)
                self.assertEqual(identity, (COMMIT, release.config.revision))
                api.events.append("smoke-preview")
            else:
                self.assertEqual(Cloud.traffic(api.service), {release.config.revision: 100})
                api.events.append("smoke-live")
        check.side_effect = verify
        release.execute(snapshot)
        self.assertEqual(api.events, ["preview", "smoke-preview", "worker-new", "traffic-new", "smoke-live"])
        self.assertEqual(records[-1]["phase"], "verified")
        self.assertEqual(github.current.call_count, 4)

    def test_async_operation_status_is_polled(self):
        release, snapshot, api, _, _, records = self.flow()
        api.async_operations = True
        release.execute(snapshot)
        self.assertEqual(records[-1]["phase"], "verified")
        polls = [url for method, url, _ in api.calls if method == "GET" and "/operations/" in url]
        self.assertEqual(len(polls), 3)

    def test_project_operation_permission_is_required_before_mutation(self):
        release, snapshot, api, _, _, records = self.flow()
        api.deny_operations = True
        with self.assertRaisesRegex(ReleaseError, "run.operations.get"):
            release.prepare(NEW)
        with self.assertRaisesRegex(ReleaseError, "run.operations.get"):
            release.execute(snapshot)
        self.assertFalse(api.events)
        self.assertEqual(records[-1]["failed_phase"], "preflight")
        self.assertEqual(records[-1]["rollback"], "not_needed")

    def test_operation_denial_after_accepted_write_reports_failed_phase(self):
        release, snapshot, api, check, _, records = self.flow()
        api.async_operations = True
        original = api.__call__
        def deny_status(method, url, token, data=None):
            if "/operations/" in url:
                raise ApiError(403, method, url)
            return original(method, url, token, data)
        release.cloud.api = deny_status
        with self.assertRaisesRegex(ReleaseError, "deploying_preview.*HTTP 403"):
            release.execute(snapshot)
        self.assertEqual(Cloud.traffic(api.service), {"old": 100})
        self.assertEqual(Cloud.worker_image(api.worker), OLD)
        self.assertEqual(Cloud.tags(api.service), {})
        check.assert_not_called()
        self.assertEqual(records[-1]["failed_phase"], "deploying_preview")
        self.assertIn("/operations/", records[-1]["error"])

    def test_api_error_excludes_query_and_credentials(self):
        error = ApiError(403, "GET", "https://user:secret@run.googleapis.com/v2/operations/test?token=secret")
        self.assertNotIn("secret", str(error))
        self.assertIn("GET run.googleapis.com/v2/operations/test", str(error))

    def test_missing_or_mutable_image_build_output_blocks_preparation(self):
        for image in ("", REPO + "@", REPO + ":latest"):
            release, _, api, _, _, _ = self.flow()
            with self.subTest(image=image), self.assertRaises(ReleaseError):
                release.prepare(image)
            self.assertFalse(api.events)

    def test_worker_authentication_refresh_preserves_release_order(self):
        release, snapshot, api, _, _, records = self.flow()
        original = api.__call__
        denied = []
        def authenticate(method, url, token, data=None):
            if method == "PATCH" and "/jobs/" in url and not denied:
                denied.append(True)
                raise ApiError(401, method, url)
            return original(method, url, token, data)
        release.cloud.api = authenticate
        release.execute(snapshot)
        self.assertEqual(api.events, ["preview", "worker-new", "traffic-new"])
        self.assertEqual(records[-1]["phase"], "verified")

    def test_repeated_worker_authentication_failure_still_restores_preview(self):
        release, snapshot, api, _, _, records = self.flow()
        original = api.__call__
        denied = []
        def authenticate(method, url, token, data=None):
            if method == "PATCH" and "/jobs/" in url:
                denied.append(True)
                raise ApiError(401, method, url)
            return original(method, url, token, data)
        release.cloud.api = authenticate
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(len(denied), 2)
        self.assertEqual(Cloud.traffic(api.service), {"old": 100})
        self.assertEqual(Cloud.tags(api.service), {})
        self.assertEqual(Cloud.worker_image(api.worker), OLD)
        self.assertEqual(records[-1]["rollback"], "verified")

    def test_staging_uses_tagged_url_and_its_own_region(self):
        release, snapshot, api, check, _, _ = self.flow("staging")
        release.execute(snapshot)
        self.assertEqual(check.call_args_list[0].args[0], "https://release-preview-example.run.app")
        self.assertTrue(all("europe-north1" in url for method, url, _ in api.calls if "/v2/projects/" in url))

    def test_snapshot_is_safe_and_contains_recoverable_state(self):
        _, snapshot, api, _, _, _ = self.flow()
        encoded = json.dumps(snapshot)
        for excluded in ("do-not-persist", "never-replay", "containers", "env", "token"):
            self.assertNotIn(excluded, encoded)
        self.assertEqual(snapshot["previous_traffic"], {"old": 100})
        self.assertEqual(snapshot["previous_worker_image"], OLD)
        self.assertFalse(api.events)

    def test_missing_permissions_stop_before_mutation(self):
        release, _, api, _, _, _ = self.flow()
        api.deny = True
        with self.assertRaisesRegex(ReleaseError, "permissions"):
            release.prepare(NEW)
        self.assertFalse(api.events)

    def test_snapshot_from_another_attempt_stops_before_mutation(self):
        release, snapshot, api, _, _, records = self.flow()
        snapshot["release_id"] = "gha-123456-2"
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertFalse(api.events)
        self.assertEqual(records[-1]["rollback"], "not_needed")

    def test_snapshot_conflict_stops_before_mutation(self):
        for resource in ("service", "worker"):
            release, snapshot, api, _, _, _ = self.flow()
            getattr(api, resource)["etag"] = "external"
            with self.subTest(resource=resource), self.assertRaises(ReleaseError):
                release.execute(snapshot)
            self.assertFalse(api.events)

    def test_wrong_repository_digest_is_rejected(self):
        release, _, api, _, _, _ = self.flow()
        with self.assertRaises(ReleaseError):
            release.prepare("elsewhere/image@sha256:" + "3" * 64)
        self.assertFalse(api.events)

    def test_preview_failure_restores_old_tag_without_worker_or_promotion(self):
        release, _, api, check, _, records = self.flow()
        api.service["traffic"].append({"revision": "old-tag", "tag": TAG, "percent": 0})
        api.service["trafficStatuses"] = copy.deepcopy(api.service["traffic"])
        # Preserve ordinary traffic as explicit; latest allocation is resolved by the real API.
        api.service["trafficStatuses"][0] = {"revision": "old", "percent": 100}
        snapshot = release.prepare(NEW)
        check.side_effect = RuntimeError("preview failed")
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(api.events, ["preview", "routes-old"])
        self.assertEqual(Cloud.tags(api.service), {TAG: "old-tag"})
        self.assertEqual(records[-1]["rollback"], "verified")

    def test_preview_smoke_error_retains_safe_diagnostic_and_rolls_back(self):
        from smoke_test import SmokeError
        release, snapshot, api, check, _, records = self.flow()
        check.side_effect = SmokeError("CMS smoke test failed: Admin asset is not served by this CMS")
        with self.assertRaisesRegex(ReleaseError, "verifying_preview.*Admin asset is not served"):
            release.execute(snapshot)
        self.assertEqual(records[-1]["error"], str(check.side_effect))
        self.assertEqual(records[-1]["rollback"], "verified")
        self.assertEqual(Cloud.traffic(api.service), {"old": 100})
        self.assertEqual(Cloud.worker_image(api.worker), OLD)

    def test_preview_failure_removes_new_tag(self):
        release, snapshot, api, check, _, records = self.flow()
        check.side_effect = RuntimeError()
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(Cloud.tags(api.service), {})
        self.assertEqual(Cloud.worker_image(api.worker), OLD)
        self.assertEqual(records[-1]["rollback"], "verified")

    def test_wrong_revision_digest_prevents_activation(self):
        release, snapshot, api, check, _, _ = self.flow()
        api.bad_digest = True
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertNotIn("worker-new", api.events)
        check.assert_not_called()

    def test_ambiguous_preview_error_restores_routes(self):
        release, snapshot, api, _, _, records = self.flow()
        api.failure = "preview"
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(Cloud.tags(api.service), {})
        self.assertEqual(records[-1]["rollback"], "verified")

    def test_denied_preview_does_not_restore_or_update_unchanged_resources(self):
        release, snapshot, api, _, _, records = self.flow()
        api.failure = "denied-preview"
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(api.events, ["preview"])
        self.assertEqual(records[-1]["rollback"], "verified")

    def test_obsolete_commit_before_activation_prevents_worker_update(self):
        release, snapshot, api, _, github, _ = self.flow()
        github.current.side_effect = [None, ReleaseError("obsolete")]
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertNotIn("worker-new", api.events)
        self.assertEqual(Cloud.traffic(api.service), {"old": 100})

    def test_worker_traffic_and_live_failures_restore_both_resources(self):
        for failure in ("worker-new", "traffic-new", "pending-traffic", "live"):
            release, snapshot, api, check, _, records = self.flow()
            if failure == "live":
                check.side_effect = [None, RuntimeError("live failed"), None]
            else:
                api.failure = failure
            with self.subTest(failure=failure), self.assertRaises(ReleaseError):
                release.execute(snapshot)
            self.assertEqual(Cloud.traffic(api.service), {"old": 100})
            self.assertEqual(Cloud.traffic(api.service, desired=True), {"old": 100})
            self.assertEqual(Cloud.worker_image(api.worker), OLD)
            self.assertEqual(Cloud.tags(api.service), {})
            self.assertEqual(records[-1]["rollback"], "verified")
            if failure != "worker-new":
                self.assertEqual(check.call_args.args, (release.config.live_url,))

    def test_failure_to_restore_is_recorded(self):
        release, snapshot, api, check, _, records = self.flow()
        def verify(url, *args):
            if url == release.config.live_url and args:
                api.failure = "restore"
                raise RuntimeError()
        check.side_effect = verify
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(records[-1]["rollback"], "failed")
        self.assertTrue(records[-1]["recovery_errors"])

    def test_external_config_and_traffic_are_not_overwritten(self):
        for change in ("traffic", "service", "worker", "tag"):
            release, snapshot, api, check, _, records = self.flow()
            def verify(url, *args):
                if url == release.config.live_url and args:
                    if change == "traffic":
                        api.service["traffic"] = [{"revision": "external", "percent": 100}]
                        api.service["trafficStatuses"] = copy.deepcopy(api.service["traffic"])
                    elif change == "service":
                        api.service["ingress"] = "INGRESS_TRAFFIC_INTERNAL_ONLY"
                    elif change == "worker":
                        api.worker["template"]["taskCount"] = 9
                    else:
                        api.service["traffic"].append({"revision": "external", "tag": TAG, "percent": 0})
                    raise RuntimeError()
            check.side_effect = verify
            with self.subTest(change=change), self.assertRaises(ReleaseError):
                release.execute(snapshot)
            self.assertEqual(records[-1]["rollback"], "failed")
            self.assertNotIn("worker-old" if change == "worker" else "routes-old", api.events)

    def test_mutable_previous_worker_tag_is_resolved_and_recovered_by_digest(self):
        release, _, api, check, _, records = self.flow()
        tag = REPO + ":previous"
        api.worker["template"]["template"]["containers"][0]["image"] = tag
        release.cloud.run.side_effect = registry_command
        snapshot = release.prepare(NEW)
        check.side_effect = [None, RuntimeError(), None]
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(Cloud.worker_image(api.worker), OLD)
        self.assertEqual(records[-1]["rollback"], "verified")

    def test_preview_ingress_and_tls_fail_closed(self):
        for defect in ("ingress", "tls"):
            release, _, api, _, _, _ = self.flow()
            if defect == "ingress":
                api.service["ingress"] = "INGRESS_TRAFFIC_ALL"
            else:
                release.tls.side_effect = ReleaseError("TLS failed")
            with self.subTest(defect=defect), self.assertRaises(ReleaseError):
                release.prepare(NEW)
            self.assertFalse(api.events)

    def test_missing_or_conflicting_traffic_is_rejected(self):
        for traffic in ([], [{"percent": 100}], [{"revision": "old", "percent": 50}]):
            with self.subTest(traffic=traffic), self.assertRaises(ReleaseError):
                Cloud.traffic({"trafficStatuses": traffic})

    def test_api_patch_uses_cas_and_preserves_settings(self):
        release, snapshot, api, _, _, _ = self.flow()
        release.execute(snapshot)
        patches = [call for call in api.calls if call[0] == "PATCH"]
        self.assertTrue(all(call[2].get("etag") for call in patches))
        self.assertTrue(all("uid" not in call[2] and "runExecutionToken" not in call[2] for call in patches))
        self.assertEqual(api.service["template"]["containers"][0]["env"][0]["value"], "do-not-persist")
        self.assertTrue("updateMask=" in patches[0][1])

    def test_read_timeout_never_becomes_a_success(self):
        clock = Clock()
        cloud = Cloud(config(), run=Mock(side_effect=registry_command),
                      api=Mock(return_value={"reconciling": True}), clock=clock, sleep=clock.sleep)
        with self.assertRaisesRegex(ReleaseError, "reconciling"):
            cloud.stable("services")
        self.assertEqual(clock.now, 240)

    def test_operation_errors_and_timeouts_fail(self):
        cloud = Cloud(config())
        with self.assertRaisesRegex(ReleaseError, "failed"):
            cloud.operation({"done": True, "error": {"message": "not logged"}})
        clock = Clock()
        cloud = Cloud(config(), run=Mock(side_effect=registry_command), api=Mock(return_value={"name": "operation"}),
                      clock=clock, sleep=clock.sleep)
        with self.assertRaisesRegex(ReleaseError, "timed out"):
            cloud.operation({"name": "operation"})

    def test_cas_conflict_prevents_patch(self):
        release, _, api, _, _, _ = self.flow()
        old = copy.deepcopy(api.service)
        api.service["etag"] = "external"
        with self.assertRaisesRegex(ReleaseError, "another actor"):
            release.cloud.set_routes({"old": 100}, {}, old)
        self.assertFalse(api.events)

    def test_config_and_github_reject_invalid_identity_or_branch(self):
        for branch, commit, run, attempt in (
                ("other", COMMIT, "1", "1"), ("main", "short", "1", "1"),
                ("main", COMMIT, "1;cmd", "1"), ("main", COMMIT, "1", "0")):
            with self.subTest(branch=branch, commit=commit, run=run, attempt=attempt), self.assertRaises(ReleaseError):
                Config(branch, commit, run, attempt)
        for response in ({}, {"sha": "b" * 40}):
            with self.assertRaises(ReleaseError):
                GitHub(config(), "token", api=Mock(return_value=response)).current()
        with self.assertRaises(ApiError):
            GitHub(config(), "token", api=Mock(side_effect=ApiError(403))).current()
        with self.assertRaises(ReleaseError):
            GitHub(config(), "").current()

    def test_attempts_have_distinct_release_and_revision_names(self):
        self.assertNotEqual(config().revision, config(attempt="2").revision)

    def test_tls_dns_and_certificate_errors_stop_preflight(self):
        with patch("release.socket.create_connection", side_effect=OSError("DNS failed")):
            with self.assertRaises(ReleaseError):
                preview_tls("https://preview.cms.armada.nu")


    def test_branch_advancing_during_worker_update_rolls_worker_back(self):
        release, snapshot, api, _, github, records = self.flow()
        github.current.side_effect = [None, None, ReleaseError("obsolete commit")]
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertNotIn("traffic-new", api.events)
        self.assertEqual(Cloud.worker_image(api.worker), OLD)
        self.assertEqual(records[-1]["rollback"], "verified")

    def test_deleted_previous_image_blocks_release(self):
        release, _, api, _, _, _ = self.flow()
        release.cloud.run.side_effect = ReleaseError("Registry image missing")
        with self.assertRaises(ReleaseError):
            release.prepare(NEW)
        self.assertFalse(api.events)

    def test_external_worker_update_during_live_check_never_reports_success(self):
        release, snapshot, api, check, _, records = self.flow()
        def verify(url, *args):
            if url == release.config.live_url:
                api.worker["etag"] = "external"
                api.worker["template"]["taskCount"] = 7
        check.side_effect = verify
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(records[-1]["phase"], "failed")
        self.assertEqual(records[-1]["rollback"], "failed")
        self.assertEqual(api.worker["template"]["taskCount"], 7)


    def test_worker_cas_conflict_never_reverts_an_external_identical_image(self):
        release, snapshot, api, _, _, records = self.flow()
        original = release.cloud.set_worker
        def conflicting_update(image, expected):
            api.worker["template"]["template"]["containers"][0]["image"] = NEW
            api.worker["etag"] = "external"
            return original(image, expected)
        release.cloud.set_worker = conflicting_update
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(Cloud.worker_image(api.worker), NEW)
        self.assertNotIn("worker-old", api.events)
        self.assertEqual(records[-1]["rollback"], "failed")

    def test_traffic_cas_conflict_never_reverts_an_external_identical_route(self):
        release, snapshot, api, _, _, records = self.flow()
        original = release.cloud.set_routes
        def conflicting_update(traffic, tags, expected):
            api.service["traffic"] = Cloud.routes(traffic, tags)
            api.service["trafficStatuses"] = copy.deepcopy(api.service["traffic"])
            api.service["etag"] = "external"
            return original(traffic, tags, expected)
        release.cloud.set_routes = conflicting_update
        with self.assertRaises(ReleaseError):
            release.execute(snapshot)
        self.assertEqual(Cloud.traffic(api.service), {release.config.revision: 100})
        self.assertNotIn("routes-old", api.events)
        self.assertEqual(records[-1]["rollback"], "failed")
