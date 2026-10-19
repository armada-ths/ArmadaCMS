"""Exercise CLI dispatch and failure handling without external services."""
import contextlib
import io
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

from scripts import release
from scripts.tests.test_release import NEW, config


class EntrypointTests(unittest.TestCase):
    def invoke(self, phase, *, snapshot=True, failure=None):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            snapshot_path, result_path = root / "restore.json", root / "result.json"
            if snapshot:
                snapshot_path.write_text(json.dumps(config().identity()))
            flow = Mock()
            flow.prepare.return_value = {**config().identity(), "image": NEW}
            flow.prepare.side_effect = flow.execute.side_effect = failure
            argv = ["release.py", phase, "--snapshot", str(snapshot_path),
                    "--image", NEW, "--result", str(result_path)]
            environment = {"GITHUB_OUTPUT": str(root / "output"), "GH_TOKEN": "not-persisted"}
            with patch.dict(os.environ, environment), patch.object(sys, "argv", argv), \
                    patch.object(release.Config, "environment", return_value=config()), \
                    patch.object(release, "GitHub"), patch.object(release, "Cloud"), \
                    patch.object(release, "Release", return_value=flow), \
                    contextlib.redirect_stderr(io.StringIO()):
                code = release.main()
            result = json.loads(result_path.read_text()) if result_path.exists() else None
            return code, flow, result

    def test_prepare_persists_only_safe_snapshot(self):
        code, flow, result = self.invoke("prepare")
        self.assertEqual(code, 0)
        flow.prepare.assert_called_once_with(NEW)
        assert result is not None
        self.assertEqual(result["phase"], "prepared")
        self.assertNotIn("not-persisted", json.dumps(result))

    def test_prepare_failure_does_not_produce_success(self):
        code, _, result = self.invoke("prepare", failure=release.ApiError(403))
        self.assertEqual(code, 1)
        self.assertIsNone(result)

    def test_missing_snapshot_prevents_release(self):
        code, flow, _ = self.invoke("deploy", snapshot=False)
        self.assertEqual(code, 1)
        flow.execute.assert_not_called()

    def test_deploy_failure_returns_nonzero(self):
        code, flow, _ = self.invoke("deploy", failure=release.ReleaseError("rollback verified"))
        self.assertEqual(code, 1)
        flow.execute.assert_called_once()

    def test_target_initializes_failure_result_and_build_outputs(self):
        code, flow, result = self.invoke("target")
        self.assertEqual(code, 0)
        assert result is not None
        self.assertEqual(result["phase"], "awaiting_build")
        flow.execute.assert_not_called()

    def test_module_entrypoints_import_cleanly_without_external_services(self):
        root = Path(__file__).resolve().parents[2]
        for module in ("scripts.release", "scripts.smoke_test"):
            with self.subTest(module=module):
                result = subprocess.run([sys.executable, "-B", "-m", module, "--help"],
                                        cwd=root, capture_output=True, text=True, timeout=10)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn("usage:", result.stdout)

    def test_fork_and_nonpush_events_are_rejected(self):
        for repo, event in (("fork/ArmadaCMS", "push"), ("armada-ths/ArmadaCMS", "pull_request"),
                            ("armada-ths/ArmadaCMS", "workflow_dispatch")):
            with patch.dict(os.environ, {"GITHUB_REPOSITORY": repo, "GITHUB_EVENT_NAME": event}):
                with self.subTest(repo=repo, event=event), self.assertRaises(release.ReleaseError):
                    release.Config.environment()
