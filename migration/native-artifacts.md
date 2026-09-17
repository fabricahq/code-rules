# Native artifact review

Build unpublished archives for macOS and Linux, on amd64 and arm64:

```sh
go run ./cmd/package-native --candidate --output /tmp/code-rules-artifacts
```

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

For walkthrough PR38, place the artifact directory at `native-artifacts` beside
`rules-lab`. The lab verifies and extracts the actual host archive, runs its CLI
in a temporary project, and shows artifact metadata, commands, and output files.

## Publication is separate

These are review candidates. The current package declares UNLICENSED and has no
approved tool license. Non-candidate packaging refuses missing terms, mismatched
package versions, and dirty source. Publishing, an npm native loader, release
signing, and changing the default distribution require separate owner approval.
The existing TypeScript npm entrypoint remains intact.
