---
title: "Configuration"
description: "Fields in .code-rules/config.yaml."
---

The **project configuration**, `.code-rules/config.yaml`, records the project's sources, the groups, rules, and skills it selects from each, and its exceptions. The **Code Rules directory**, `.code-rules/`, also holds `local/`, `vendor/`, and `generated/`. Installed skills go in [target folders](#install-targets) outside it.

Initialize from the project root. In Git repositories, other project commands find the nearest Git root and use its `.code-rules/config.yaml`. Outside Git, run commands from the project root. See [Working directories](/reference/cli/#working-directories).
A project can import rules and skills directly from multiple canonical libraries, install skills from plain Git repositories, and start from an organization's [preset](#extend-a-preset), choosing versions for each source independently.
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

Repository names, groups, rule IDs, and skill IDs in examples are illustrative.
Replace them with libraries, rules, and skills your project can access.

## Example with skills

```yaml
schemaVersion: 2
skillTargets: [claude, agents]
extends:
  repository: https://github.com/fabricahq/.code-rules.git
  preset: fabrica-defaults
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    groups: [practices/testing]
    skills: [skills/build-feature]
    exclude:
      skills/fix-bug:
        reason: We use our own bug workflow.
        replacedBy: local/skills/fix-bug
        basedOn: "1.0.0"
    pins:
      skills/build-feature:
        version: "1.2.0"
        reason: Waiting on feedback about the new review step.
  pstack:
    repository: https://github.com/cursor/plugins.git
    follow: main
    skillsRoot: pstack/skills
    skills: [skills/how, skills/architect]
    skillUses:
      skills/architect: [skills/how]
```

This project starts from the `fabrica-defaults` preset, which selects `skills/fix-bug` and other skills from the same `fabrica` repository. The project adds `practices/testing` and `skills/build-feature` to the preset's `fabrica` selection, replaces the preset's `skills/fix-bug` with a local fork, and keeps `skills/build-feature` at version 1.2.0. `pstack` is a [plain repository](#plain-repositories): Code Rules installs two of its skills from the newest commit on `main` when the project updates.

## Fields

| Field | Meaning |
| --- | --- |
| `schemaVersion` | Configuration format version: `1` or `2`. Version `2` is required when any skill or preset field appears; see [Schema versions](#schema-versions). |
| `skillTargets` | Optional. Where installed copies of skills go: an array of `claude` (`.claude/skills/`) and `agents` (`.agents/skills/`). Missing means none. See [Install targets](#install-targets). |
| `extends` | Optional. A preset to start from, with `repository`, `preset`, and optional `ref`. See [Extend a preset](#extend-a-preset). |
| `sources` | Map of stable source names to library or plain repository configurations. Use an empty object for a project with only local groups and skills, or one whose sources all come from its preset. |
| `sources.<name>.repository` | Explicit HTTPS or SSH Git address, including scp-style SSH. See [Repository addresses](#repository-addresses). |
| `sources.<name>.groups` | Optional groups to import in full: an array of IDs such as `techs/typescript`, or `"*"`, `"practices/*"`, or `"techs/*"` to select all groups in that scope. Libraries only. |
| `sources.<name>.rules` | Optional individual rules to import without the rest of their group: an array of library rule IDs. Libraries only. See [Select individual rules](#select-individual-rules). |
| `sources.<name>.skills` | Optional skills to install: an array of skill IDs such as `skills/build-feature`, or `"*"` for every skill in the source. See [Select skills](#select-skills). |
| `sources.<name>.exclude` | Optional map of this source's rule or skill IDs to a `reason` and, optionally, a local rule or skill that `replacedBy` names to use instead, with the version it's `basedOn`. See [Exclude or replace a rule](#exclude-or-replace-a-rule). |
| `sources.<name>.pins` | Optional map of this library's rule or skill IDs to an exact `version` and a `reason`. Libraries only. See [Pin a rule](#pin-a-rule). |
| `sources.<name>.ref` | Optional and advanced for a library: import it exactly as it was at one tag or commit, and can't be combined with `pins`; see [Import one revision](#import-one-revision). For a plain repository, `ref` or `follow` is required; see [Plain repositories](#plain-repositories). |
| `sources.<name>.follow` | Plain repositories only. A branch to track; `code-rules project update` moves the source to the branch's newest commit. |
| `sources.<name>.skillsRoot` | Plain repositories only. The folder that holds the repository's skill folders. Defaults to `skills`. |
| `sources.<name>.skillUses` | Optional map of a skill ID to the skill IDs it uses that the skill itself doesn't declare. See [Declare a skill's dependencies](#declare-a-skills-dependencies). |

Unknown configuration fields are rejected, including unknown source and exclusion fields.
Local groups are discovered from `local/<group-id>/_group.yaml`; no source entry or separate group list is required.
The former `localGroups` field is rejected with migration guidance. Remove it and keep the group metadata files.
`exclude` is optional; omit it when a source has no exceptions.
Each source must select at least one group, rule, or skill, counting what its [preset](#extend-a-preset) selects under the same source name.

Each source owns its selection, pins, and exceptions.
The earlier singular `source` and top-level `groups`, `exclude`, and `replace` fields are not part of this format.
Sources have no `version` or `replace` field. Rule and skill versions are chosen one by one; use `pins` to keep individual rules or skills at exact versions, and `exclude` with `replacedBy` to replace one.

### Schema versions

| Version | Accepts |
| --- | --- |
| `1` | Every field above except `skillTargets`, `extends`, and a source's `skills`, `follow`, `skillsRoot`, and `skillUses`. |
| `2` | Everything version 1 does, plus those fields. |

A version 1 configuration that uses a version 2 field fails and names the field; set `schemaVersion: 2`. Code Rules writes `2` only when it writes one of those fields, such as when `code-rules project init` writes `skillTargets`, or `code-rules project add library --skills` adds a source's `skills`, so a project that uses only rules keeps version 1 when you upgrade Code Rules, and stays readable by earlier versions. It never lowers the version. `code-rules project init --skill-targets none` writes no `skillTargets`, and so writes version 1.

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

Select at least one group, individual rule, or skill. For `groups`, use one supported selector string or an explicit array of group IDs. Wildcard arrays, mixed selectors, and arbitrary globs such as `techs/**` are unsupported.
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
Names must match `[a-z][a-z0-9-]*`; `local` is reserved for project-authored rules and skills.
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

Skills follow the same pattern. A skill ID is its path in the library without the trailing `/SKILL.md`, such as `skills/build-feature`, and an imported skill's project ID is `<source-name>:<skill-id>`:

```text
fabrica:skills/build-feature
pstack:skills/architect
local:skills/fix-bug
```

In a plain repository, a skill's ID is `skills/<name>` whatever its [`skillsRoot`](#plain-repositories), where `<name>` is the skill's folder name under that root. Local skills use `local:skills/<name>`. See [Skill IDs](/reference/skill-format/#skill-ids) for the name grammar.

Project IDs keep skills from different sources distinct in configuration, generated files, and provenance. The installed copy, though, goes in a folder named only `<name>`, because agents find skills by name. Two skills with the same name, from any sources or your local skills, fail with `skill-name-conflict`; exclude one of them. See [Who owns the target folders](/reference/sync/#who-owns-the-target-folders).

## Choose versions

Each rule and skill in a library has its own [version](/reference/rule-versions/). By default, a project follows each one's newest version, but only when someone updates:

- `vendor/<source-name>/_source.json` records the exact version of every imported rule and skill, like a lockfile. [`code-rules project sync`](/reference/cli/#project-sync) restores those versions, so every checkout imports the same content.
- [`code-rules project update`](/reference/cli/#project-update) previews newer versions, new rules, and retirements, and applies them after you confirm.

Two optional source fields change that:

| Field | Use it to |
| --- | --- |
| `pins` | Keep individual rules or skills at exact versions while the rest follow updates. |
| `ref` | Import the library exactly as it was at one Git revision. Advanced. |

A source can use `pins` or `ref`, not both. Neither changes which rules and skills are imported; `groups`, `rules`, and `skills` do that.

Plain repositories have no versions. They follow a branch or stay on one revision instead; see [Plain repositories](#plain-repositories).

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

Pin a skill the same way, by its skill ID:

```yaml
pins:
  skills/build-feature:
    version: "1.2.0"
    reason: Waiting on feedback about the new review step.
```

A pinned skill's version decides its whole folder, the skills it uses, and the rules it cites: Code Rules reads its `cr-skills-used` and `cr-rules-used` lists from the pinned version. The skills it uses and the rules it cites still follow their own versions, so a pinned skill can run with newer versions of them than it was published with.

A pin keeps the rule or skill at exactly that version:

- `code-rules project update` never moves a pinned rule. Its preview lists the pin, the newest available version, and your reason.
- When you add or change a pin, `code-rules project sync` moves the rule to the pinned version, up or down.
- Removing a pin doesn't move the rule; the next `code-rules project update` offers its newest version.

To keep a rule at the version you have, pin it to the version recorded in `_source.json`, which also appears in the rule's generated file, or, for a skill, in its installed copy's marker line. `code-rules project update --keep` writes that pin for you.

A pin must name a rule or skill the source imports. A pinned rule that its library retires stays at its version, and update reports the retirement. To keep a rule the library is retiring, pin it before or during the update that retires it; once the retirement is applied, the rule is no longer imported and can't be pinned.

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

- **A [library release](/reference/rule-versions/#library-releases)**, which is the usual case. Every selected rule and skill is imported at the version that library release published, as recorded in its release record.
- **Any other commit or tag**, such as unreleased changes a library author wants to test in a real project, or a library that hasn't published its first library release. Rules and skills whose files match a published version record that version. Those with unreleased changes record their version as `null`, and generated guidance and installed copies show no version for them.

Importing a revision other than a library release opts that source out of versions: rules and skills with unreleased changes have no version to cite, and no update preview reviews their changes. To keep that from shipping by accident, `code-rules project sync` and `code-rules project update` print a warning naming the source and its unreleased rules and skills, and the generated library summary in `generated/libraries/<source-name>/README.md` says the source is imported from unreleased changes. `code-rules project check` still passes.

`code-rules project update` doesn't move a source that uses `ref`. To import another revision, change `ref` and run `code-rules project sync`. To go back to following versions, remove `ref` and run `code-rules project sync`: rules keep their recorded versions when those are published versions, and the rest get their newest version.

### Where each rule's files come from

Each rule's [Markdown file and asset directory](/reference/rule-versions/#what-a-version-covers) come from the library release that published its version. Each skill's whole folder comes from the library release that published its version. Library-wide files, such as group metadata, shared assets, and license files, come from one library release, recorded as `release` in the source's [snapshot record](/reference/provenance/#inspect-the-original-imported-files), or from the revision your `ref` names. A new source, and newly selected groups or rules, take them from the newest library release, and `code-rules project update` moves them to the newest library release; `code-rules project sync` keeps the recorded one. That library release is never older than the one that published any imported rule version. When that library release no longer has the metadata of an imported rule's group, such as the group of a retired rule you kept, the group's metadata comes from the newest library release among its imported rule versions that still has it.

Offline `code-rules project build`, `code-rules project check`, and ordinary agent work use the recorded versions without contacting the repository.

## Exclude or replace a rule

A source's `exclude` map leaves rules out of the generated guidance, and skills out of the target folders. Each entry records why, and can name a local rule or skill for agents to use instead:

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
| `replacedBy` | Optional. A local rule file that agents read in place of the excluded rule. The path is relative to the Code Rules directory, must stay under `local/`, and must be in the same group as the rule it replaces. Each local file can replace only one rule. For a skill, a local skill folder, `local/skills/<name>`; see [Exclude or replace a skill](#exclude-or-replace-a-skill). |
| `basedOn` | Optional, and only with `replacedBy`. The library version of the excluded rule or skill that your local rule or skill is based on, in quotes, such as `"2.0.0"`. [Forking a rule](/reference/cli/#fork-a-library-rule) records the forked version here, and so does [replacing a fork with the newest version](/reference/cli/#replace-a-fork-with-the-newest-version). Offline commands check only that it's a version; `code-rules project sync` checks that the library published that version of the rule, and otherwise fails with `version-not-found`, listing the versions it did publish. |

An excluded rule or skill is still imported into `vendor/`, so you can review its changes. The `code-rules project update` preview lists a replaced rule as `replaced` so you can decide whether your local rule needs the same change:

- With `basedOn`, whenever the library's newest version of the rule is newer than `basedOn`, whatever version the project imports. The preview lists every change after `basedOn`. To replace your local rule with a fork of the newest version, overwriting your edits, pass `code-rules project update --update-fork SOURCE:RULE`, or answer `replace` to the terminal question; either sets `basedOn` to that version. If you merge the changes into your rule by hand instead, edit `basedOn` to the version you merged. Code Rules never advances it otherwise, because only you know whether your rule took the changes in.
- Without `basedOn`, such as for a replacement you wrote yourself, whenever the library's newest version of the rule is newer than the version the project imports, also when a pin keeps the imported copy there. The preview lists the changes since the imported version, which your rule may already have. Replacing your rule with a fork, as above, also records `basedOn`.

### Exclude or replace a skill

```yaml
exclude:
  skills/write-changelog:
    reason: Release Planner writes our release notes.
  skills/fix-bug:
    reason: We use our own bug workflow.
    replacedBy: local/skills/fix-bug
    basedOn: "1.0.0"
```

An excluded skill isn't installed into the target folders or listed in `generated/SKILLS.md`. A replacement is a local skill folder under `local/skills/`, installed in place of the excluded skill under its own folder name. Its name can differ from the excluded skill's. Each local skill can replace only one skill, and a rule can't replace a skill or the reverse. [Forking a skill](/reference/cli/#project-add-skill) writes the exclusion for you, with the forked version as `basedOn`. For a skill from a plain repository, which has no versions, `basedOn` is the full 40-character commit SHA the fork copied.

When another installed skill uses a skill you exclude without a replacement, `code-rules project sync` warns that the skill it uses isn't installed. With a replacement, the skill that uses it lists your local skill instead in its installed copy's "Skills this skill uses" line.

You can also exclude a rule that an installed skill cites, such as one from a group you don't select. The skill's installed copy then says the rule is excluded by this project, with your reason; see [Select skills](#select-skills).

### Source-scoped exceptions

Keys inside a source's `exclude` use the library-relative rule ID, without a source prefix.
For example, `sources.fabrica.exclude["practices/testing/verify-retry-limits"]` affects only Fabrica's rule.
An identically named rule from `acme` remains active.
Full paths distinguish matching filenames in different groups of the same library.

Generated files and review findings retain source-qualified IDs because they appear outside the configuration's source nesting.

An `exclude` or `pins` key must name a rule or skill this source imports: through its groups or its `rules` list, through its `skills` list, as a skill an installed skill uses, or as a rule an installed skill cites. Any other ID fails validation. When the library retires a rule or skill that an exclusion names, the entry no longer does anything. `code-rules project sync` and `code-rules project update` warn about it so you can delete it; nothing else is blocked. Offline, `code-rules project build` and `code-rules project check` accept a pin at the version the source imports, an exclusion of an imported rule, and a pin or exclusion naming a rule the source's [snapshot record](/reference/provenance/#inspect-the-original-imported-files) lists as retired. For any other pin or exclusion, they ask you to run `code-rules project sync`, which checks it against the library. The snapshot record doesn't hold pins, so adding, rewording, or removing a pin that moves no rule never changes it.

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

## Select skills

List the skills to install under the source's `skills`:

```yaml
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    groups:
      - practices/testing
    skills:
      - skills/build-feature
      - skills/review-change
```

Or set `skills` to the string `"*"` to install every skill in the source, including skills the library or repository adds later, when the project updates. `"*"` is the only selector; patterns such as `"skills/review-*"` are rejected.

- Each entry is a skill ID that must exist in the source.
- Skills don't belong to groups. Selecting a group never installs a skill, and selecting a skill never selects a group.
- A source can select only skills: `groups` and `rules` are optional when `skills` is set.

When you install a skill, Code Rules also installs:

- **The skills it uses**, transitively: every skill named in its `cr-skills-used` list, the skills those use, and so on, from the same source, plus any your [`skillUses`](#declare-a-skills-dependencies) adds. These are recorded as dependencies, not selections. A skill that a selected skill uses, and that you also list, counts as selected.
- **The rules it cites**: every rule in each installed library skill's `cr-rules-used` list, from the same library, even from groups the project doesn't select. These **cited rules** are reference-only: each gets its generated rule file, and the installed skill links to it, but `generated/RULES.md` and the group pages don't list it, so installing a skill never widens the rules that apply to all of the project's work. A cited rule that the project imports anyway, through its groups or `rules`, is an ordinary imported rule. Cited rules follow updates, pins, and exclusions like any other imported rule.

Local skills cite rules by project ID, such as `fabrica:practices/testing/test-changed-behavior` or `local:practices/testing/test-project-contracts`, and name the skills they use by project ID too, such as `fabrica:skills/tdd`. Each ID must name a configured source, or `local`. Code Rules imports those rules as cited rules, and installs those skills as dependencies, from the named source. [Forking a skill](/reference/cli/#project-add-skill) rewrites its lists to project IDs for you. See [Skill format](/reference/skill-format/#metadata).

### Declare a skill's dependencies

A skill may use other skills without listing them in `cr-skills-used`, especially one from a [plain repository](#plain-repositories), which has no such metadata. Declare those dependencies in the source's `skillUses`:

```yaml
skillUses:
  skills/architect:
    - skills/how
```

Each key is a skill ID the source installs, and each value lists skill IDs in the same source. Every ID must exist in the source. `skillUses` adds to what the skill declares itself; it can't remove a dependency the skill declares. Code Rules installs the listed skills as dependencies, and the installed copy's "Skills this skill uses" line includes them. Dependencies can't form a cycle.

## Install targets

`skillTargets` says where installed copies of skills go:

| Value | Folder | Read by |
| --- | --- | --- |
| `claude` | `.claude/skills/` at the project root | Claude Code |
| `agents` | `.agents/skills/` at the project root | Codex, Cursor, and other agents that read the shared folder |

```yaml
schemaVersion: 2
skillTargets: [claude, agents]
```

`code-rules project init` writes `[claude, agents]`; `--skill-targets claude`, `agents`, or `none` changes it. Each value can appear once. A missing `skillTargets` means no targets, so upgrading Code Rules never starts writing into these folders, and `skillTargets: []` means the same. With no targets, skills are still vendored, listed in `generated/SKILLS.md`, and checked; agents read them through `SKILLS.md`.

Each installed skill gets a folder `<target>/<name>/`. Code Rules writes and removes only the skill folders it installed, and leaves every other folder in the target alone, such as a skill another tool installed. See [Who owns the target folders](/reference/sync/#who-owns-the-target-folders).

When a preset sets `skillTargets`, the project's `skillTargets` wins when the project sets it, including to `[]`.

## Plain repositories

A **plain repository** is any Git repository that holds skill folders but no `rule-library.yaml`, such as a third party's collection of skills. Code Rules decides whether a source is a library or a plain repository by whether `rule-library.yaml` exists at the revision it reads.

```yaml
sources:
  pstack:
    repository: https://github.com/cursor/plugins.git
    follow: main
    skillsRoot: pstack/skills
    skills:
      - skills/how
      - skills/architect
```

| | Library | Plain repository |
| --- | --- | --- |
| Skills | `skills/` at the library root | Under `skillsRoot` |
| Selection | `groups`, `rules`, and `skills` | `skills` only |
| Versions | Per skill, from library releases | None; `_source.json` records the exact commit |
| Update | Previews each skill's new version and summaries | With `follow`, previews added, changed, and removed skill folders between the recorded and newest commits; with `ref`, never moves |
| Validation | Strict, as `code-rules library check` applies | Lenient: violations of the Agent Skills format are warnings, and the skill installs as it is |
| Licenses | From `rule-library.yaml` | Found in the skill or the repository; see [Plain repositories](/reference/imports/#plain-repositories) |
| Dependencies | `cr-skills-used`, plus `skillUses` | `skillUses` |

A plain repository source:

- Sets exactly one of `follow` and `ref`. `follow` names a branch; `code-rules project update` moves the source to the branch's newest commit, and `code-rules project sync` keeps the commit recorded in `_source.json`. `ref` names a tag or full commit SHA, as for a library, and never moves.
- Selects skills with `skills`, which is required. `groups`, `rules`, and `pins` are rejected, because a plain repository has no rules or versions.
- Can set `skillsRoot`, the folder holding its skill folders, relative to the repository root, such as `pstack/skills`, or `.` for the root itself. It defaults to `skills`. Each skill is a folder directly inside it with a `SKILL.md` file. The path can't contain `..` segments.
- Can use `exclude` and `skillUses`.

`follow` and `skillsRoot` are rejected on a library. When a repository gains or loses `rule-library.yaml`, so that a source's fields no longer fit it, `code-rules project update` fails and names the fields to change.

## Extend a preset

A **preset** is a starting configuration that a library publishes in `presets/<name>.yaml`, such as an organization's default libraries, groups, and skills. A project starts from one with `extends`:

```yaml
schemaVersion: 2
extends:
  repository: https://github.com/fabricahq/.code-rules.git
  preset: fabrica-defaults
sources: {}
```

| Field | Meaning |
| --- | --- |
| `repository` | Required. The library that publishes the preset, as a [repository address](#repository-addresses). |
| `preset` | Required. The preset's name: its file name in `presets/`, without `.yaml`. |
| `ref` | Optional. A tag or full commit SHA, as for a source's [`ref`](#import-one-revision), that keeps the preset at one revision. |

Code Rules reads the preset from the library's newest library release, records that library release in `vendor/_preset/_source.json`, and keeps it there on `code-rules project sync`. `code-rules project update` moves it to the newest library release and previews what the preset added or removed; with `ref`, it stays put.

The preset's configuration combines with the project's:

- Each source the preset declares is added under its own name.
- When the project declares a source with the same name, its `repository` must match the preset's. Its selections, `groups`, `rules`, and `skills`, are added to the preset's. Its `exclude`, `pins`, and `skillUses` entries win over the preset's for the same key, and its `ref`, `follow`, and `skillsRoot` win when set.
- The project's `skillTargets` wins when set.
- The project can't remove a preset's selection or exclusion. To diverge that far, stop extending the preset: copy the sources you want into the project's configuration, remove `extends`, and run `code-rules project sync`.

Each rule and skill still comes from its own source, with its own version, credit, and license. A preset only selects; it never republishes another library's rules or skills.

A preset can't extend another preset. For the preset file's format and how to publish one, see [Share defaults with presets](/guides/presets/).

## Conflicting rules

Source-qualified IDs prevent naming collisions, but distinct rules can still require incompatible behavior in the same situation.
Both remain active unless the project explicitly excludes or replaces a rule under its owning source.
The importer does not infer priority from source order or interpret prose to resolve contradictions.

See [Resolve conflicting rules](/guides/conflicting-guidance/) for examples, an agent review prompt, and ways to resolve competing instructions.

## Version boundaries

`schemaVersion` describes this configuration.
Each library's `formatVersion` describes its authoring format.
The caller-supplied `toolVersion` identifies the tool that generated the output.

Each source's `pins`, and its `ref` or `follow` when present, state which versions the project wants; every other rule and skill follows its newest version when the project updates.
Vendored provenance records each imported rule's and skill's version, library release, and commit, or a plain repository's commit, used by offline commands. It is importer-owned output, not a second user-selected version.
A project imports multiple sources directly, and can start from one [preset](#extend-a-preset).
A preset selects sources, groups, rules, and skills; it never republishes them. Each comes from its own library or repository, and a preset can't extend another preset.

See [Library format](/reference/library-format/) for library metadata and [Adapt rules](/guides/select-rules/#adapt-the-import-to-your-project) for examples of exceptions.
