"""Exercise mirror failures without networking or modifying the host's APT state."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
INSTALL = ROOT / ".github/actions/setup-libvips/install.sh"
IS_UBUNTU = os.name == "posix" and Path("/etc/os-release").exists() and (
    "ID=ubuntu" in Path("/etc/os-release").read_text(encoding="utf-8")
)

FAKE_APT = r'''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys

args = sys.argv[1:]
source_option = next(arg for arg in args if arg.startswith("Dir::Etc::sourcelist="))
source = Path(source_option.split("=", 1)[1]).read_text()
mirror = "primary" if "azure.archive.ubuntu.com" in source else "fallback"
phase = "update" if "update" in args else (
    "download" if "--download-only" in args else "install"
)
with open(os.environ["APT_CALLS"], "a", encoding="utf-8") as log:
    log.write(json.dumps({"mirror": mirror, "phase": phase, "args": args}) + "\n")
failure = os.environ.get("APT_FAILURE", "")
if failure == "primary-update" and mirror == "primary" and phase == "update":
    sys.exit(100)
if phase == "download":
    if failure == "both-downloads" or (failure == "primary-download" and mirror == "primary"):
        sys.exit(124)
if failure == "install" and phase == "install":
    sys.exit(100)
'''

@unittest.skipUnless(IS_UBUNTU, "libvips setup tests require Ubuntu")
class LibvipsSetupTests(unittest.TestCase):
    def run_setup(self, failure=""):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binaries = root / "bin"
            binaries.mkdir()
            commands = {
                "apt-get": FAKE_APT,
                "sudo": '#!/bin/sh\nexec "$@"\n',
                "dpkg": "#!/bin/sh\necho amd64\n",
                "pkg-config": "#!/bin/sh\necho 8.15.1\n",
            }
            for name, content in commands.items():
                command = binaries / name
                command.write_text(content, encoding="utf-8")
                command.chmod(0o755)
            log = root / "calls"
            cache = root / "cache"
            cache.mkdir()
            cached_package = cache / "existing.deb"
            cached_package.write_bytes(b"cached-package")
            environment = {
                **os.environ,
                "PATH": str(binaries) + os.pathsep + os.environ["PATH"],
                "RUNNER_TEMP": str(root),
                "VIPS_APT_CACHE": str(cache),
                "APT_CALLS": str(log),
                "APT_FAILURE": failure,
            }
            result = subprocess.run(
                ["bash", str(INSTALL)], env=environment, capture_output=True,
                text=True, check=False, timeout=15,
            )
            calls = [json.loads(line) for line in log.read_text().splitlines()]
            self.assertEqual(cached_package.read_bytes(), b"cached-package")
            self.assertFalse(list(root.glob("libvips-apt.*")))
            return result, calls

    def test_primary_success_installs_offline_without_fallback(self):
        result, calls = self.run_setup()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([call["phase"] for call in calls], ["update", "download", "install"])
        self.assertTrue(all(call["mirror"] == "primary" for call in calls))
        self.assertIn("--no-download", calls[-1]["args"])
        self.assertIn("libvips 8.15.1 installed", result.stdout)

    def test_update_failure_switches_mirror(self):
        result, calls = self.run_setup("primary-update")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([(call["mirror"], call["phase"]) for call in calls], [
            ("primary", "update"), ("fallback", "update"),
            ("fallback", "download"), ("fallback", "install"),
        ])

    def test_download_timeout_switches_mirror(self):
        result, calls = self.run_setup("primary-download")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(calls[-1]["mirror"], "fallback")
        self.assertIn("::warning::", result.stdout)

    def test_both_download_failures_stop_before_install(self):
        result, calls = self.run_setup("both-downloads")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(call["phase"] == "install" for call in calls))
        self.assertIn("::error::", result.stdout)

    def test_install_failure_is_not_retried_on_another_mirror(self):
        result, calls = self.run_setup("install")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(calls), 3)
        self.assertTrue(all(call["mirror"] == "primary" for call in calls))
