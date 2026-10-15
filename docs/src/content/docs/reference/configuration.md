---
title: "Configuration"
description: "Fields in .code-rules/config.yaml."
---

The **project configuration**, `.code-rules/config.yaml`, records the project's sources, the rules and skills it includes from each, its exceptions, and where it installs skills. The **Code Rules directory**, `.code-rules/`, also holds `rules/`, `assets/`, `skills/`, `vendor/`, and `generated/`.

:::note[Preview]
`skillTargets`, `skills.include`, `skills.uses`, and sources that aren't libraries come in the next Code Rules release.
:::

Initialize from the project root. In Git repositories, other project commands find the nearest Git root and use its `.code-rules/config.yaml`. Outside Git, run commands from the project root. See [Working directories](/reference/cli/#working-directories).
A project can import rules directly from multiple canonical libraries, choosing versions for each one independently.
Sync, build, and check validate the fields below.

Use one YAML document. Duplicate keys, anchors, aliases, and explicit tags are rejected. Quote selectors that start with `*`, such as `"*"`, because YAML reads a bare `*` as an alias.

## Complete example

```yaml
schemaVersion: 2
skillTargets: [claude, agents]
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    rules:
      include: [techs/typescript/*, practices/testing/*]
    skills:
      include: [skills/build-feature]
      exclude:
        - id: techs/typescript/prefer-type-aliases
          reason: Our public extension API relies on declaration merging.
          replacedBy: techs/typescript/prefer-interfaces
  acme:
    repository: https://github.com/acme/.code-rules.git
    rules:
      include: [techs/react/*, practices/*]
      pins:
        - id: practices/testing/verify-retry-limits
          version: "1.3.0"
          reason: Waiting on the author's response to acme/.code-rules#45.
  review-tools:
    repository: https://github.com/example/review-skills.git
    ref: v2.1.0
    skills:
      include: ["*"]
```

Repository names, groups, rule IDs, and skill IDs in examples are illustrative.
Replace them with libraries and rules your project can access.

## Fields

| Field | Meaning |
| --- | --- |
| `schemaVersion` | Configuration format version; Code Rules accepts only `2`. See [Version 1 configuration](#version-1-configuration). |
| `skillTargets` | Optional. Where build and sync install skills: a list of `claude`, for `.claude/skills/`, and `agents`, for `.agents/skills/`. An empty list, or no field, installs no copies. `code-rules project init` writes `[claude, agents]`. See [Skill targets](#skill-targets). |
| `sources` | Map of stable source names to source configurations. Use an empty object for a project with only its own groups and skills. |
| `sources.<name>.repository` | Explicit HTTPS or SSH Git address, including scp-style SSH. See [Repository addresses](#repository-addresses). |
| `sources.<name>.ref` | Import the source exactly as it was at one tag or commit. Required for a repository without library releases, such as one that isn't a library. Can't be combined with `rules.pins`. See [Import one revision](#import-one-revision). |
| `sources.<name>.rules` | The rules to include from this library, and the project's exceptions. A source needs `rules`, `skills`, or both. |
| `sources.<name>.rules.include` | Required. A list of at least one [selector](#select-rules): whole groups, such as `techs/typescript/*`, every group of a type, such as `practices/*`, everything, `"*"`, or single rule IDs. |
| `sources.<name>.rules.exclude` | Optional list of entries, each with a selector as its `id`, a `reason` and, for a single rule, optionally a local rule that `replacedBy` names to use instead, with the library version it's `basedOn`. See [Exclude or replace a rule](#exclude-or-replace-a-rule). |
| `sources.<name>.rules.pins` | Optional list of entries, each with a single rule ID as its `id`, an exact `version`, and a `reason`. See [Pin a rule](#pin-a-rule). |
| `sources.<name>.skills.include` | A list of skill IDs, such as `skills/build-feature`, or `"*"` alone for every skill the source offers. See [Select skills](#select-skills). |
| `sources.<name>.skills.uses` | Optional map of a skill ID to the skill IDs it uses that its own `fg-skills-used` doesn't list. See [Declare the skills a skill uses](#declare-the-skills-a-skill-uses). |
| `library` | Optional. Makes the repository a library that publishes the groups in `rules/`: the library's manifest, which can be empty, `library: {}`, or declare the library's `license`. See [Library section](#library-section). |

Unknown configuration fields are rejected, including unknown source and exclusion fields.
Groups the project authors are discovered from `rules/<group-id>/_group.yaml`; no source entry or separate group list is required.
The former `localGroups` field is rejected with migration guidance. Remove it and keep the group metadata files.
`rules.exclude` and `rules.pins` are optional; omit them when a source has no exceptions.

Each source owns its selection, pins, and exceptions.
Sources have no `version` or `replace` field. Rule versions are chosen rule by rule; use `rules.pins` to keep individual rules at exact versions, and `rules.exclude` with `replacedBy` to replace a rule.

## Select rules

A source's `rules.include` lists selectors. A **selector** is a rule ID, or a prefix that ends in `/*`:

| Selector | Includes |
| --- | --- |
| `"*"` | Every rule in the library. |
| `practices/*` | Every practice rule, at any depth. |
| `techs/*` | Every technology rule, at any depth. |
| `techs/go/*` | Every rule in one group, such as `techs/go`. |
| `practices/testing/verify-retry-limits` | One rule. See [Select individual rules](#select-individual-rules). |

For example, to adopt the whole library:

```yaml
schemaVersion: 2
sources:
  team:
    repository: https://github.com/my-team/rules.git
    rules:
      include: ["*"]
```

The `*` is allowed only at the end, after a `/`, or alone: `techs/go*`, `techs/*/testing`, and `**` are rejected. A prefix ending in `/*` names a type, `techs` or `practices`, or one group; a directory inside a group, such as `techs/go/errors/*`, is rejected. A group ID on its own, such as `techs/go`, is rejected with a hint to write `techs/go/*`. Each entry may appear once.

Combine selectors freely: `[techs/*, practices/testing/*]` includes every technology group and one practice group. A selector that another in the same list already covers, such as `techs/go/*` beside `techs/*`, changes nothing; `code-rules project sync` and `code-rules project update` warn about it so you can delete it.

A selector ending in `/*` includes every group in its scope in the imported library, including empty groups with valid metadata. Each group it names must exist.
Rule exclusions and replacements still apply. Agents still select relevant rules for each task.
When a newer library release adds rules or groups within a selector's scope, `code-rules project update` adds them. Review those additions in the changed source records and generated provenance.
Offline builds do not discover changes on the remote repository.

Local metadata can describe a group that is also selected from a library, including through a selector.

Snapshots record the selected groups and type selectors as `groupSelection`, and the concrete `groups` list.
A snapshot of a type selector or `"*"` must record it in `groupSelection` and contain every group within that scope at its resolved commit.
For example, `"practices/*"` requires all practice groups, while `"*"` requires both kinds.
The builder compares discovered groups with the recorded expansion and validates every group and rule within the selected scope.
An old partial snapshot is insufficient even if its recorded groups look complete; changing selection intent requires sync.
This completeness declaration comes from the snapshot supplier; offline checks do not independently authenticate it against the remote repository.

## Skill targets

`skillTargets` names the folders that build and sync install each skill into, so agents that load skills from them find it:

| Target | Folder |
| --- | --- |
| `claude` | `.claude/skills/` |
| `agents` | `.agents/skills/` |

Each installed skill is a folder named by its install name, such as `.claude/skills/build-feature/`. List each target once. With `skillTargets: []`, or without the field, the project still records and lists its skills in `generated/SKILLS.md`, and sync warns that it installed no copy. Code Rules writes only to these folders in the project, never to your home directory, and only to folders it installed; see [Skills](/reference/sync/#skills).

## Select skills

A source's `skills.include` lists the skills to install from it, all from the one revision its record locks:

```yaml
sources:
  team:
    repository: https://github.com/my-team/rules.git
    rules:
      include: [techs/go/*]
    skills:
      include: [skills/build-feature, skills/review-tests]
```

A **skill ID** is the path of the skill's folder. A library's skills are the folders directly in its `.code-rules/skills/`, with IDs such as `skills/build-feature`. In a repository that isn't a library, a skill is any folder that holds a `SKILL.md`, other than the repository root and a folder inside another skill's folder, and its ID is its path from the repository root, such as `tools/review`.

`"*"` includes every skill the source offers, except, in a repository that isn't a library, skills in a folder whose name starts with a dot, such as another tool's installed copies in `.claude/skills/`; name one by its ID to include it. `"*"` can't be combined with skill IDs, and skill selectors have no prefixes such as `skills/*`. Each ID may appear once.

A source that isn't a library has only skills, so it needs a `ref` and can't include `rules`:

```yaml
sources:
  review-tools:
    repository: https://github.com/example/review-skills.git
    ref: v2.1.0
    skills:
      include: [tools/review]
```

Sync also installs every skill an included skill uses, from the same source; see [Declare the skills a skill uses](#declare-the-skills-a-skill-uses).

Skill levels, such as `recommended`, and the `rename` and `exclude` fields aren't supported yet.

### Declare the skills a skill uses

A skill lists the other skills it calls in its `fg-skills-used` metadata, one skill ID per line, and sync installs them with it, and the skills they use in turn, all from the same source and revision. A skill installed only because another uses it isn't listed in `skills.include`, and `generated/SKILLS.md` says which skill uses it.

Many skills, especially ones from a repository that isn't a library, call other skills without listing them. Declare those calls in the source's `skills.uses`, so sync installs them too:

```yaml
sources:
  pstack:
    repository: https://github.com/example/pstack.git
    ref: v1.4.0
    skills:
      include: [skills/poteto-mode]
      uses:
        skills/architect: [skills/how]
```

Each key is a skill the source installs, because `skills.include` selects it or another installed skill uses it, and each value lists skill IDs in the same repository. `uses` adds to what the skill's `fg-skills-used` lists; it can't remove one. A skill can't use itself, and the skills used can't form a cycle, directly or through other skills. Code Rules doesn't search skill text for calls: `fg-skills-used` and `skills.uses` are the only ways it learns of them.

Sync fails, writing nothing, when a skill uses one the source's revision doesn't have, naming both skills, and when skills use each other in a cycle. A `skills.uses` entry that names a skill the recorded revision lacks fails with `skill-not-in-revision`, because sync never moves a source. Offline, `code-rules project build` and `code-rules project check` ask for `code-rules project sync` when `skills.uses` changed what the source installs.

Dependencies don't cross sources: a source's skill uses skills from its own repository. A skill you author in `.code-rules/skills/` can use a source's skill by its project ID, such as `pstack:skills/how`, but Code Rules doesn't install it for that: the source must install it, so add it to the source's `skills.include`.

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

Keep selected rules in `rules.include` and pinned versions in `rules.pins`.
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
The `id` of each of its `rules.exclude` and `rules.pins` entries remains library-relative.
Replacing a repository under an existing source name also requires reviewing those entries.

A skill's project ID works the same way: `<source-name>:<skill-id>`, such as `fabrica:skills/build-feature`, while a skill authored in `.code-rules/skills/` keeps its bare ID, such as `skills/release-checklist`. Installed copies are named by install name, not ID, so two skills with the same install name can't both be installed: build and sync fail with `skill-name-clash` until you stop including one of them.

## Choose versions

Each rule in a library has its own [version](/reference/rule-versions/). By default, a project follows each rule's newest version, but only when someone updates:

- `vendor/<source-name>/_source.json` records the exact version of every imported rule, like a lockfile. [`code-rules project sync`](/reference/cli/#project-sync) restores those versions, so every checkout imports the same content.
- [`code-rules project update`](/reference/cli/#project-update) previews newer versions, new rules, and retirements, and applies them after you confirm.

Two optional source fields change that:

| Field | Use it to |
| --- | --- |
| `rules.pins` | Keep individual rules at exact versions while the rest follow updates. |
| `ref` | Import the library exactly as it was at one Git revision. Advanced. |

A source can use `rules.pins` or `ref`, not both. Neither changes which rules are imported; `rules.include` does that.

### Pin a rule

```yaml
rules:
  include: [practices/testing/*]
  pins:
    - id: practices/testing/verify-retry-limits
      version: "1.3.0"
      reason: Waiting on the author's response to acme/.code-rules#45.
```

| Field | Meaning |
| --- | --- |
| `id` | Required. The library rule ID of one rule. A pin keeps one rule, so its `id` can't be a selector ending in `*`. Each rule can have one pin. |
| `version` | Required. An exact published version of the rule, such as `"1.3.0"`. Quote it so YAML reads it as text. |
| `reason` | Required non-blank text explaining why the rule stays at this version. `code-rules project update` shows it next to any newer version, so the decision keeps its context. |

A pin keeps the rule at exactly that version:

- `code-rules project update` never moves a pinned rule. Its preview lists the pin, the newest available version, and your reason.
- When you add or change a pin, `code-rules project sync` moves the rule to the pinned version, up or down.
- Removing a pin doesn't move the rule; the next `code-rules project update` offers its newest version.

To keep a rule at the version you have, pin it to the version recorded in `_source.json`, which also appears in the rule's generated file. `code-rules project update --keep` writes that pin for you.

A pin must name a rule the source imports. A pinned rule that its library retires stays at its version, and update reports the retirement. To keep a rule the library is retiring, pin it before or during the update that retires it; once the retirement is applied, the rule is no longer imported and can't be pinned.

### Import one revision

Set a source's `ref` to import the source exactly as it was at one Git revision. A source whose repository has no `release/<number>` tags, such as one that isn't a library, needs a `ref`; without one, sync fails with `ref-required`:

```yaml
sources:
  vendor-rules:
    repository: https://github.com/example/engineering-rules.git
    ref: release/5
    rules:
      include: ["*"]
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

A source's `rules.exclude` list leaves rules out of the generated guidance. Each entry records why, and an entry for a single rule can name a local rule for agents to read instead:

```yaml
rules:
  include: [techs/typescript/*, practices/*]
  exclude:
    - id: practices/observability/*
      reason: We don't run our own telemetry.
    - id: practices/testing/avoid-snapshot-tests
      reason: Contract snapshots follow our separate review policy.
    - id: techs/typescript/prefer-type-aliases
      reason: Our public extension API relies on declaration merging.
      replacedBy: techs/typescript/prefer-interfaces
      basedOn: "2.0.0"
```

| Field | Meaning |
| --- | --- |
| `id` | Required. A [selector](#select-rules): one rule ID, or a prefix ending in `/*` that excludes every rule it matches, including rules the library adds there later. Each `id` can appear once. When several entries match a rule, the rule's own entry applies, then its group's, then its type's, then `"*"`. |
| `reason` | Required non-blank text explaining why the project leaves the rule out. |
| `replacedBy` | Optional, with a single rule ID only. The ID of an authored rule, under `rules/`, that agents read in place of the excluded rule, such as `techs/typescript/prefer-interfaces` for `rules/techs/typescript/prefer-interfaces.md`. It must be in the same group as the rule it replaces, and each authored rule can replace only one rule. |
| `basedOn` | Optional, and only with `replacedBy`. The library version of the excluded rule that your local rule is based on, in quotes, such as `"2.0.0"`. [Forking a rule](/reference/cli/#fork-a-library-rule) records the forked version here, and so does [replacing a fork with the newest version](/reference/cli/#replace-a-fork-with-the-newest-version). Offline commands check only that it's a version; `code-rules project sync` checks that the library published that version of the rule, and otherwise fails with `version-not-found`, listing the versions it did publish. |

An excluded rule is still imported into `vendor/`, so you can review its changes. The `code-rules project update` preview lists a replaced rule as `replaced` so you can decide whether your local rule needs the same change:

- With `basedOn`, whenever the library's newest version of the rule is newer than `basedOn`, whatever version the project imports. The preview lists every change after `basedOn`. To replace your local rule with a fork of the newest version, overwriting your edits, pass `code-rules project update --update-fork SOURCE:RULE`, or answer `replace` to the terminal question; either sets `basedOn` to that version. If you merge the changes into your rule by hand instead, edit `basedOn` to the version you merged. Code Rules never advances it otherwise, because only you know whether your rule took the changes in.
- Without `basedOn`, such as for a replacement you wrote yourself, whenever the library's newest version of the rule is newer than the version the project imports, also when a pin keeps the imported copy there. The preview lists the changes since the imported version, which your rule may already have. Replacing your rule with a fork, as above, also records `basedOn`.

### Source-scoped exceptions

The `id` of an entry in a source's `rules.exclude` is library-relative, without a source prefix.
For example, an entry with `id: practices/testing/verify-retry-limits` under `sources.fabrica.rules.exclude` affects only Fabrica's rule.
An identically named rule from `acme` remains active.
Full paths distinguish matching filenames in different groups of the same library.

Generated files and review findings retain source-qualified IDs because they appear outside the configuration's source nesting.

A `rules.pins` entry must name a rule this source imports, and a `rules.exclude` entry must match at least one; any other entry fails validation. When the library retires a rule that an exclusion names, or every rule an exclusion's selector matches, the entry no longer does anything. `code-rules project sync` and `code-rules project update` warn about it so you can delete it; nothing else is blocked. Offline, `code-rules project build` and `code-rules project check` accept a pin at the version the source imports, an exclusion of an imported rule, and a pin or exclusion naming a rule the source's [snapshot record](/reference/provenance/#inspect-the-original-imported-files) lists as retired. For any other pin or exclusion, they ask you to run `code-rules project sync`, which checks it against the library. The snapshot record doesn't hold pins, so adding, rewording, or removing a pin that moves no rule never changes it.

## Group selection

Select complete groups separately for each source, add individual rules if needed, then exclude rules when necessary.
Each selected group must exist in the imported library.
A selected group includes rules the library adds to it later.
Selecting `practices/testing` from two sources combines both sets of rules into one effective testing group, with a group page and individual resolved rule files.
Source order never establishes precedence.

Every valid rule file under `rules/` automatically joins its adopted group; individual authored rules do not need source entries.
Local rules can join any imported group. A matching filename does not override an imported rule.
A local rule that an exclusion names as `replacedBy` appears once under its local ID; the excluded rule is removed from active output.
The complete local definition supplies the metadata, guidance, attribution, and assets. Configuration and provenance retain the replacement relationship.
A local `_group.yaml` defines a group, including an empty group, without any configuration entry.
If local metadata exists for an imported group, its complete description and reading cues take precedence for project discovery.
Otherwise, descriptions from every contributing library remain source-labeled.
Provenance retains all group metadata and identifies the sources supplying the effective discovery guidance.
Local rules without either local or imported group metadata are errors, with the missing `_group.yaml` path in the diagnostic. When a sync or update removes the last import supplying an authored rule's group, it writes the group's last imported metadata to `rules/<group-id>/_group.yaml` instead of failing; see [Keep an authored rule's group](/reference/sync/#keep-an-authored-rules-group).
The root `rules/README.md` and each group-root `README.md` are authoring documentation, not rules; other misplaced Markdown files are still validated.
Shared assets that authored rules link to sit in `assets/`, beside `rules/`, so a rule in `rules/practices/testing/` links to one as `../../../assets/retry-lifecycle.svg`. A file in `rules/assets/` is an error.

The generated index shows group names, applicability guidance, and explicit **Open group** links. Without local metadata, guidance from multiple libraries remains labeled by source.
Local metadata supplies the complete project description when present.
The importer does not silently choose one library's description over another's.

## Select individual rules

To import some rules without the rest of their group, list their rule IDs in the source's `rules.include`:

```yaml
sources:
  team:
    repository: https://github.com/acme/.code-rules.git
    rules:
      include:
        - techs/react/*
        - practices/testing/verify-retry-limits
        - techs/go/wrap-errors-with-operation
```

This imports every rule in `techs/react`, including rules added to it later, plus exactly two other rules. Rules the library later adds to `practices/testing` or `techs/go` don't join, because those groups aren't selected.

- Each entry is a library-relative rule ID that must exist in the library.
- An individually selected rule brings its group's metadata, so its group appears in the generated index with only the selected rules.
- The imported rules are the union of all selectors: every rule in the selected groups, plus every listed rule. Listing a rule whose group another selector also selects is allowed and changes nothing; `code-rules project sync` and `code-rules project update` warn about it so you can delete the entry.
- `rules.exclude` applies to every imported rule, however it was selected.
- When the library retires an individually selected rule, `code-rules project update` shows the retirement in its preview. After you confirm, the entry no longer does anything, and later syncs and updates warn about it so you can delete it. To keep the rule instead, pin it to its last version.

Individually selected rules follow updates and pins like any other imported rule.

## Conflicting rules

Source-qualified IDs prevent naming collisions, but distinct rules can still require incompatible behavior in the same situation.
Both remain active unless the project explicitly excludes or replaces a rule under its owning source.
The importer does not infer priority from source order or interpret prose to resolve contradictions.

See [Resolve conflicting rules](/guides/conflicting-guidance/) for examples, an agent review prompt, and ways to resolve competing instructions.

## Library section

The optional `library` section makes the repository a [library](/reference/library-format/) as well as, or instead of, a project. `code-rules library init` adds it. Its only field is `license`, which [License metadata](/reference/library-format/#license-metadata) describes; `library.skills` is rejected, since this version of Code Rules doesn't support skill levels yet:

```yaml
schemaVersion: 2
sources: {}
library:
  license:
    spdxExpression: MIT
    file: LICENSE.md
    notices: []
```

A library publishes only this section of the configuration, never its `sources`. A repository can keep both: project commands read `sources`, library commands read `library`, and both read the groups in `rules/`. See [A repository that is a project and a library](/reference/library-format/#a-repository-that-is-a-project-and-a-library).

## Version boundaries

`schemaVersion` describes this configuration, including its `library` section, and so each library's manifest.
The caller-supplied `toolVersion` identifies the tool that generated the output.

Each source's `rules.pins`, and its `ref` when present, state which rule versions the project wants; every other rule follows its newest version when the project updates.
Vendored provenance records each imported rule's version, library release, and commit, used by offline commands. It is importer-owned output, not a second user-selected version.
A project imports multiple sources directly.
Libraries that themselves inherit and republish other libraries are not supported: a library that is also a project uses its sources itself and publishes only its own rules.

## Version 1 configuration

Code Rules reads only version 2. A configuration with `schemaVersion: 1` fails to load with an example of the version 2 shape, and nothing converts it for you. Rewrite it by hand:

| Version 1 | Version 2 |
| --- | --- |
| `schemaVersion: 1` | `schemaVersion: 2` |
| `groups: [techs/go]` | `rules: {include: [techs/go/*]}`: each group becomes its selector. |
| `groups: "*"`, `"techs/*"`, or `"practices/*"` | The same selector in `rules.include`, such as `include: ["*"]`. |
| `rules: [techs/go/wrap-errors]` | The rule ID in `rules.include`. |
| `exclude:` map keyed by rule ID | `rules.exclude:` list of entries, each with the rule ID as its `id`. |
| `pins:` map keyed by rule ID | `rules.pins:` list of entries, each with the rule ID as its `id`. |
| `replacedBy: local/techs/go/x.md` | `replacedBy: techs/go/x`, the authored rule's ID. |
| `.code-rules/local/<group-id>/` | `.code-rules/rules/<group-id>/` |
| `.code-rules/local/assets/` | `.code-rules/assets/` |

Move the files yourself too: every project command fails while `.code-rules/local/` exists, with a message that shows where each file belongs. Rule IDs don't change. A moved rule sits one directory deeper, so add one `../` to its links to shared assets.

Then run `code-rules project check`. A rewrite that selects the same rules keeps the recorded versions in `vendor/`, so it needs no sync.

See [Library format](/reference/library-format/) for library metadata and [Adapt rules](/guides/select-rules/#adapt-the-import-to-your-project) for examples of exceptions.
