---
title: "Install Code Rules"
## Build a review candidate

The repository's packaging workflow builds review archives from a pull request. Its PR comment links to downloadable GitHub Actions artifacts after a successful build. GitHub sign-in is required; artifacts expire after seven days.

To build your own candidate, commit the intended source first, then run:

```sh
go run ./cmd/package-binaries --candidate --output /tmp/code-rules-artifacts
```

Choose a new output directory. Packaging builds committed HEAD in isolation, excluding uncommitted edits. Candidate versions identify the source commit. Explicit release versions come from the approved release request; tool terms come from `LICENSE.md`.

The directory contains target-specific `.tar.gz` archives, `manifest.json`, and `SHA256SUMS`. Compare the target archive's SHA-256 against a trusted manifest before extracting it into a new directory. Inspect the manifest's source commit, target, and version, then run the extracted `./code-rules --version` and `./code-rules --help`.

Checksums detect changed bytes; replacing both an archive and its manifest defeats that comparison. Candidate packaging does not publish a release or grant a tool license.
