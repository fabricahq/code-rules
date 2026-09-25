# Releases

[Release Planner](https://release-planner.fabricahq.com) publishes Code Rules releases. Say **"let's release"** to an agent working in this repository: it follows the Releases section of [AGENTS.md](../AGENTS.md) and prepares a release PR that adds `releases/v<version>.md`. Edit the notes in the PR, then **merge it to approve publication**. The merged file becomes the GitHub release description.

Release Planner owns the procedure, the release notes style, and retries; see [Make a release](https://release-planner.fabricahq.com/start-here/release/). [.release-planner/policy.md](../.release-planner/policy.md) owns the version policy and what counts as a breaking change. [.release-planner/config.yml](../.release-planner/config.yml) holds the settings. After changing the config or upgrading Release Planner, run `release-planner install` and commit the result; never edit the generated [release workflow](../.github/workflows/release-planner.yml) by hand. [Research notes](release-style-research.md) record the release note examples that shaped the style.

This document covers what is specific to Code Rules.

## Version policy

Code Rules follows [Semantic Versioning](https://semver.org/). [.release-planner/policy.md](../.release-planner/policy.md) defines what counts as a breaking change, including changes to the managed project guide, and which version each kind of change gets.

## Release assets

On the release PR, Release Planner calls [build-release.yml](../.github/workflows/build-release.yml) with the release commit. It runs gofmt, `go vet`, staticcheck, `go test -race`, and a fresh govulncheck scan on Linux and macOS, and builds the release files with the [packager](../cmd/package-binaries/) on Linux:

- `code-rules_<version>_<os>_<arch>.tar.gz` for `darwin` and `linux` on `amd64` and `arm64`, each with the executable, `README.txt`, and the [MIT license](../LICENSE.md)
- `manifest.json`, which records the version, source commit, and each archive's checksum
- `SHA256SUMS`

The [standalone installer](../_distribution/README.md) and the Homebrew formula download these files by name, so keep the names stable. The build is reproducible: rebuilding the same commit produces identical files. The workflow runs with read-only permissions and no secrets. Release Planner attests each file's build provenance on the PR, and merging publishes exactly the files the PR built. After publishing, it attests the published files again from `main`; the Homebrew tap and the [manual install instructions](../docs/src/content/docs/start-here/install.md) require that attestation, so only approved, published files pass.

## Homebrew updates

After a stable release publishes, Release Planner's `downstream` job runs **Update Code Rules** (`update-code-rules.yml`) in [fabricahq/homebrew-tap](https://github.com/fabricahq/homebrew-tap) with the release's `tag` and `version`. Prereleases do not update the formula. The tap verifies that `SHA256SUMS` was attested by `release-planner.yml` on `main`, and validates the published archives and checksums, before committing an update directly to `main`. Routine formula updates require no pull request or human approval; changes to updater code and workflows go through review in the tap repository.

The job authenticates through **Fabrica Homebrew Releaser**, a shared trigger App for trusted Fabrica products. It is installed only on the tap with Actions write and Metadata read permissions. Its credentials live in this repository's `downstream` environment, which allows only the `main` branch: the `DOWNSTREAM_APP_CLIENT_ID` variable and the `DOWNSTREAM_APP_PRIVATE_KEY` secret. The job checks out no source and creates a short-lived token restricted to the tap. [CR-7](https://linear.app/ohmygoshjosh/issue/CR-7/replace-shared-homebrew-trigger-keys-with-an-oidc-dispatch-service) tracks replacing shared-key access with an OIDC dispatch service for Fabrica tools.

The tap uses a separate publishing App whose key stays in its protected environment. Product repositories never receive that key.

If the dispatch fails, use **Re-run failed jobs** on the Release run. If the tap update itself fails, fix the reported problem and run the tap's **Update Code Rules** workflow manually. A tap failure does not modify or unpublish the release.

## Repository protection

Configure the `release` and `downstream` environments to allow only the `main` branch, with no reviewers or wait timers. Require pull requests for changes to `main` and protect its history from deletion and force pushes. Enable immutable releases so published assets and tags cannot be replaced. If administrators need bypass access, set their ruleset bypass mode to **For pull requests only**, never **Always allow**. They can then bypass review requirements through a PR, but cannot push directly.

Follow [security practices](security-practices.md) for credentials, dependency updates, and review protections. Resolve repository permission restrictions before releasing; do not work around them with a personal token. These controls authenticate the release workflow and preserve published artifacts; they cannot detect malicious code approved into that workflow.

## Testing PR preview builds

After a successful build, PRs opened by a maintainer from a branch in this repository automatically receive preview links. Fork PRs and other contributors need a maintainer to approve the exact commit before the bot posts links.

**Warning: These executables run code from the PR. Use a disposable test environment without credentials or private files. Even `--help` executes the program.**

The PR comment offers direct executable downloads for macOS (Apple Silicon or Intel) and Linux (ARM or Intel/AMD). Download the file for your computer and rename it to `code-rules`, then run:

```sh
chmod +x code-rules
./code-rules --help
```

Each preview command prints this warning to stderr before command output, including help, version, and JSON commands:

```text
WARNING: Unreleased preview from commit <full SHA>. For testing only; not for production use.
```

JSON output on stdout remains unchanged. The warning is a reminder, not proof of authenticity: someone modifying the binary could remove it. Preview binaries are not publisher-signed or attested by Code Rules.

No extraction or installation is needed. These previews have not been released. The comment also links to the build results and license. GitHub sign-in is required; downloads expire after seven days, regardless of whether the PR is open, closed, or merged.

## Candidate archives

For unpublished review builds, commit intended source changes first:

```sh
go run ./cmd/package-binaries --candidate --output /tmp/code-rules-artifacts
```

Use a new output directory for each attempt. Verify archive checksums against a trusted manifest before extraction. Describe these builds as unpublished review candidates, and claim platform compatibility only where testing supports it.

Read the [packaging command](../cmd/package-binaries/) for options and the [PR packaging workflow](../.github/workflows/package-binaries.yml) for downloadable CI builds.
