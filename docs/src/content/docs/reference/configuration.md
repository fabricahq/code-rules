---
title: "Configuration"
description: "Fields in .code-rules/config.yaml."
---

The **project configuration**, `.code-rules/config.yaml`, records the project's sources, selected groups, and exceptions. The **Code Rules directory**, `.code-rules/`, also holds `local/`, `vendor/`, and `generated/`.

Initialize from the project root. In Git repositories, other project commands find the nearest Git root and use its `.code-rules/config.yaml`. Outside Git, run commands from the project root. See [Working directories](/reference/cli/#working-directories).
A project can import rules directly from multiple canonical libraries, choosing versions for each one independently.
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
    groups:
      - techs/react
      - practices/testing
      - practices/observability
    versions:
      default: hold
      rules:
        practices/testing/verify-retry-limits: latest
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
| `sources.<name>.groups` | Groups to import in full: an array of IDs such as `techs/typescript`, or `"*"`, `"practices/*"`, or `"techs/*"` to select all groups in that scope. Required unless `rules` selects individual rules. |
| `sources.<name>.rules` | Optional individual rules to import without the rest of their group: an array of library rule IDs. See [Select individual rules](#select-individual-rules). |
| `sources.<name>.exclude` | Map of this library's rule IDs to exclusion reasons. |
| `sources.<name>.replace` | Map of this library's rule IDs to a local `file` and a `reason`. |
| `sources.<name>.versions` | Optional version choices. Omit it to follow each rule's newest version. See [Choose versions](#choose-versions). |

Unknown configuration fields are rejected, including unknown source and replacement fields.
Local groups are discovered from `local/<group-id>/_group.yaml`; no source entry or separate group list is required.
The former `localGroups` field is rejected with migration guidance. Remove it and keep the group metadata files.
Each source includes its own `exclude` and `replace` objects, empty when unused.
Replacement paths resolve relative to the Code Rules directory and must stay under its `local/` directory.

Each source owns its version choices, groups, and exceptions.
The earlier singular `source` and top-level `groups`, `exclude`, and `replace` fields are not part of this format.
The former source-level `ref` and `version` fields are rejected with migration guidance. Use `versions.ref` to import a fixed revision, or omit `versions` to follow each rule's newest version.
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

Each selector includes all groups in its scope in the imported library, including empty groups with valid metadata.
Rule exclusions and replacements still apply. Agents still select relevant rules for each task.
When a newer library release adds groups within the selector's scope, `code-rules project update` adds them and their rules. Review those additions in the changed source records and generated provenance.
Offline builds do not discover changes on the remote repository.

Select at least one group or individual rule. For `groups`, use one supported selector string or an explicit array of group IDs. Wildcard arrays, mixed selectors, and arbitrary globs such as `techs/**` are unsupported.
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

Keep version choices in `versions` and selected groups in `groups`.
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
Declare each repository once, with its own version choices and selected groups.

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

## Choose versions

Each rule in a library has its own [version](/reference/rule-versions/). A source's optional `versions` field chooses which versions to import. Omit it to follow each rule's newest version, which suits most projects.

`versions` takes one of two forms:

| Form | Use it to |
| --- | --- |
| `default` and `rules` | Choose versions rule by rule. |
| `ref` | Import the library exactly as it was at one Git revision, such as a library release. |

`ref` can't be combined with `default` or `rules`.

Configuration states what the project wants. The source's `vendor/<source-name>/_source.json` records the exact version and commit of every rule it imported, like a lockfile. [`code-rules project sync`](/reference/cli/#project-sync) restores those recorded versions, so every checkout imports the same content, and [`code-rules project update`](/reference/cli/#project-update) looks for newer versions that `versions` allows. Neither changes configuration.

### Choose versions rule by rule

```yaml
versions:
  default: hold
  rules:
    practices/testing/verify-retry-limits: latest
    practices/testing/verify-backoff: "~> 1.3"
```

Read this as: keep every rule at its current version, let `verify-retry-limits` follow its newest version, and let `verify-backoff` take compatible updates from `1.3` onward.

`default` sets the choice for every imported rule, and `rules` overrides it for individual rules by library-relative rule ID:

| Choice | Allowed in | Meaning |
| --- | --- | --- |
| `latest` | `default`, `rules` | Follow the newest version. This is the default when `versions` or `default` is omitted. |
| `hold` | `default` | Keep each rule at the version recorded in `_source.json`, or import the newest version if none is recorded yet. Update reports newer versions without applying them. |
| A constraint, such as `"1.3.0"` or `"~> 1.3"` | `rules` | Allow only versions that satisfy the constraint. An exact version pins the rule. |

`default` accepts only `latest` or `hold`. Each rule has its own version numbers, so a constraint such as `"~> 1.3"` can't apply across rules. To keep one rule where it is, pin it to its current version, such as `"1.3.0"`; you'll find the version in `_source.json`, in the rule's generated file, and in `code-rules project update` output. Pinning to an exact version, rather than holding, keeps the decision visible in configuration.

`code-rules project update` moves each rule to the newest version its choice allows. Major changes and retirements of rules the project uses still need `--accept-major`, even when a constraint allows them. A constraint limits which versions update can choose; consent confirms a major change.

`versions` never changes which rules are imported. Rules that newer library releases add to a selected group join, at the version their choice allows, and update reports them as new. To import only specific rules, [select them individually](#select-individual-rules).

A `versions.rules` entry must name a rule the source imports, including a retired rule the project still imports at a pinned version; other IDs are errors. Remove an entry to return the rule to `default`; the next sync or update reports any resulting change. A pinned rule that its library retires stays at its version, and update reports the retirement. Nothing moves an entry to a replacement rule automatically; add the replacement yourself.

### Constraints

Constraints use [HashiCorp go-version syntax](https://github.com/hashicorp/go-version). Surrounding whitespace is trimmed, and comma-separated comparisons must all match.

| Constraint | Allowed versions |
| --- | --- |
| `"1.3.0"` or `"= 1.3.0"` | Exactly 1.3.0 |
| `"~> 1.3"` | At least 1.3.0, below 2.0.0 |
| `"~> 1.3.2"` | At least 1.3.2, below 1.4.0 |
| `">= 1.3.0, < 2.0.0"` | Explicit lower and upper bounds |
| `"!= 1.3.1"` | Any version except 1.3.1 |

Quote constraints so YAML reads them as text. Caret ranges (`^`), npm tilde ranges (`~`), wildcard versions (`1.x`), OR (`||`), and space-separated comparisons are not supported; use commas for AND. Rule versions have no prerelease or build suffixes.

### Import one revision

Set `ref` to import the library exactly as it was at one Git revision:

```yaml
versions:
  ref: release/5
```

`ref` accepts a tag name, such as the library release tag `release/5`, or a full 40-character commit SHA. A tag may also use the explicit `refs/tags/<name>` form. Branch names and abbreviated SHAs aren't supported, so every import can be reproduced. Code Rules resolves the tag when you add or change `ref`, then keeps the recorded commit even if someone later moves the tag.

What you get depends on what the revision is:

- **A [library release](/reference/rule-versions/#library-releases)**, which is the usual case. Every selected rule is imported at the version that library release published, as recorded in its release manifest.
- **Any other commit or tag**, such as unreleased changes a library author wants to test in a real project, or a library that hasn't published its first library release. Rules whose files match a published version record that version. Rules with unreleased changes record their version as `null`, and generated guidance shows no version for them.

Importing a revision other than a library release opts that source out of rule versions: rules with unreleased changes have no version to cite, and nothing asks for consent to major changes. To keep that from shipping by accident, `code-rules project sync` and `code-rules project update` print a warning naming the source and its unreleased rules, and the generated library summary in `generated/libraries/<source-name>/README.md` says the source is imported from unreleased changes. `code-rules project check` still passes.

`code-rules project update` doesn't move a source that uses `ref`. To import another revision, change `ref` and run `code-rules project sync`. To go back to following versions, remove `ref`.

### Where each rule's files come from

Each rule's [versioned content](/reference/rule-versions/#what-a-version-covers) comes from the library release that published its version, so rules at different versions keep the shared files they were written with. Group metadata and the library's license files come from the newest library release among the imported rule versions, or from the revision your `ref` names.

Offline `code-rules project build`, `code-rules project check`, and ordinary agent work use the recorded versions without contacting the repository.

## Source-scoped exceptions

Keys inside a source's `exclude` and `replace` use the library-relative rule ID, without a source prefix.
For example, `sources.fabrica.exclude["practices/testing/verify-retry-limits"]` affects only Fabrica's rule.
An identically named rule from `acme` remains active.
Full paths distinguish matching filenames in different groups of the same library.

Generated files and review findings retain source-qualified IDs because they appear outside the configuration's source nesting.

## Group selection

Select complete groups separately for each source, then exclude individual rules when necessary.
Each selected group must exist in the imported library.
A selected group includes rules the library adds to it later.
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

## Select individual rules

To import some rules without the rest of their group, list them under the source's `rules`:

```yaml
sources:
  team:
    repository: https://github.com/acme/.code-rules.git
    groups:
      - techs/react
    rules:
      - practices/testing/verify-retry-limits
      - techs/go/wrap-errors-with-operation
    exclude: {}
    replace: {}
```

This imports every rule in `techs/react`, including rules added to it later, plus exactly two other rules. Rules the library later adds to `practices/testing` or `techs/go` don't join, because those groups aren't selected.

- Each entry is a library-relative rule ID that must exist in the library.
- An individually selected rule brings its group's metadata, so its group appears in the generated index with only the selected rules.
- Listing a rule whose group is already selected is an error, because the entry would add nothing and could be misread as narrowing the group or choosing the rule's version. The error suggests removing the entry, or using `versions.rules` to choose the rule's version.
- To stop using an individually selected rule, remove it from `rules`. Exclusions apply only to rules imported through groups.
- When the library retires an individually selected rule, `code-rules project update` reports the retirement and writes nothing. Remove the entry to stop using the rule, or pin the rule to its last version in `versions.rules` to keep it.

Individually selected rules follow `versions` like any other imported rule.

## Conflicting rules

Source-qualified IDs prevent naming collisions, but distinct rules can still require incompatible behavior in the same situation.
Both remain active unless the project explicitly excludes or replaces a rule under its owning source.
The importer does not infer priority from source order or interpret prose to resolve contradictions.

See [Resolve conflicting rules](/guides/conflicting-guidance/) for examples, an agent review prompt, and ways to resolve competing instructions.

## Version boundaries

`schemaVersion` describes this configuration.
Each library's `formatVersion` describes its authoring format.
The caller-supplied `toolVersion` identifies the tool that generated the output.

Each source's `versions`, or its absence, states which rule versions the project wants.
Vendored provenance records each imported rule's version, library release, and commit, used by offline commands. It is importer-owned output, not a second user-selected version.
A project imports multiple sources directly.
Libraries that themselves inherit and republish other libraries are not supported.

See [Library format](/reference/library-format/) for library metadata and [Adapt rules](/guides/select-rules/#adapt-the-import-to-your-project) for examples of exceptions.
