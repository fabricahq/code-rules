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
    exclude:
      techs/typescript/prefer-type-aliases:
        reason: Our public extension API relies on declaration merging.
        replacedBy: local/techs/typescript/prefer-interfaces.md
  acme:
    repository: https://github.com/acme/.code-rules.git
    groups:
      - techs/react
      - practices/testing
      - practices/observability
    pins:
      practices/testing/verify-retry-limits:
        version: "1.3.0"
        reason: Waiting on the author's response to acme/.code-rules#45.
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
| `sources.<name>.exclude` | Optional map of this library's rule IDs to a `reason` and, optionally, a local rule that `replacedBy` names to use instead, with the library version it's `basedOn`. See [Exclude or replace a rule](#exclude-or-replace-a-rule). |
| `sources.<name>.pins` | Optional map of this library's rule IDs to an exact `version` and a `reason`. See [Pin a rule](#pin-a-rule). |
| `sources.<name>.ref` | Optional and advanced. Import the library exactly as it was at one tag or commit. Can't be combined with `pins`. See [Import one revision](#import-one-revision). |

Unknown configuration fields are rejected, including unknown source and exclusion fields.
Local groups are discovered from `local/<group-id>/_group.yaml`; no source entry or separate group list is required.
The former `localGroups` field is rejected with migration guidance. Remove it and keep the group metadata files.
`exclude` is optional; omit it when a source has no exceptions.

Each source owns its selection, pins, and exceptions.
The earlier singular `source` and top-level `groups`, `exclude`, and `replace` fields are not part of this format.
Sources have no `version` or `replace` field. Rule versions are chosen rule by rule; use `pins` to keep individual rules at exact versions, and `exclude` with `replacedBy` to replace a rule.
The schema version is `1`.

## Import every group

Set `groups` to the string `"*"` to adopt the whole library:

```yaml
schemaVersion: 1
sources:
  team:
    repository: https://github.com/my-team/rules.git
    groups: "*"
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

Keep selected groups in `groups` and pinned versions in `pins`.
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
Declare each repository once, with its own selection and pins.

An imported rule's project ID is `<source-name>:<library-rule-id>`:

```text
fabrica:practices/testing/verify-retry-limits
acme:practices/testing/verify-retry-limits
local:practices/testing/test-project-contracts
```

A library rule ID is its path without `.md`.
The source prefix keeps identical paths from different libraries distinct.
Renaming a source changes its generated project rule IDs and any external references to those IDs.
Its nested `exclude` and `pins` keys remain library-relative.
Replacing a repository under an existing source name also requires reviewing those keys.

## Choose versions

Each rule in a library has its own [version](/reference/rule-versions/). By default, a project follows each rule's newest version, but only when someone updates:

- `vendor/<source-name>/_source.json` records the exact version of every imported rule, like a lockfile. [`code-rules project sync`](/reference/cli/#project-sync) restores those versions, so every checkout imports the same content.
- [`code-rules project update`](/reference/cli/#project-update) previews newer versions, new rules, and retirements, and applies them after you confirm.

Two optional source fields change that:

| Field | Use it to |
| --- | --- |
| `pins` | Keep individual rules at exact versions while the rest follow updates. |
| `ref` | Import the library exactly as it was at one Git revision. Advanced. |

A source can use `pins` or `ref`, not both. Neither changes which rules are imported; `groups` and `rules` do that.

### Pin a rule

```yaml
pins:
  practices/testing/verify-retry-limits:
    version: "1.3.0"
    reason: Waiting on the author's response to acme/.code-rules#45.
```

| Field | Meaning |
| --- | --- |
| `version` | Required. An exact published version of the rule, such as `"1.3.0"`. Quote it so YAML reads it as text. |
| `reason` | Required non-blank text explaining why the rule stays at this version. `code-rules project update` shows it next to any newer version, so the decision keeps its context. |

A pin keeps the rule at exactly that version:

- `code-rules project update` never moves a pinned rule. Its preview lists the pin, the newest available version, and your reason.
- When you add or change a pin, `code-rules project sync` moves the rule to the pinned version, up or down.
- Removing a pin doesn't move the rule; the next `code-rules project update` offers its newest version.

To keep a rule at the version you have, pin it to the version recorded in `_source.json`, which also appears in the rule's generated file. `code-rules project update --keep` writes that pin for you.

A pin must name a rule the source imports. A pinned rule that its library retires stays at its version, and update reports the retirement. To keep a rule the library is retiring, pin it before or during the update that retires it; once the retirement is applied, the rule is no longer imported and can't be pinned.

### Import one revision

Set a source's `ref` to import the library exactly as it was at one Git revision:

```yaml
sources:
  vendor-rules:
    repository: https://github.com/example/engineering-rules.git
    groups: "*"
    ref: release/5
```

`ref` accepts a tag name, such as the library release tag `release/5`, or a full 40-character commit SHA. A tag may also use the explicit `refs/tags/<name>` form. Branch names and abbreviated SHAs aren't supported, so every import can be reproduced. Code Rules resolves the tag when you add or change `ref`, then keeps the recorded commit even if someone later moves the tag.

What you get depends on what the revision is:

- **A [library release](/reference/rule-versions/#library-releases)**, which is the usual case. Every selected rule is imported at the version that library release published, as recorded in its release record.
- **Any other commit or tag**, such as unreleased changes a library author wants to test in a real project, or a library that hasn't published its first library release. Rules whose files match a published version record that version. Rules with unreleased changes record their version as `null`, and generated guidance shows no version for them.

Importing a revision other than a library release opts that source out of rule versions: rules with unreleased changes have no version to cite, and no update preview reviews their changes. To keep that from shipping by accident, `code-rules project sync` and `code-rules project update` print a warning naming the source and its unreleased rules, and the generated library summary in `generated/libraries/<source-name>/README.md` says the source is imported from unreleased changes. `code-rules project check` still passes.

`code-rules project update` doesn't move a source that uses `ref`. To import another revision, change `ref` and run `code-rules project sync`. To go back to following versions, remove `ref` and run `code-rules project sync`: rules keep their recorded versions when those are published versions, and the rest get their newest version.

### Where each rule's files come from

Each rule's [Markdown file and asset directory](/reference/rule-versions/#what-a-version-covers) come from the library release that published its version. Library-wide files, such as group metadata, shared assets, and license files, come from one library release, recorded as `release` in the source's [snapshot record](/reference/provenance/#inspect-the-original-imported-files), or from the revision your `ref` names. A new source, and newly selected groups or rules, take them from the newest library release, and `code-rules project update` moves them to the newest library release; `code-rules project sync` keeps the recorded one. That library release is never older than the one that published any imported rule version. When that library release no longer has the metadata of an imported rule's group, such as the group of a retired rule you kept, the group's metadata comes from the newest library release among its imported rule versions that still has it.

Offline `code-rules project build`, `code-rules project check`, and ordinary agent work use the recorded versions without contacting the repository.

## Exclude or replace a rule

A source's `exclude` map leaves rules out of the generated guidance. Each entry records why, and can name a local rule for agents to read instead:

```yaml
exclude:
  practices/testing/avoid-snapshot-tests:
    reason: Contract snapshots follow our separate review policy.
  techs/typescript/prefer-type-aliases:
    reason: Our public extension API relies on declaration merging.
    replacedBy: local/techs/typescript/prefer-interfaces.md
    basedOn: "2.0.0"
```

| Field | Meaning |
| --- | --- |
| `reason` | Required non-blank text explaining why the project leaves the rule out. |
| `replacedBy` | Optional. A local rule file that agents read in place of the excluded rule. The path is relative to the Code Rules directory, must stay under `local/`, and must be in the same group as the rule it replaces. Each local file can replace only one rule. |
| `basedOn` | Optional, and only with `replacedBy`. The library version of the excluded rule that your local rule incorporates, in quotes, such as `"2.0.0"`. [Forking a rule](/reference/cli/#fork-a-library-rule) records the forked version here. Offline commands check only that it's a version; `code-rules project sync` checks that the library published that version of the rule, and otherwise fails with `version-not-found`, listing the versions it did publish. |

An excluded rule is still imported into `vendor/`, so you can review its changes. The `code-rules project update` preview lists a replaced rule as `replaced` so you can decide whether your local rule needs the same change:

- With `basedOn`, whenever the library's newest version of the rule is newer than `basedOn`, whatever version the project imports. The preview lists every change after `basedOn`. Once your local rule has them, record that with `code-rules project update --incorporated SOURCE:RULE`, or the terminal question, which sets `basedOn` to the newest version, or edit `basedOn` yourself. Code Rules never advances it on its own, because only you know whether your rule took the changes in.
- Without `basedOn`, such as for a replacement you wrote yourself, when the update moves the imported copy to a newer version. The preview lists the changes since the version the project imports, which your rule may already have.

### Source-scoped exceptions

Keys inside a source's `exclude` use the library-relative rule ID, without a source prefix.
For example, `sources.fabrica.exclude["practices/testing/verify-retry-limits"]` affects only Fabrica's rule.
An identically named rule from `acme` remains active.
Full paths distinguish matching filenames in different groups of the same library.

Generated files and review findings retain source-qualified IDs because they appear outside the configuration's source nesting.

An `exclude` or `pins` key must name a rule this source imports, through its groups or its `rules` list; any other ID fails validation. When the library retires a rule that an exclusion names, the entry no longer does anything. `code-rules project sync` and `code-rules project update` warn about it so you can delete it; nothing else is blocked. Offline, `code-rules project build` and `code-rules project check` accept a pin at the version the source imports, an exclusion of an imported rule, and a pin or exclusion naming a rule the source's [snapshot record](/reference/provenance/#inspect-the-original-imported-files) lists as retired. For any other pin or exclusion, they ask you to run `code-rules project sync`, which checks it against the library. The snapshot record doesn't hold pins, so adding, rewording, or removing a pin that moves no rule never changes it.

## Group selection

Select complete groups separately for each source, add individual rules if needed, then exclude rules when necessary.
Each selected group must exist in the imported library.
A selected group includes rules the library adds to it later.
Selecting `practices/testing` from two sources combines both sets of rules into one effective testing group, with a group page and individual resolved rule files.
Source order never establishes precedence.

Every valid rule file under `local/` automatically joins its adopted group; individual local rules do not need source entries.
Local rules can join any imported group. A matching filename does not override an imported rule.
A local file that an exclusion names as `replacedBy` appears once under its local ID; the excluded rule is removed from active output.
The complete local definition supplies the metadata, guidance, attribution, and assets. Configuration and provenance retain the replacement relationship.
A local `_group.yaml` defines a group, including an empty group, without any configuration entry.
If local metadata exists for an imported group, its complete description and reading cues take precedence for project discovery.
Otherwise, descriptions from every contributing library remain source-labeled.
Provenance retains all group metadata and identifies the sources supplying the effective discovery guidance.
Local rules without either local or imported group metadata are errors, with the missing `_group.yaml` path in the diagnostic. When a sync or update removes the last import supplying a local rule's group, it writes the group's last imported metadata to `local/<group-id>/_group.yaml` instead of failing; see [Keep a local rule's group](/reference/sync/#keep-a-local-rules-group).
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
```

This imports every rule in `techs/react`, including rules added to it later, plus exactly two other rules. Rules the library later adds to `practices/testing` or `techs/go` don't join, because those groups aren't selected.

- Each entry is a library-relative rule ID that must exist in the library.
- An individually selected rule brings its group's metadata, so its group appears in the generated index with only the selected rules.
- The imported rules are the union of both lists: every rule in the selected groups, plus every listed rule. Listing a rule whose group is also selected is allowed and changes nothing; `code-rules project sync` and `code-rules project update` warn about it so you can delete the entry.
- `exclude` applies to every imported rule, however it was selected.
- When the library retires an individually selected rule, `code-rules project update` shows the retirement in its preview. After you confirm, the entry no longer does anything, and later syncs and updates warn about it so you can delete it. To keep the rule instead, pin it to its last version.

Individually selected rules follow updates and pins like any other imported rule.

## Conflicting rules

Source-qualified IDs prevent naming collisions, but distinct rules can still require incompatible behavior in the same situation.
Both remain active unless the project explicitly excludes or replaces a rule under its owning source.
The importer does not infer priority from source order or interpret prose to resolve contradictions.

See [Resolve conflicting rules](/guides/conflicting-guidance/) for examples, an agent review prompt, and ways to resolve competing instructions.

## Version boundaries

`schemaVersion` describes this configuration.
Each library's `formatVersion` describes its authoring format.
The caller-supplied `toolVersion` identifies the tool that generated the output.

Each source's `pins`, and its `ref` when present, state which rule versions the project wants; every other rule follows its newest version when the project updates.
Vendored provenance records each imported rule's version, library release, and commit, used by offline commands. It is importer-owned output, not a second user-selected version.
A project imports multiple sources directly.
Libraries that themselves inherit and republish other libraries are not supported.

See [Library format](/reference/library-format/) for library metadata and [Adapt rules](/guides/select-rules/#adapt-the-import-to-your-project) for examples of exceptions.
