---
title: "Install Code Rules"
description: "Install Code Rules with the shell script or Homebrew."
---

Code Rules is a standalone binary for macOS and Linux.

Native Windows is not supported yet. See [Installing on Windows](#installing-on-windows) below.

Install with the [shell script](#use-the-standalone-installer), or use [Homebrew](#install-with-homebrew).

## Use the standalone installer

The installer downloads the latest stable release, verifies its SHA-256, and installs into `~/.local/bin`. It does not use `sudo` or change your shell configuration.

Run:

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh
```

You can also download and inspect the script before running it:

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh -o install.sh
sh install.sh
```

The installer needs `curl`, `tar`, and either `sha256sum` or `shasum`, along with standard shell utilities. It checks the archive and executable before replacing an existing installation. It retains the license beside the executable as `code-rules.LICENSE`.

### Set up your PATH

If the installation directory is not on your `PATH`, the installer prints the command to add it. For the default directory in sh, bash, or zsh:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Run that command to use Code Rules in the current terminal. To keep it for new terminals, add it yourself to the startup file your shell uses, such as `~/.zshrc` for zsh or `~/.bashrc` for interactive bash. For fish, run `fish_add_path "$HOME/.local/bin"`.

Then check the installed executable:

```sh
code-rules --version
code-rules --help
```

### Choose a version or installation directory

Pass installer options through `sh -s --`. Replace the example version with a version listed in [GitHub Releases](https://github.com/fabricahq/code-rules/releases).

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh -s -- --version 0.1.0
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh -s -- --install-dir "$HOME/bin"
```

You can combine both options. The installation directory must be an absolute, writable path. A version can include its `v` prefix or a prerelease suffix. Omitting `--version` selects the latest stable release.

### Upgrade, roll back, or remove

Rerun the installer to upgrade. To roll back, rerun it with `--version` set to the older published version. If you used `--install-dir`, pass the same directory again.

If the destination executable is a symlink managed by another installer, the script stops. Upgrade through that package manager, or select a different installation directory. If another `code-rules` appears earlier on your `PATH`, the installer tells you which one your shell will use.

To remove the default standalone installation:

```sh
rm "$HOME/.local/bin/code-rules" "$HOME/.local/bin/code-rules.LICENSE"
```

Use your custom installation directory if you chose one. Removing the executable does not remove rules from your projects.

## Install with Homebrew

With [Homebrew](https://brew.sh/) installed on macOS or Linux:

```sh
brew install fabricahq/tap/code-rules
code-rules --version
```

The Fabrica tap selects the release archive for your operating system and processor and verifies its checksum. Homebrew manages the installed executable and its license.

To upgrade or remove it:

```sh
brew upgrade code-rules
brew uninstall code-rules
```

## Download a release manually

Choose an archive and `SHA256SUMS` from the same [GitHub release](https://github.com/fabricahq/code-rules/releases):

| Platform | Archive |
| --- | --- |
| macOS, Apple silicon | `code-rules_<version>_darwin_arm64.tar.gz` |
| macOS, Intel | `code-rules_<version>_darwin_amd64.tar.gz` |
| Linux, ARM64 | `code-rules_<version>_linux_arm64.tar.gz` |
| Linux, Intel/AMD 64-bit | `code-rules_<version>_linux_amd64.tar.gz` |

Compute the archive's SHA-256 with `shasum -a 256 <archive>` on macOS or `sha256sum <archive>` on Linux. Compare it with that archive's entry in `SHA256SUMS` **before extracting it**. Checksums detect corruption; they do not authenticate a download if someone can replace both the archive and its checksum record.

Extract into a new directory, retain `LICENSE.md`, and run `./code-rules --version` and `./code-rules --help`. Add that directory to your `PATH`, or use the executable's absolute path. Keep separate versioned directories if you want to switch versions without downloading again.

## Installing on Windows

The Linux version is expected to work in [Windows Subsystem for Linux 2 (WSL 2)](https://learn.microsoft.com/en-us/windows/wsl/about), but we have not tested Code Rules on WSL end to end. WSL 1 is not a supported target.

Follow the Linux installation instructions in your WSL 2 terminal.

Keep your project in the [WSL Linux filesystem](https://learn.microsoft.com/en-us/windows/wsl/filesystems), such as `~/projects/my-app`, rather than `/mnt/c/`. To sync remote libraries, use Git and repository credentials configured inside WSL 2.

## After installation or an upgrade

Installing the executable does not initialize projects or fetch libraries. For a new installation, follow [Set up your first project](/start-here/set-up-project/) or [Create your first library](/start-here/create-library/).

After changing the executable version, run from an existing project:

```sh
code-rules init
code-rules build
code-rules check
```

Init refreshes the managed project README while preserving valid configuration and local rules. It refuses to overwrite manual edits to the guide. Build regenerates guidance and provenance; review those changes before committing. Use `sync` separately when you want to resolve remote library revisions again.

Use the same review process after rolling back. The `code-rules update` command is not implemented; upgrade the executable through your chosen installation method.

## Build from source

Install the Go version declared in the repository's `go.mod`, then run from the Code Rules checkout:

```sh
go build -o ./dist/code-rules ./cmd/code-rules
./dist/code-rules --version
./dist/code-rules --help
```

Use the executable's absolute path, or add its directory to your `PATH`. A source build reports a development version unless built with an explicit version.

## Build a review candidate

The repository's packaging workflow builds review archives from a pull request. Its PR comment links to downloadable GitHub Actions artifacts after a successful build. GitHub sign-in is required; artifacts expire after seven days.

To build your own candidate, commit the intended source first, then run:

```sh
go run ./cmd/package-binaries --candidate --output /tmp/code-rules-artifacts
```

Choose a new output directory. Packaging builds committed HEAD in isolation, excluding uncommitted edits. Candidate versions identify the source commit. Explicit release versions come from the approved release request; tool terms come from `LICENSE.md`.

The directory contains target-specific `.tar.gz` archives, `manifest.json`, and `SHA256SUMS`. Compare the target archive's SHA-256 against a trusted manifest before extracting it into a new directory. Inspect the manifest's source commit, target, and version, then run the extracted `./code-rules --version` and `./code-rules --help`.

Checksums detect changed bytes; replacing both an archive and its manifest defeats that comparison. Candidate packaging does not publish a release or grant a tool license.
