"""Exercise the piped installer with real shell, tar, checksums, and isolated files.

Only network downloads and platform detection are replaced with fixtures.
"""

import hashlib
import io
import os
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
INSTALLER = ROOT / "docs/public/install.sh"


class InstallerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="code-rules-installer-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.home = self.root / "home"
        self.home.mkdir()
        self.bin = self.root / "commands"
        self.bin.mkdir()
        self.fixtures = self.root / "releases"
        self.fixtures.mkdir()
        self.destination = self.home / ".local/bin"
        self.env = dict(os.environ, HOME=str(self.home), FIXTURES=str(self.fixtures),
                        PATH=f"{self.bin}:{os.environ['PATH']}", OS="Linux", ARCH="x86_64")
        self.command("uname", '#!/bin/sh\ncase "$1" in -s) echo "$OS";; -m) echo "$ARCH";; esac\n')
        self.command("curl", f"#!{sys.executable}\n" + '''import os, pathlib, shutil, sys
args = sys.argv[1:]
assert args[args.index('--proto') + 1] == '=https'
assert args[args.index('--proto-redir') + 1] == '=https'
assert '--tlsv1.2' in args
url = args[-1]
assert url.startswith('https://github.com/fabricahq/code-rules/releases/')
if url.endswith('/latest'):
    print('https://github.com/fabricahq/code-rules/releases/tag/v1.2.3', end='')
else:
    source = pathlib.Path(os.environ['FIXTURES']) / url.split('/download/')[1]
    if not source.is_file():
        sys.exit(22)
    shutil.copyfile(source, args[args.index('--output') + 1])
''')
        self.binary = b'#!/bin/sh\nprintf "code-rules 1.2.3\\n"\n'
        self.release()

    def command(self, name, text):
        path = self.bin / name
        path.write_text(text)
        path.chmod(0o755)

    def release(self, version="1.2.3", target="linux_amd64", entries=None):
        directory = self.fixtures / f"v{version}"
        directory.mkdir(exist_ok=True)
        archive = directory / f"code-rules_{version}_{target}.tar.gz"
        entries = entries if entries is not None else [("code-rules", self.binary), ("LICENSE.md", b"MIT license"), ("README.txt", b"Code Rules")]
        with tarfile.open(archive, "w:gz", format=tarfile.USTAR_FORMAT) as tar:
            for name, content in entries:
                info = tarfile.TarInfo(name)
                if content is None:
                    info.type = tarfile.SYMTYPE
                    info.linkname = "/tmp/should-not-be-touched"
                    tar.addfile(info)
                else:
                    info.size = len(content)
                    info.mode = 0o755 if name == "code-rules" else 0o644
                    tar.addfile(info, io.BytesIO(content))
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        (directory / "SHA256SUMS").write_text(f"{digest}  {archive.name}\n")
        return archive

    def run_installer(self, *args):
        return subprocess.run(["sh", "-s", "--", *args], input=INSTALLER.read_bytes(),
                              env=self.env, capture_output=True, timeout=30)

    def assert_success(self, result):
        self.assertEqual(result.returncode, 0, (result.stdout + result.stderr).decode())

    def test_piped_latest_installs_and_prints_path_without_editing_profiles(self):
        profile = self.home / ".zshrc"
        profile.write_text("# keep my configuration\n")
        result = self.run_installer()
        self.assert_success(result)
        self.assertEqual((self.destination / "code-rules").read_bytes(), self.binary)
        self.assertTrue(os.access(self.destination / "code-rules", os.X_OK))
        self.assertEqual((self.destination / "code-rules.LICENSE").read_text(), "MIT license")
        self.assertIn(b"export PATH=", result.stdout)
        self.assertEqual(profile.read_text(), "# keep my configuration\n")
        self.assertEqual(list(self.destination.glob(".code-rules-install.*")), [])

    @unittest.skipUnless(os.environ.get("CODE_RULES_TEST_BINARY"), "set CODE_RULES_TEST_BINARY for the native executable smoke test")
    def test_native_executable_installs_and_runs(self):
        (self.bin / "uname").unlink()
        os_name = subprocess.check_output(["uname", "-s"], text=True).strip()
        arch = subprocess.check_output(["uname", "-m"], text=True).strip()
        target = ("darwin" if os_name == "Darwin" else "linux") + "_" + ("arm64" if arch in ("arm64", "aarch64") else "amd64")
        self.binary = Path(os.environ["CODE_RULES_TEST_BINARY"]).read_bytes()
        self.release(target=target)
        self.assert_success(self.run_installer())
        result = subprocess.run([str(self.destination / "code-rules"), "--help"], capture_output=True)
        self.assert_success(result)
        self.assertIn(b"sync", result.stdout)

    def test_all_four_platforms(self):
        for os_name, machine, target in [("Darwin", "arm64", "darwin_arm64"), ("Darwin", "x86_64", "darwin_amd64"), ("Linux", "aarch64", "linux_arm64"), ("Linux", "amd64", "linux_amd64")]:
            with self.subTest(target=target):
                self.env.update(OS=os_name, ARCH=machine)
                self.release(target=target)
                self.assert_success(self.run_installer("--version", "v1.2.3"))

    def test_upgrade_and_explicit_prerelease_in_custom_directory(self):
        destination = self.root / "custom bin"
        self.assert_success(self.run_installer("--install-dir", str(destination)))
        self.binary = b'#!/bin/sh\necho "code-rules 1.3.0-rc.1"\n'
        self.release(version="1.3.0-rc.1")
        self.assert_success(self.run_installer("--version", "1.3.0-rc.1", "--install-dir", str(destination)))
        self.assertEqual((destination / "code-rules").read_bytes(), self.binary)

    def test_failed_download_or_checksum_keeps_installed_binary(self):
        self.assert_success(self.run_installer())
        archive = self.release()
        for failure in ("missing", "corrupt", "duplicate-checksum"):
            with self.subTest(failure=failure):
                archive = self.release()
                if failure == "missing":
                    archive.unlink()
                elif failure == "corrupt":
                    archive.write_bytes(b"corrupted download")
                else:
                    sums = archive.parent / "SHA256SUMS"
                    sums.write_text(sums.read_text() * 2)
                result = self.run_installer()
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual((self.destination / "code-rules").read_bytes(), self.binary)

    def test_rejects_unsafe_incomplete_and_duplicate_archive_members(self):
        for entries in [
            [("../escape", b"bad"), ("code-rules", self.binary), ("LICENSE.md", b"MIT")],
            [("code-rules", None), ("LICENSE.md", b"MIT")],
            [("code-rules", self.binary)],
            [("code-rules", self.binary), ("code-rules", self.binary), ("LICENSE.md", b"MIT")],
            [("code-rules", b""), ("LICENSE.md", b"MIT")],
        ]:
            with self.subTest(entries=entries):
                self.release(entries=entries)
                result = self.run_installer()
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse((self.destination / "code-rules").exists())

    def test_unrunnable_download_preserves_existing_binary(self):
        self.assert_success(self.run_installer())
        self.release(entries=[("code-rules", b"#!/bin/sh\nexit 1\n"), ("LICENSE.md", b"MIT")])
        self.assertNotEqual(self.run_installer().returncode, 0)
        self.assertEqual((self.destination / "code-rules").read_bytes(), self.binary)

    def test_refuses_to_replace_symlink_or_directory(self):
        self.destination.mkdir(parents=True)
        destination = self.destination / "code-rules"
        target = self.root / "other-manager-binary"
        target.write_text("preserve")
        destination.symlink_to(target)
        self.assertNotEqual(self.run_installer().returncode, 0)
        self.assertEqual(target.read_text(), "preserve")
        destination.unlink()
        destination.mkdir()
        self.assertNotEqual(self.run_installer().returncode, 0)
        self.assertTrue(destination.is_dir())

    def test_rejects_unsupported_platform_and_invalid_options(self):
        for args in [("--version", "../../bad"), ("--version",), ("--unknown",), ("--install-dir", "relative"), ("--install-dir", "/tmp/a:b")]:
            with self.subTest(args=args):
                self.assertNotEqual(self.run_installer(*args).returncode, 0)
        for os_name, arch in [("FreeBSD", "x86_64"), ("Linux", "riscv64")]:
            self.env.update(OS=os_name, ARCH=arch)
            self.assertNotEqual(self.run_installer().returncode, 0)
        self.assertFalse(self.destination.exists())

    def test_printed_path_command_safely_quotes_custom_directory(self):
        directory = self.root / "it's a $(touch unwanted) bin"
        result = self.run_installer("--install-dir", str(directory))
        self.assert_success(result)
        line = next(line for line in result.stdout.decode().splitlines() if line.startswith("export PATH="))
        checked = subprocess.run(["sh", "-c", line + '\ncommand -v code-rules'], env=self.env, capture_output=True, text=True)
        self.assertEqual(checked.returncode, 0, checked.stderr)
        self.assertEqual(checked.stdout.strip(), str(directory / "code-rules"))
        self.assertFalse((ROOT / "unwanted").exists())

    def test_help_needs_no_download(self):
        shutil.rmtree(self.fixtures)
        self.assert_success(self.run_installer("--help"))
        self.assertFalse(self.destination.exists())


if __name__ == "__main__":
    unittest.main()
