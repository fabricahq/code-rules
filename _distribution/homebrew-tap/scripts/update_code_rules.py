#!/usr/bin/env python3
"""Generate a Homebrew formula from the latest complete, stable GitHub release.

Run in the tap checkout. This writes a formula but does not commit or push it.
"""

import json
import re
import sys
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import urlopen

REPOSITORY = "https://github.com/fabricahq/code-rules"
LATEST_RELEASE = "https://api.github.com/repos/fabricahq/code-rules/releases/latest"
TARGETS = ("darwin_arm64", "darwin_amd64", "linux_arm64", "linux_amd64")


def read_url(url):
    """Read bounded public metadata over HTTPS; never download executable code."""
    with urlopen(url, timeout=30) as response:
        if not response.url.startswith("https://"):
            raise ValueError("release metadata redirected away from HTTPS")
        data = response.read(1024 * 1024 + 1)
    if len(data) > 1024 * 1024:
        raise ValueError("release metadata exceeds 1 MiB")
    return data


def stable_version(tag):
    match = re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", tag)
    if not match:
        raise ValueError("Homebrew requires a stable vMAJOR.MINOR.PATCH release")
    return tuple(map(int, match.groups()))


def render_formula(release, checksums):
    """Return a formula only when all four published archives have valid checksums."""
    tag = release["tag_name"]
    stable_version(tag)
    if release["draft"] or release["prerelease"] or not release.get("published_at"):
        raise ValueError("release must be published and stable")
    version = tag[1:]
    sums = {}
    for line in checksums.splitlines():
        match = re.fullmatch(r"([0-9a-f]{64})  (\S+)", line)
        if not match or match[2] in sums:
            raise ValueError("invalid or duplicate release checksum")
        sums[match[2]] = match[1]
    assets = {}
    for asset in release["assets"]:
        if asset["name"] in assets:
            raise ValueError("duplicate release asset")
        assets[asset["name"]] = asset
    lines = [
        "# Generated from a published Code Rules release; update with scripts/update_code_rules.py.",
        "class CodeRules < Formula",
        '  desc "Version and share engineering rules for coding agents"',
        '  homepage "https://code-rules.fabricahq.com"',
        f'  version "{version}"',
        '  license "MIT"',
        "",
    ]
    for os, targets in (("macos", TARGETS[:2]), ("linux", TARGETS[2:])):
        lines.append(f"  on_{os} do")
        for target in targets:
            filename = f"code-rules_{version}_{target}.tar.gz"
            asset = assets.get(filename, {})
            digest = sums.get(filename)
            url = f"{REPOSITORY}/releases/download/{tag}/{filename}"
            if not digest or asset.get("state") != "uploaded" or asset.get("size", 0) <= 0 or asset.get("browser_download_url") != url:
                raise ValueError(f"missing or invalid published archive: {filename}")
            if asset.get("digest") and asset["digest"] != f"sha256:{digest}":
                raise ValueError(f"GitHub asset digest disagrees with SHA256SUMS: {filename}")
            arch = "arm" if target.endswith("arm64") else "intel"
            lines += [f"    on_{arch} do", f'      url "{url}"', f'      sha256 "{digest}"', "    end"]
        lines += ["  end", ""]
    lines += [
        "  def install",
        '    bin.install "code-rules"',
        '    prefix.install "LICENSE.md"',
        "  end",
        "",
        "  test do",
        '    assert_match version.to_s, shell_output("#{bin}/code-rules --version")',
        '    system bin/"code-rules", "--help"',
        "  end",
        "end",
        "",
    ]
    return "\n".join(lines)


def update_formula(path, release, checksums):
    """Write a newer formula, reject downgrades or changed assets at the same version."""
    formula = render_formula(release, checksums)
    if path.exists():
        current = path.read_text()
        if current == formula:
            return False
        match = re.search(r'^  version "([^"]+)"$', current, re.MULTILINE)
        if not match:
            raise ValueError("cannot identify the current formula version")
        if stable_version(release["tag_name"]) <= stable_version("v" + match[1]):
            raise ValueError("refusing to replace the same or a newer formula version")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(formula)
    return True


def main():
    try:
        release = json.loads(read_url(LATEST_RELEASE))
    except HTTPError as error:
        if error.code == 404:
            print("No published stable release yet; leaving the tap unchanged.")
            return
        raise
    tag = release["tag_name"]
    stable_version(tag)
    checksums = read_url(f"{REPOSITORY}/releases/download/{tag}/SHA256SUMS").decode("ascii")
    changed = update_formula(Path("Formula/code-rules.rb"), release, checksums)
    print(f"{'Prepared' if changed else 'Already current:'} Code Rules {tag}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError) as error:
        sys.exit(f"Homebrew update failed: {error}")
