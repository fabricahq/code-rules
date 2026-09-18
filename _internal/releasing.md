# Packaging and releases

Build unpublished archives for macOS and Linux, on amd64 and arm64:

```sh
go run ./cmd/package-binaries --candidate --output /tmp/code-rules-artifacts
```

The build compiles an isolated copy of committed HEAD. Commit intended source changes first; uncommitted edits are excluded even from candidates. The manifest reports whether the original checkout had uncommitted edits.

The output directory must be new. It contains a manifest, SHA256SUMS, and one
`.tar.gz` per target. Each archive contains the native executable, README.txt,
and the tool's original LICENSE.md if present. The manifest records the version,
source commit, dirty state, Go toolchain, target, exact sizes, and SHA-256 hashes.
A failed build retains an INCOMPLETE marker; retry into a new directory.

For a trusted archive, verify its SHA-256 against the trusted manifest before
extracting into a new versioned directory. Run `./code-rules --version` and
`./code-rules --help`. Offline commands need neither Node nor Bun. Sync needs Git.
Keep the previous directory until the new version passes your project checks;
rollback selects the previous executable. Existing installs are never replaced
by the internal verification helper. Checksums detect corruption, not substitution
of both the archive and manifest.

The packaging tests execute extracted host binaries with an empty PATH, exercise
local authoring/build/check, install two versions, and verify rollback and corrupt
archive refusal. CI runs these tests on Linux and macOS and cross-compiles all
four targets. Cross-compilation alone does not establish runtime compatibility
on an architecture that CI did not execute.

## Publication is separate

Version and license metadata come from committed `release.json`.
These are review candidates. The current metadata declares UNLICENSED and has no
declared tool license. Non-candidate packaging checks declared terms as a prerequisite; it cannot establish owner approval and never publishes. It refuses missing terms, mismatched
release versions, and dirty source. Publishing and release signing require separate owner approval. Packaging never publishes or replaces an installed binary.

## Pull request downloads

`package-binaries.yml` builds and uploads the candidate bundle for seven days. After a successful pull-request run, `comment-binary-preview.yml` posts or updates one bot comment with a direct download link, the PR commit, expiry, and build results. Reviewers must sign in to GitHub to download artifacts. Pull-request builds check out the exact PR head, so the manifest and preview comment identify the same source commit. Manual runs use the selected workflow revision.

The notification uses GitHub API metadata only. It never checks out PR code, downloads artifacts, or executes their contents. Failed builds, missing or expired artifacts, closed PRs, and superseded commits do not produce a preview comment. A manual packaging run uploads artifacts without commenting on a PR.

GitHub activates `workflow_run` notifications only after the notification workflow reaches the repository's default branch (`main`). Feature-branch-only workflow changes do not activate automatic comments.
