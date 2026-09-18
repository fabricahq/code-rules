---
title: "Install Code Rules"
description: "Build the Go CLI or try a binary candidate, then upgrade or roll back deliberately."
---

Code Rules is a standalone Go executable for macOS and Linux on amd64 and arm64. Running it requires neither Node.js nor Bun. Sync also needs Git and your existing repository credentials. Windows is not supported.

Public release publication remains separate work. The checkout supports source builds and unpublished candidate archives.

## Build from source

Install the Go version declared in the repository's `go.mod`, then run from the Code Rules checkout:

```sh
go build -o ./dist/code-rules ./cmd/code-rules
./dist/code-rules --version
./dist/code-rules --help
```

Use the executable's absolute path, or add its directory to your `PATH`. A source build reports a development version unless built with an explicit version. Installing the executable does not initialize projects or fetch libraries.

Follow [Set up a project](/guides/set-up-project/) or [Create a rule library](/guides/create-library/).

## Try an unpublished release candidate

The repository's packaging workflow builds candidate archives for review. Its PR comment links to downloadable GitHub Actions artifacts after a successful build. GitHub sign-in is required; artifacts expire after seven days.

To build your own candidate, commit the intended source first, then run:

```sh
go run ./cmd/package-binaries --candidate --output /tmp/code-rules-artifacts
```

Choose a new output directory. Packaging builds committed HEAD in isolation, excluding uncommitted edits. Candidate versions identify the source commit. Explicit release versions come from the approved release request; tool terms come from `LICENSE.md`.

The directory contains target-specific `.tar.gz` archives, `manifest.json`, and `SHA256SUMS`. Compare the target archive's SHA-256 against a trusted manifest before extracting it into a new directory. On macOS, use `shasum -a 256`; on Linux, use `sha256sum`. Inspect the manifest's source commit, target, and version, then run the extracted `./code-rules --version` and `./code-rules --help`.

Checksums detect changed bytes; replacing both an archive and its manifest defeats that comparison. Candidate packaging does not publish a release or grant a tool license.

## Upgrade or roll back

Keep executables in separate versioned directories. Test the new executable against a copy of your project before selecting it on `PATH`; retain the old directory for rollback.

After selecting a new version, run from your project:

```sh
code-rules init
code-rules build
code-rules check
```

Init refreshes the managed project README while preserving valid configuration and local rules. It refuses to overwrite manual edits to the guide. Build regenerates guidance and provenance; review those changes before committing. Use `sync` separately when you want to resolve remote library revisions again.

To roll back, select the previous executable and review regeneration with that version. The `code-rules update` command is not implemented.
