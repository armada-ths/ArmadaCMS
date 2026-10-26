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
import subprocess

args = sys.argv[1:]
source_option = next(arg for arg in args if arg.startswith("Dir::Etc::sourcelist="))
source_path = Path(source_option.split("=", 1)[1])
source = source_path.read_text()
archive_option = next(arg for arg in args if arg.startswith("Dir::Cache::archives="))
archives = Path(archive_option.split("=", 1)[1])
if os.environ.get("APT_CHECK_SANDBOX") == "1":
    subprocess.run(["/usr/bin/sudo", "-n", "-u", "_apt", "--", "test", "-r", str(source_path)], check=True)
    subprocess.run(["/usr/bin/sudo", "-n", "-u", "_apt", "--", "test", "-x", str(archives)], check=True)
    subprocess.run(["/usr/bin/sudo", "-n", "-u", "_apt", "--", "test", "-x", str(source_path.parent / "lists")], check=True)

mirror = "primary" if "azure.archive.ubuntu.com" in source else "fallback"
phase = "update" if "update" in args else (
    "download" if "--download-only" in args else "install"
)
with open(os.environ["APT_CALLS"], "a", encoding="utf-8") as log:
    log.write(json.dumps({"mirror": mirror, "phase": phase, "args": args, "source": str(source_path)}) + "\n")
failure = os.environ.get("APT_FAILURE", "")
if failure == "primary-update" and mirror == "primary" and phase == "update":
    sys.exit(100)
if phase == "download":
    if failure == "both-downloads" or (failure == "primary-download" and mirror == "primary"):
        sys.exit(124)
if failure == "install" and phase == "install":
    sys.exit(100)
if phase == "download":
    assert (archives / "existing.deb").read_bytes() == b"cached-package"
    (archives / "downloaded.deb").write_bytes(b"downloaded-package")
'''

@unittest.skipUnless(IS_UBUNTU, "libvips setup tests require Ubuntu")
class LibvipsSetupTests(unittest.TestCase):
    def run_setup(self, failure="", check_sandbox=False):
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
                (binaries / name).write_text(content, encoding="utf-8")
                (binaries / name).chmod(0o755)
            log = root / "calls"
            cache = root / "cache"
            cache.mkdir()
            (cache / "existing.deb").write_bytes(b"cached-package")
            environment = {
                **os.environ,
                "PATH": str(binaries) + os.pathsep + os.environ["PATH"],
                "RUNNER_TEMP": str(root),
                "VIPS_APT_CACHE": str(cache),
                "APT_CALLS": str(log),
                "APT_FAILURE": failure,
                "APT_CHECK_SANDBOX": "1" if check_sandbox else "0",
            }
            result = subprocess.run(
                ["bash", str(INSTALL)], env=environment, capture_output=True,
                text=True, check=False, timeout=15,
            )
            calls = [json.loads(line) for line in log.read_text().splitlines()]
            self.assertEqual((cache / "existing.deb").read_bytes(), b"cached-package")
            for call in calls:
                self.assertFalse(Path(call["source"]).parent.exists())
            if result.returncode == 0:
                self.assertEqual((cache / "downloaded.deb").read_bytes(), b"downloaded-package")
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

    def test_apt_sandbox_can_access_staging_with_private_runner_temp(self):
        sudo = subprocess.run(
            ["/usr/bin/sudo", "-n", "true"], capture_output=True, check=False,
        )
        if sudo.returncode != 0:
            self.skipTest("Noninteractive sudo is unavailable")
        # TemporaryDirectory defaults to 0700, like a private runner ancestor.
        result, _ = self.run_setup(check_sandbox=True)
        self.assertEqual(result.returncode, 0, result.stderr)
