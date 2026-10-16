"""Security/release boundaries and executable aggregate job contracts."""
import itertools
import os
import re
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = ROOT / ".github" / "workflows"


class WorkflowContractTests(unittest.TestCase):
    def test_deploy_is_push_only_and_requires_all_five_gates(self):
        text = (WORKFLOWS / "deploy.yml").read_text()
        self.assertIn("needs: [go, frontend, supabase, codeql, release-checks]", text)
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
        self.assertEqual(text.count("id-token: write"), 1)

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
