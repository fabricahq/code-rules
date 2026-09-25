# Release policy

Release Planner's agent reads this file before every release. Treat the public contract below as what users depend on, and flag uncertainty instead of inventing compatibility guarantees.

## Breaking changes

A change is breaking when it changes any of these in a way that forces users to change their configuration, rules, scripts, or workflow:

- Command names, flags, arguments, and exit codes
- JSON output from `--json`, which scripts parse
- Project and library configuration files and their keys
- The rule library format: rule, group, and library files, and the IDs other projects reference
- Generated files, including `generated/provenance.json`
- The managed project README format (`.code-rules/README.md`); see [Choosing a version](#choosing-a-version)
- Supported platforms, and the release archive names that the standalone installer and Homebrew formula download

Internal refactors, CI changes, and website-only changes are not part of the contract.

## Choosing a version

Versions follow [SemVer 2.0.0](https://semver.org/), with Git tags `vMAJOR.MINOR.PATCH`. The CLI reports the same version without the `v`. Prereleases use a suffix such as `v0.2.0-rc.1`. Release tags never carry build metadata. The first release is v0.1.0.

Before 1.0.0:

- Minor: any breaking change or new feature. Always document breaking changes; 0.x is not permission to hide them.
- Patch: compatible fixes, documentation, and internal changes

From 1.0.0:

- Major: any breaking change
- Minor: compatible new features
- Patch: compatible fixes, documentation, and internal changes

A change to the managed project README format is always breaking: a minor release before 1.0.0, a major release from 1.0.0. The notes describe the format change, and that `code-rules project build` and `project sync` replace an older, unedited guide with the one bundled in the new version but refuse to overwrite a manually edited guide.

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
- For the first release, describe the product as it is, not the sequence of internal commits that built it.
- Never present internal refactors, CI changes, or website-only changes as new CLI features.
- Never mention dependency updates unless they fix a security issue.
