"""Security/release boundaries and executable aggregate job contracts."""
import itertools
import os
import shutil
import tempfile
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = ROOT / ".github" / "workflows"


class WorkflowContractTests(unittest.TestCase):
    def test_deploy_is_push_only_and_requires_all_five_gates(self):
        text = (WORKFLOWS / "deploy.yml").read_text()
        self.assertIn("needs: [build, go, frontend, supabase, codeql, release-checks]", text)
        header, jobs = text.split("jobs:", 1)
        self.assertIn("branches: [main, staging]", header)
        self.assertNotIn("pull_request", header)
        self.assertNotIn("paths:", header)
        self.assertNotIn("concurrency:", header)
        deploy = jobs.split("\n  deploy:", 1)[1]
        self.assertNotIn("if: always()", deploy.split("steps:", 1)[0])
        self.assertIn("cancel-in-progress: false", deploy)
        self.assertIn("vars.CMS_STAGING_DEPLOY_ENABLED == 'true'", deploy)
        self.assertIn("vars.CMS_PRODUCTION_DEPLOY_ENABLED == 'true'", deploy)
        self.assertEqual(text.count("id-token: write"), 2)

    def test_image_build_runs_independently_with_no_deployment_identity(self):
        text = (WORKFLOWS / "deploy.yml").read_text()
        build = text.split("\n  build:\n", 1)[1].split("\n  deploy:\n", 1)[0]
        deploy = text.split("\n  deploy:\n", 1)[1]
        self.assertNotIn("needs:", build)
        self.assertNotIn("    environment:", build)
        self.assertNotIn("vars.GCP_DEPLOY_SERVICE_ACCOUNT", build)
        self.assertNotIn("scripts.release prepare", build)
        self.assertNotIn("scripts.release deploy", build)
        self.assertIn("CMS_PRODUCTION_BUILD_WIF_PROVIDER", build)
        self.assertIn("CMS_STAGING_BUILD_SERVICE_ACCOUNT", build)
        self.assertIn("image: ${{ steps.target.outputs.repository }}@${{ steps.build.outputs.digest }}", build)
        self.assertIn("build-args: BUILD_COMMIT=${{ github.sha }}", build)
        self.assertIn("IMAGE: ${{ needs.build.outputs.image }}", deploy)
        self.assertNotIn("docker/build-push-action", deploy)
        self.assertIn("cancel-in-progress: true", build)
        self.assertIn("group: cms-build-", build)
        self.assertIn("group: cms-release-", deploy)
        # No status-function override may allow failed/skipped CI or build through the gate.
        condition = deploy.split("    if: >-", 1)[1].split("    runs-on:", 1)[0]
        for bypass in ("always()", "failure()", "cancelled()", "!cancelled()"):
            self.assertNotIn(bypass, condition)

    def test_build_identity_is_registry_only_and_branch_bound_in_both_roots(self):
        for environment, branch in (("prod", "main"), ("staging", "staging")):
            root = ROOT / "infra/terraform/gcp" / environment
            iam = (root / "iam.tf").read_text(encoding="utf-8")
            actions = (root / "actions.tf").read_text(encoding="utf-8")
            builder = iam.split('resource "google_service_account" "build"', 1)[1]
            self.assertIn('role       = "roles/artifactregistry.writer"', builder)
            self.assertNotIn('roles/run.', builder)
            self.assertNotIn('roles/iam.serviceAccountUser', builder)
            reader = iam.split('"deploy_reader"', 1)[1].split('resource ', 1)[0]
            self.assertIn('roles/artifactregistry.reader', reader)
            provider = actions.split('"github_build"', 1)[1].split('resource "google_service_account_iam_member"', 1)[0]
            for condition in ("assertion.repository_owner_id == '6577346'",
                              "assertion.repository_id == '961605060'",
                              "assertion.event_name == 'push'",
                              f"assertion.ref == 'refs/heads/{branch}'",
                              f"assertion.sub == 'repo:armada-ths/ArmadaCMS:ref:refs/heads/{branch}'",
                              f"deploy.yml@refs/heads/{branch}"):
                self.assertIn(condition, provider)
            self.assertIn('from = google_artifact_registry_repository_iam_member.deploy_writer', actions)

    def test_codeql_only_caches_downloaded_go_dependencies(self):
        text = (WORKFLOWS / "codeql.yml").read_text()
        self.assertIn("dependency-caching: ${{ matrix.language == 'go' && 'true' || 'false' }}", text)
        self.assertIn("cache: false", text)
        self.assertIn("build-mode: manual", text)
        self.assertIn("run: go build ./...", text)

    def test_immutable_snapshot_is_published_before_release_and_cannot_be_ignored(self):
        text = (WORKFLOWS / "deploy.yml").read_text()
        before, after = text.split("- name: Verify preview, promote or restore", 1)
        self.assertIn("- name: Persist recovery snapshot before changing Cloud Run", before)
        self.assertIn("if-no-files-found: error", before)
        self.assertNotIn("continue-on-error", text)
        self.assertNotIn("always()", before)
        self.assertIn("--snapshot", after)
        self.assertIn("if: always()", after)
        self.assertIn("retention-days: 90", after)
        self.assertIn("github.run_attempt", before)

    def test_ci_never_cancels_the_caller_workflow_or_receives_cloud_identity(self):
        for name in ("go-checks", "frontend-checks", "supabase-checks", "release-checks", "codeql"):
            text = (WORKFLOWS / (name + ".yml")).read_text()
            header = text.split("jobs:", 1)[0]
            with self.subTest(workflow=name):
                self.assertIn("workflow_call:", header)
                self.assertIn("pull_request:", header)
                self.assertNotIn("  push:", header)
                self.assertIn("cancel-in-progress: ${{ github.event_name == 'pull_request' }}", header)
                self.assertIn("github.ref || github.run_id", header)
                self.assertIn("ci-" + name, header)
                self.assertNotIn("id-token:", text)
                self.assertNotIn("google-github-actions/auth@", text)
                if name in ("go-checks", "frontend-checks", "supabase-checks"):
                    self.assertIn("'.github/workflows/deploy.yml'", text)

    def test_storage_verification_stays_in_the_existing_backend_container(self):
        text = (WORKFLOWS / "supabase-checks.yml").read_text()
        self.assertNotIn("actions/setup-go@", text)
        self.assertNotIn("apt-get", text)
        storage = text.split("- name: Verify local Supabase Storage upload path", 1)[1].split("- name: Stop backend", 1)[0]
        self.assertIn("docker compose -f docker-compose.dev.yml -f \"$CI_COMPOSE_OVERRIDE\" exec -T", storage)
        self.assertIn("backend go test -tags=integration -count=1 -v ./utils", storage)
        self.assertIn("RUN_S3_INTEGRATION: \"1\"", storage)
        for path in ("/storage/v1/s3", "/storage/v1/object/public"):
            self.assertIn("http://host.docker.internal:54321" + path, storage)
        for name in ("RUN_S3_INTEGRATION", "S3_BUCKET", "S3_ENDPOINT", "S3_PUBLIC_URL",
                     "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "S3_REGION"):
            self.assertIn("--env " + name, storage)
        for check in ("supabase db reset --local", "supabase migration list --local",
                      "http://127.0.0.1:8080/health", "supabase stop --no-backup",
                      'target: /go/pkg/mod', 'target: /root/.cache/go-build'):
            self.assertIn(check, text)

    def test_storage_exec_preserves_exit_code_and_keeps_credentials_out_of_arguments(self):
        git_bash = Path("C:/Program Files/Git/bin/bash.exe")
        bash = str(git_bash) if os.name == "nt" and git_bash.exists() else shutil.which("bash")
        if not bash:
            self.skipTest("Bash is unavailable")
        text = (WORKFLOWS / "supabase-checks.yml").read_text()
        storage = text.split("- name: Verify local Supabase Storage upload path", 1)[1].split("      - name: Stop backend", 1)[0]
        script = storage.split("        run: |\n", 1)[1]
        script = "\n".join(line[10:] if line.startswith("          ") else line for line in script.splitlines())
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            docker = root / "docker"
            docker.write_text('#!/bin/sh\nprintf \"%s\\n\" \"$@\" > \"$CAPTURE\"\nexit \"$FAKE_DOCKER_EXIT\"\n')
            docker.chmod(0o755)
            capture = root / "arguments"
            environment = {**os.environ, "PATH": str(root) + os.pathsep + os.environ["PATH"],
                           "CAPTURE": capture.as_posix(), "CI_COMPOSE_OVERRIDE": "test-override.yml",
                           "AWS_SECRET_ACCESS_KEY": "placeholder-secret"}
            for code in ("0", "17"):
                with self.subTest(exit_code=code):
                    result = subprocess.run([bash, "-eu", "-c", script],
                                            env={**environment, "FAKE_DOCKER_EXIT": code},
                                            capture_output=True, timeout=15)
                    self.assertEqual(result.returncode, int(code), result.stderr.decode(errors="replace"))
                    arguments = capture.read_text().splitlines()
                    self.assertEqual(arguments[-7:], ["backend", "go", "test", "-tags=integration", "-count=1", "-v", "./utils"])
                    self.assertNotIn("placeholder-secret", arguments)
                    self.assertIn("AWS_SECRET_ACCESS_KEY", arguments)

    def test_release_artifacts_and_build_context_exclude_credentials(self):
        text = (WORKFLOWS / "deploy.yml").read_text()
        for artifact in text.split("uses: actions/upload-artifact@")[1:]:
            self.assertNotIn("path: .\n", artifact.split("retention-days:", 1)[0])
        for name in (".gitignore", ".dockerignore"):
            self.assertIn("gha-creds-*.json", (ROOT / name).read_text())

    @unittest.skipIf(os.name == "nt", "Bash aggregate contracts run on Linux CI")
    def test_real_aggregate_scripts_accept_only_expected_results(self):
        for name, changed, result_names in (
            ("go-checks", "GO_CHANGED", ("VET_RESULT", "LINT_RESULT", "TEST_RESULT", "DATABASE_TEST_RESULT")),
            ("frontend-checks", "FRONTEND_CHANGED", ("LINT_RESULT", "TYPE_CHECK_RESULT", "FORMAT_RESULT", "TEST_RESULT")),
            ("supabase-checks", "SUPABASE_CHANGED", ("RUNTIME_RESULT",)),
        ):
            text = (WORKFLOWS / (name + ".yml")).read_text()
            # Each file ends with its aggregate's actual Bash script.
            script = text.rsplit("        run: |\n", 1)[1]
            script = "\n".join(line[10:] if line.startswith("          ") else line for line in script.splitlines())
            for changed_value, expected in (("true", "success"), ("false", "skipped")):
                baseline = {**os.environ, "CHANGES_RESULT": "success", changed: changed_value,
                            **dict.fromkeys(result_names, expected)}
                self.assertEqual(subprocess.run(["bash", "-eu", "-c", script], env=baseline,
                                                capture_output=True, timeout=10).returncode, 0)
                for result_name, bad in itertools.product(result_names, ("failure", "cancelled", "timed_out", "neutral", "")):
                    environment = {**baseline, result_name: bad}
                    with self.subTest(workflow=name, changed=changed_value, result=result_name, value=bad):
                        self.assertNotEqual(subprocess.run(["bash", "-eu", "-c", script], env=environment,
                                                           capture_output=True, timeout=10).returncode, 0)
                environment = {**baseline, result_names[0]: "skipped" if expected == "success" else "success"}
                self.assertNotEqual(subprocess.run(["bash", "-eu", "-c", script], env=environment,
                                                  capture_output=True, timeout=10).returncode, 0)
            for change_result, flag in (("failure", "true"), ("cancelled", "false"), ("success", "")):
                environment = {**os.environ, "CHANGES_RESULT": change_result, changed: flag,
                               **dict.fromkeys(result_names, "skipped")}
                self.assertNotEqual(subprocess.run(["bash", "-eu", "-c", script], env=environment,
                                                  capture_output=True, timeout=10).returncode, 0)
