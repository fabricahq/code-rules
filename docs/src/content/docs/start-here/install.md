---
title: "Install Code Rules"
description: "Install Code Rules with the shell script or Homebrew."
---

Code Rules is a standalone binary for macOS and Linux. [Using Windows?](#installing-on-windows)

## Use the standalone installer

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh
```

This installs the latest stable release into `~/.local/bin` and sets up your shell's `PATH`. No `sudo` needed. [View the installer script](/install.sh).

<span id="set-up-your-path"></span>

Open a new terminal, then verify the installation:

```sh
code-rules --version
```

To use your current terminal, run the command the installer prints instead.

**Next:** [Set up your first project](/start-here/set-up-project/) or [create your first library](/start-here/create-library/).

## Upgrade

Rerun the installer. Then, from each existing project root, refresh and check its generated files:

```sh
code-rules project build
code-rules project check
```

## Install with Homebrew

With [Homebrew](https://brew.sh/) installed on macOS or Linux:

```sh
brew install fabricahq/tap/code-rules
code-rules --version
```

To upgrade or remove it:

```sh
brew upgrade code-rules
brew uninstall code-rules
```

## Other options

<details id="choose-a-version-or-installation-directory">
<summary>Choose a version or installation directory</summary>

Add `--version` to install an older or specific [release](https://github.com/fabricahq/code-rules/releases), or `--install-dir` to choose an absolute, writable directory:

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh -s -- --version 0.1.0 --install-dir "$HOME/bin"
```

Use the same custom directory when upgrading. Pass `--no-update-path` if you prefer to manage your shell configuration yourself.

</details>

<details id="download-a-release-manually">
<summary>Download a release manually</summary>

Download the archive for your OS and processor, plus `SHA256SUMS`, from [GitHub Releases](https://github.com/fabricahq/code-rules/releases). If you have the GitHub CLI, first verify the checksum manifest came from our release workflow:

```sh
gh attestation verify SHA256SUMS --repo fabricahq/code-rules \
  --signer-workflow fabricahq/code-rules/.github/workflows/release.yml \
  --source-ref refs/heads/main
```

Stop if verification fails. Before extracting, compare the archive checksum with the matching entry using `shasum -a 256 <archive>` on macOS or `sha256sum <archive>` on Linux.

Extract the archive and move the executable into a directory on your `PATH`.

To build from source instead, follow the [contributing guide](https://github.com/fabricahq/code-rules/blob/main/CONTRIBUTING.md#build-the-cli).

</details>

<details id="installing-on-windows">
<summary>Installing on Windows</summary>

Native Windows is not supported yet. The Linux version is expected to work in [WSL 2](https://learn.microsoft.com/en-us/windows/wsl/about), but we have not tested it end to end. Run the installer above in your WSL 2 terminal.

</details>

<details id="upgrade-roll-back-or-remove">
<summary>Uninstall</summary>

```sh
rm "$HOME/.local/bin/code-rules"
```

Adjust the path if you chose a custom installation directory. Your project's rules are left in place.

</details>

For version compatibility and managed guide changes, see the [release version policy](https://github.com/fabricahq/code-rules/blob/main/_engineering/releasing.md#version-policy).
