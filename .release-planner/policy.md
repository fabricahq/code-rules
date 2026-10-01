# Release policy

Release Planner's agent reads this file before every release. Treat the public contract below as what users depend on, and flag uncertainty instead of inventing compatibility guarantees.

## Breaking changes

A change is breaking when it changes any of these in a way that forces users to change their configuration, rules, scripts, or workflow:

- Command names, flags, arguments, and exit codes
- JSON output from `--json`, which scripts parse
- Project and library configuration files and their keys
- The rule library format: rule, group, and library files, and the IDs other projects reference
- Library releases: change notes in `changes/`, `release/<number>` tags, and the release record in their tag messages, which other Code Rules versions read
- Vendored source records (`vendor/<source>/_source.json`), which projects commit
- Generated files, including `generated/provenance.json`
- The managed project README format (`.code-rules/README.md`); see [Choosing a version](#choosing-a-version)
- Supported platforms, and the release archive names, `SHA256SUMS`, and attestation signer that the standalone installer, the Homebrew tap, and generated library CI workflows depend on

Internal refactors, CI changes, and website-only changes are not part of the contract.

## Choosing a version

Versions follow [SemVer 2.0.0](https://semver.org/), with Git tags `vMAJOR.MINOR.PATCH`. The CLI reports the same version without the `v`. Prereleases use a suffix such as `v0.2.0-rc.1`. Release tags never carry build metadata.

Before 1.0.0:

- Minor: any breaking change or new feature. Always document breaking changes; 0.x is not permission to hide them.
- Patch: compatible fixes, documentation, and internal changes

From 1.0.0:

- Major: any breaking change
- Minor: compatible new features
- Patch: compatible fixes, documentation, and internal changes

A change to the managed project README format is always breaking: a minor release before 1.0.0, a major release from 1.0.0. The notes describe the format change, and that `code-rules project build`, `code-rules project sync`, `code-rules project update`, and `code-rules project init` replace an older, unedited guide with the one bundled in the new version but refuse to overwrite a manually edited guide.

## Who reads the release notes

Readers:

- Developers and teams who install the `code-rules` CLI to adopt rules in their projects
- Authors of rule libraries, who care about changes to the library format and authoring commands

## Order of the release notes

The notes present changes in this order, leaving out any with nothing to say:

1. New features
2. Improvements
3. Bug fixes
4. Breaking changes

## Always and never

- Always also call out a breaking change in a sentence near the start of the notes, so readers can't miss it.
- Always give migration steps for a breaking change, with the commands or before-and-after examples readers need.
- Always credit external contributors by GitHub handle.
- Never present internal refactors, CI changes, or website-only changes as new CLI features.
- Never mention dependency updates unless they fix a security issue.
