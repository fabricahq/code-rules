"""Test formula generation and update refusals using published-release metadata fixtures."""

import copy
import importlib.util
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "homebrew-tap/scripts/update_code_rules.py"
spec = importlib.util.spec_from_file_location("update_code_rules", SCRIPT)
updater = importlib.util.module_from_spec(spec)
spec.loader.exec_module(updater)


def release_fixture(version="1.2.3"):
    release = {"tag_name": f"v{version}", "draft": False, "prerelease": False,
               "published_at": "2026-09-18T00:00:00Z", "assets": []}
    sums = []
    for target in updater.TARGETS:
        filename = f"code-rules_{version}_{target}.tar.gz"
        release["assets"].append({"name": filename, "size": 1234, "state": "uploaded",
                                  "digest": "sha256:" + "a" * 64,
                                  "browser_download_url": f"{updater.REPOSITORY}/releases/download/v{version}/{filename}"})
        sums.append(f"{'a' * 64}  {filename}")
    return release, "\n".join(sums)


class HomebrewTest(unittest.TestCase):
    def test_formula_has_each_platform_checksum_and_license(self):
        formula = updater.render_formula(*release_fixture())
        for target in updater.TARGETS:
            self.assertIn(f"code-rules_1.2.3_{target}.tar.gz", formula)
        self.assertEqual(formula.count('sha256 "' + "a" * 64 + '"'), 4)
        self.assertIn('prefix.install "LICENSE.md"', formula)
        self.assertIn('system bin/"code-rules", "--help"', formula)

    def test_incomplete_or_unpublished_releases_are_rejected(self):
        release, sums = release_fixture()
        variants = []
        for field, value in [("draft", True), ("prerelease", True), ("published_at", None), ("tag_name", "v1.2.3-rc.1"), ("tag_name", 'v1.2.3"; system("bad")')]:
            variant = copy.deepcopy(release)
            variant[field] = value
            variants.append(variant)
        missing = copy.deepcopy(release)
        missing["assets"].pop()
        variants.append(missing)
        corrupt = copy.deepcopy(release)
        corrupt["assets"][0]["digest"] = "sha256:" + "b" * 64
        variants.append(corrupt)
        for variant in variants:
            with self.subTest(variant=variant):
                with self.assertRaises(ValueError):
                    updater.render_formula(variant, sums)
        for invalid in ["", sums.splitlines()[0], sums + "\n" + sums, sums.replace("a" * 64, "invalid")]:
            with self.assertRaises(ValueError):
                updater.render_formula(release, invalid)

    def test_updates_are_idempotent_and_never_downgrade(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "Formula/code-rules.rb"
            self.assertTrue(updater.update_formula(path, *release_fixture()))
            self.assertFalse(updater.update_formula(path, *release_fixture()))
            old = path.read_bytes()
            with self.assertRaises(ValueError):
                updater.update_formula(path, *release_fixture("1.2.2"))
            changed, sums = release_fixture()
            for asset in changed["assets"]:
                asset["digest"] = "sha256:" + "b" * 64
            with self.assertRaises(ValueError):
                updater.update_formula(path, changed, sums.replace("a" * 64, "b" * 64))
            self.assertEqual(path.read_bytes(), old)
            self.assertTrue(updater.update_formula(path, *release_fixture("1.3.0")))


if __name__ == "__main__":
    unittest.main()
