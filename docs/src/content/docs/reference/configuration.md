---
title: "Configuration"
description: "Fields in .code-rules/config.yaml."
---

The **project configuration**, `.code-rules/config.yaml`, records the project's sources, selected groups, and exceptions. The **Code Rules directory**, `.code-rules/`, also holds `local/`, `vendor/`, and `generated/`.

Initialize from the project root. In Git repositories, other project commands find the nearest Git root and use its `.code-rules/config.yaml`. Outside Git, run commands from the project root. See [Working directories](/reference/cli/#working-directories).
A project can import rules directly from multiple canonical libraries, pinning each one independently.
Sync, build, and check validate the fields below.

Use one YAML document. Duplicate keys, anchors, aliases, and explicit tags are rejected. Quote wildcard selectors, such as `groups: "*"`.

## Complete example

```yaml
schemaVersion: 1
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    groups:
      - techs/typescript
      - practices/testing
    exclude: {}
    replace:
      techs/typescript/prefer-type-aliases:
        file: local/techs/typescript/prefer-interfaces.md
        reason: Our public extension API relies on declaration merging.
  acme:
    repository: https://github.com/acme/.code-rules.git
    ref: <full Git commit SHA>
    groups:
      - techs/react
      - practices/testing
      - practices/observability
    exclude: {}
    replace: {}
```

Repository names, groups, and rule IDs in examples are illustrative.
Replace them with libraries and rules your project can access.

## Fields

| Field | Meaning |
| --- | --- |
| `schemaVersion` | Configuration format version; the builder accepts `1`. |
| `sources` | Map of stable source names to library configurations. Use an empty object for a project with only local groups. |
| `sources.<name>.repository` | Explicit HTTPS or SSH Git address, including scp-style SSH. See [Repository addresses](#repository-addresses). |
| `sources.<name>.ref` | Optional revision to pin: a tag, such as a rule version tag, or a full Git commit SHA. Omit it to follow the library's releases. See [Select a revision](#select-a-revision). |
| `sources.<name>.groups` | Required group selection: an array of IDs such as `techs/typescript`, or `"*"`, `"practices/*"`, or `"techs/*"` to select all groups in that scope. |
| `sources.<name>.exclude` | Map of this library's rule IDs to exclusion reasons. |
| `sources.<name>.replace` | Map of this library's rule IDs to a local `file` and a `reason`. |

Unknown configuration fields are rejected, including unknown source and replacement fields.
Local groups are discovered from `local/<group-id>/_group.yaml`; no source entry or separate group list is required.
The former `localGroups` field is rejected with migration guidance. Remove it and keep the group metadata files.
Each source includes its own `exclude` and `replace` objects, empty when unused.
Replacement paths resolve relative to the Code Rules directory and must stay under its `local/` directory.

Each source owns its revision, groups, and exceptions.
The earlier singular `source` and top-level `groups`, `exclude`, and `replace` fields are not part of this format.
The former `version` field and version ranges are not supported. Omit `ref` to follow a library's releases, or pin a tag or commit.
The schema version is `1`.

## Import every group

Set `groups` to the string `"*"` to adopt the whole library:

```yaml
schemaVersion: 1
sources:
  team:
    repository: https://github.com/my-team/rules.git
    groups: "*"
    exclude: {}
    replace: {}
```

Choose one of three supported selectors:

| Selector | Included groups |
| --- | --- |
| `"*"` | Every technology and practice group. |
| `"practices/*"` | Every practice group. |
| `"techs/*"` | Every technology group. |

Each selector includes all groups in its scope at the selected revision, including empty groups with valid metadata.
Rule exclusions and replacements still apply. Agents still select relevant rules for each task.
When you update to a newer revision, newly added groups join the selection. Review those additions in the changed source records and generated provenance.
Offline builds do not discover changes on the remote repository.

Keep `groups` required. Use one supported selector string or an explicit array of group IDs. Wildcard arrays, mixed selectors, and arbitrary globs such as `techs/**` are unsupported.
Local metadata can describe a group that is also selected from a library, including through a wildcard.

Snapshots record both the original `groupSelection` and the concrete `groups` list.
A pattern snapshot must record the exact selector in `groupSelection` and contain every group within that scope at its resolved commit.
For example, `"practices/*"` requires all practice groups, while `"*"` requires both kinds.
The builder compares discovered groups with the recorded expansion and validates every group and rule within the selected scope.
An old partial snapshot is insufficient even if its recorded groups look complete; changing selection intent requires sync.
Legacy snapshots without `groupSelection` represent their explicit `groups` list and remain valid for list-based configuration.
This completeness declaration comes from the snapshot supplier; offline checks do not independently authenticate it against the remote repository.

## Repository addresses

Use a complete Git address so the host is explicit:

```yaml
repository: https://gitlab.com/my-team/engineering/rules.git
```

Supported forms include:

```text
https://github.com/my-team/rules.git
https://gitlab.com/my-team/engineering/rules.git
ssh://git@git.example.org:2222/srv/rules.git
git@git.example.org:engineering/rules.git
git@git.example.org:/srv/rules.git
```

Nested GitLab namespaces, private hosts, and explicit ports are supported.
The `.git` suffix is optional. Git uses the caller's credentials; do not embed HTTPS credentials or SSH passwords in configuration.
SSH usernames are allowed.

Keep the revision in `ref` and selected groups in `groups`.
Repository addresses do not accept query strings, fragments, `git::` prefixes, getter options, or `//subdirectory` selection.
Local paths, `file:`, unauthenticated `git:`, plain HTTP, and remote-helper protocols are outside this format.
The earlier `owner/name` shorthand is no longer accepted; use `https://github.com/owner/name.git` instead.

Code Rules preserves the supplied address in provenance and requires the snapshot to match it exactly.
Duplicate detection normalizes host spelling and recognizes standard GitHub.com and GitLab.com transport aliases and optional `.git` suffixes.
GitHub path matching ignores case; generic repository paths and GitLab paths retain case.
Other hosts retain their transport, username, port, and path distinctions.
For example, `git@host:rules.git` is home-relative, while `ssh://git@host/rules.git` is absolute. Code Rules does not equate them.

Validation rejects ambiguous paths, including dot segments, encoded separators in URLs, malformed URI escapes, raw whitespace, and backslashes.
SCP paths are literal Git paths, so percent signs in that form are not URI escapes.
Syntax validation does not fetch or authenticate a repository.

Generated source links use pinned GitHub.com or GitLab.com URLs for recognized standard endpoints.
For other hosts, they point to the retained source file under `vendor/<source>/`.
If a relative document or image is missing from that snapshot, generation fails with instructions to retain it or supply an explicit URL.
Code Rules does not infer a host's web interface from its name. See [How imports work](/reference/imports/).

## Source names and rule identity

Choose stable source names such as `fabrica` or `acme`.
Names must match `[a-z][a-z0-9-]*`; `local` is reserved for project-authored rules.
Declare each repository once, with its own revision selection and selected groups.

An imported rule's project ID is `<source-name>:<library-rule-id>`:

```text
fabrica:practices/testing/verify-retry-limits
acme:practices/testing/verify-retry-limits
local:practices/testing/test-project-contracts
```

A library rule ID is its path without `.md`.
The source prefix keeps identical paths from different libraries distinct.
Renaming a source changes its generated project rule IDs and any external references to those IDs.
Its nested exclusion and replacement keys remain library-relative.
Replacing a repository under an existing source name also requires reviewing those targets.

## Select a revision

A source's `ref` says which revision of the library to import:

| `ref` | Example | Imports |
| --- | --- | --- |
| Omitted | | The library's newest [release](/reference/rule-versions/#release-commits). |
| Rule version tag | `practices/testing/verify-retry-limits@1.3.0` | The release commit that published that version. |
| Other tag | `team-approved` | The commit the tag points to. |
| Full commit SHA | `0c9f3e…` (40 hex digits) | That commit. |

Configuration records what the project asked for. The source's `vendor/<source-name>/_source.json` records the exact commit it imported, like a lockfile. [`code-rules project sync`](/reference/cli/#project-sync) keeps that recorded commit while the source's repository and `ref` are unchanged, so every checkout imports the same content. [`code-rules project update`](/reference/cli/#project-update) moves a source without a `ref` to the library's newest release. Changing `ref` and running `code-rules project sync` imports the revision it now names.

### Follow a library's releases

Omit `ref` to follow the library's releases:

```yaml
repository: https://github.com/fabricahq/public-rules.git
groups:
  - practices/testing
exclude: {}
replace: {}
```

The first sync imports the library's newest release commit and records each rule's version. Later syncs keep that commit. `code-rules project update` moves to the newest release and asks you to accept major changes and retirements of rules the project uses. See [Update rules](/guides/update/).

A library that hasn't published its first release has nothing to follow, so sync fails until it does. Pin a commit to import it before then.

### Pin a tag or commit

Set `ref` to pin a source. `code-rules project update` never moves a pinned source; change or remove `ref` to move it.

A rule version tag names the release commit that published it. The project imports every selected rule as it was at that release, not only the named rule.

A tag may also use the explicit `refs/tags/<name>` form.
Branch names and abbreviated commit SHAs are unsupported.
A plain name resolves only as a tag, even when a branch has the same name.
Both lightweight and annotated tags must resolve to a commit.

Code Rules resolves a tag when you add or change `ref`, then keeps the recorded commit even if someone later moves the tag.
Offline `code-rules project build`, `code-rules project check`, and ordinary agent work use the committed snapshot without resolving tags again.

Snapshots and generated provenance record the requested `ref` and the `resolvedCommit`.

## Source-scoped exceptions

Keys inside a source's `exclude` and `replace` use the library-relative rule ID, without a source prefix.
For example, `sources.fabrica.exclude["practices/testing/verify-retry-limits"]` affects only Fabrica's rule.
An identically named rule from `acme` remains active.
Full paths distinguish matching filenames in different groups of the same library.

Generated files and review findings retain source-qualified IDs because they appear outside the configuration's source nesting.

## Group selection

Select complete groups separately for each source, then exclude individual rules when necessary.
Each selected group must exist in that source at its pinned revision.
Selecting `practices/testing` from two sources combines both sets of rules into one effective testing group, with a group page and individual resolved rule files.
Source order never establishes precedence.

Every valid rule file under `local/` automatically joins its adopted group; individual local rules do not need source entries.
Local rules can join any imported group. A matching filename does not override an imported rule.
A local file referenced by `replace` appears once under its local ID; the target is removed from active output.
The complete local definition supplies the metadata, guidance, attribution, and assets. Configuration and provenance retain the replacement relationship.
A local `_group.yaml` defines a group, including an empty group, without any configuration entry.
If local metadata exists for an imported group, its complete description and reading cues take precedence for project discovery.
Otherwise, descriptions from every contributing library remain source-labeled.
Provenance retains all group metadata and identifies the sources supplying the effective discovery guidance.
Local rules without either local or imported group metadata are errors, with the missing `_group.yaml` path in the diagnostic.
The root `local/README.md` and each group-root `README.md` are authoring documentation, not rules; other misplaced Markdown files are still validated.

The generated index shows group names, applicability guidance, and explicit **Open group** links. Without local metadata, guidance from multiple libraries remains labeled by source.
Local metadata supplies the complete project description when present.
The importer does not silently choose one library's description over another's.

## Conflicting rules

Source-qualified IDs prevent naming collisions, but distinct rules can still require incompatible behavior in the same situation.
Both remain active unless the project explicitly excludes or replaces a rule under its owning source.
The importer does not infer priority from source order or interpret prose to resolve contradictions.

See [Resolve conflicting rules](/guides/conflicting-guidance/) for examples, an agent review prompt, and ways to resolve competing instructions.

## Version boundaries

`schemaVersion` describes this configuration.
Each library's `formatVersion` describes its authoring format.
The caller-supplied `toolVersion` identifies the tool that generated the output.

Each source's `ref`, or its absence, selects the requested revision.
Vendored provenance records the resolved commit and each rule's version, used by offline commands. It is importer-owned output, not a second user-selected version.
A project imports multiple sources directly.
Libraries that themselves inherit and republish other libraries are not supported.

See [Library format](/reference/library-format/) for library metadata and [Adapt rules](/guides/select-rules/#adapt-the-import-to-your-project) for examples of exceptions.
