---
title: "CLI commands"
description: "Syntax, arguments, options, and results for every supported Code Rules command."
---

Find each command's syntax and options below. Every command also accepts `--json` and `-h` / `--help`; [shared behavior](#shared-options-and-prompts) and [output formats](#command-output) are documented at the end.

## Project commands

`code-rules project` manages rules and skills for a **project**, the codebase whose rules and skills Code Rules manages. Initialize from the project root. In Git repositories, other project commands also work from subdirectories.

Configuration lives in `.code-rules/config.yaml` at the project root. See [Working directories](#working-directories) for repository discovery and input-file paths.

### project init

```sh
code-rules project init [options]
```

Create project configuration, the local rules directory, and the managed Code Rules guide. Rerunning init preserves valid configuration and local rules and refreshes an unmodified guide.

| Option | Meaning |
| --- | --- |
| `--skill-targets TARGET` | Where to install skills: `claude` for `.claude/skills/`, `agents` for `.agents/skills/`, or `none`. Repeat for both targets. `none` can't be combined with a target. |
| `--non-interactive` | Accepted; init runs without prompts even when this flag is omitted. |

A new configuration gets `skillTargets: [claude, agents]` and `schemaVersion: 2`. With `--skill-targets none`, init leaves `skillTargets` out and writes `schemaVersion: 1`, so the configuration stays readable by earlier versions of Code Rules; see [Schema versions](/reference/configuration/#schema-versions). Init creates no target folders: sync writes them when the project first installs a skill.

Rerunning init on an existing configuration changes `skillTargets` only when you pass `--skill-targets`, and `none` then removes it. Without the option, init leaves a configuration that has no `skillTargets` without one, so upgrading Code Rules never starts installing skills into a project that didn't choose targets.

Init does not fetch libraries or generate rule guidance. It refuses to overwrite a manually edited Code Rules guide. For file locations and ownership, see [Project and group guides](/reference/files/#project-and-group-guides).

### project add library

```sh
code-rules project add library ALIAS [options]
```

Record a library in project configuration without fetching it. `ALIAS` is the source name, such as `team`. Run `code-rules project sync` afterward to import the selected rules and install the selected skills. To add a Git repository of skills that has no `rule-library.yaml`, use [project add repository](#project-add-repository) instead.

| Option | Meaning |
| --- | --- |
| `--repository URL` | Required. An accepted HTTPS or SSH [Git repository address](/reference/configuration/#repository-addresses). |
| `--groups GROUP` | Groups to import in full. Repeat for multiple group IDs, or supply one selector: `*`, `practices/*`, or `techs/*`. Quote wildcard values so your shell does not expand them. |
| `--rules RULE` | Individual rules to import without the rest of their group, such as `practices/testing/verify-retry-limits`. Repeat for multiple rules. |
| `--skills SKILL` | Skills to install, such as `skills/build-feature`, together with the skills they use and the rules they cite. Repeat for multiple skills, or supply `*` for every skill in the library. Quote `*` so your shell does not expand it. |
| `--ref REF` | Optional and advanced. Import the library exactly as it was at one revision: a tag, such as the library release tag `release/5`, or a full commit SHA. Recorded as the source's `ref`. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Supply at least one `--groups`, `--rules`, or `--skills`. `--skills` records the source's `skills`, a version 2 field, so the command also sets `schemaVersion: 2`. Without `--ref`, the source follows each rule's newest version when the project updates. To pin individual rules, add `pins` to the source in configuration, or use `code-rules project update --keep`; see [Choose versions](/reference/configuration/#choose-versions).

Library addition preserves existing source exceptions and local files. Pass each group or skill ID as a separate option, rather than a comma-separated flag value:

```sh
code-rules project add library team \
  --repository https://github.com/example/rules.git \
  --groups practices/testing \
  --groups techs/typescript \
  --skills skills/build-feature \
  --non-interactive
```

The repository address, groups, and skill are illustrative; replace them with a library you can access.

### project add repository

```sh
code-rules project add repository ALIAS [options]
```

Record a **plain repository**, a Git repository of skill folders without `rule-library.yaml`, in project configuration without fetching it. `ALIAS` is the source name, such as `pstack`. Run `code-rules project sync` afterward to install the selected skills. See [Configuration](/reference/configuration/#plain-repositories).

| Option | Meaning |
| --- | --- |
| `--repository URL` | Required. An accepted HTTPS or SSH [Git repository address](/reference/configuration/#repository-addresses). |
| `--ref REF` | Required. A branch to track, such as `main`, or one revision to stay on: a tag or a full commit SHA. Recorded as the source's `ref`. |
| `--skills-root PATH` | The folder that holds the repository's skill folders, relative to the repository root, such as `pstack/skills`, or `.` for the root itself. Defaults to `skills`, which isn't recorded. |
| `--skills SKILL` | Required. Skills to install, such as `skills/how`, together with the skills that `skillUses` says they use. Repeat for multiple skills, or supply `*` for every skill in the skills root. Quote `*` so your shell does not expand it. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

With a branch, sync installs the commit recorded in `_source.json`, and `code-rules project update` moves the source to the branch's newest commit; a tag or commit never moves. See [Plain repositories](/reference/configuration/#plain-repositories). A skill's ID is `skills/<folder-name>`, whatever the skills root: with `--skills-root pstack/skills`, `skills/how` is the folder `pstack/skills/how/`. The command writes `schemaVersion: 2`.

```sh
code-rules project add repository pstack \
  --repository https://github.com/cursor/plugins.git \
  --ref main \
  --skills-root pstack/skills \
  --skills skills/how \
  --skills skills/architect \
  --non-interactive
```

A plain repository has no rules or versions, so the command takes no `--groups` or `--rules`, and pins don't apply. To declare skills a skill uses without saying so in its `SKILL.md`, add [`skillUses`](/reference/configuration/#declare-a-skills-dependencies) to the source. Sync reads the repository and checks that it has no `rule-library.yaml`; if it has one, sync fails and asks you to add it with `code-rules project add library` instead. A `--ref` that names no branch, tag, or commit fails sync with `version-not-found`, and a skill with no license terms fails it with `license-not-found`; see [License rules](/guides/license-rules/#license-skills). Existing sources aren't overwritten.

### project add group

```sh
code-rules project add group ID [options]
```

Create a local group with `_group.yaml` metadata and an authoring README. `ID` is a group path such as `practices/testing` or `techs/typescript`.

| Option | Meaning |
| --- | --- |
| `--name TEXT` | Required. Group display name. |
| `--description TEXT` | Required. What the group covers. |
| `--when-to-read TEXT` | Required. When an agent should read the group. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

A group can exist before it has rules. Local groups do not need a library source declaration. Existing group files are preserved.

### project add rule

```sh
code-rules project add rule ID [options]
```

Create a rule in an existing local group. `ID` includes the group path and rule name, such as `practices/testing/check-retries`, without `.md`.

| Option | Meaning |
| --- | --- |
| `--title TEXT` | Required. Action-oriented rule title. |
| `--when-to-read TEXT` | Required. When an agent should read the rule. |
| `--impact LEVEL` | Required. One of `CRITICAL`, `HIGH`, `MEDIUM-HIGH`, `MEDIUM`, `LOW-MEDIUM`, or `LOW`. |
| `--impact-description TEXT` | Required. The consequence the rule helps prevent. |
| `--body-file PATH` | Optional UTF-8 Markdown body, without frontmatter. Relative paths start at your working directory. Omit to create an unfinished draft. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Create the group first with `code-rules project add group`. A missing group fails before metadata prompts. Without `--body-file`, complete the [template](/reference/rule-authoring/#markdown-template) and remove its `code-rules:draft` marker before building. Existing rules are not overwritten.

#### Fork a library rule

```sh
code-rules project add rule ID --from LIBRARY@VERSION [options]
```

Copy one version of a library rule into `local/` so the project controls its text. Use a fork to change what a rule says. To import one rule without the rest of its group, [select it individually](/reference/configuration/#select-individual-rules) instead, and to keep an imported rule at an older version, [pin it](/reference/configuration/#pin-a-rule); both keep the rule's identity and update reports. `ID` is the library rule ID, such as `practices/testing/verify-retry-limits`, and the fork keeps it: `local/practices/testing/verify-retry-limits.md`.

| Option | Meaning |
| --- | --- |
| `--from LIBRARY@VERSION` | Required for a fork. `LIBRARY` is a configured source name, such as `team`, or a [repository address](/reference/configuration/#repository-addresses). `VERSION` is one of the rule's versions, such as `1.3.0`. |
| `--reason TEXT` | Why the project uses the fork instead of the imported rule. Required when the fork adds an exclusion, as in step 4 below, and refused otherwise. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Metadata options and `--body-file` don't apply to a fork. The command:

1. Finds the library release that published `<VERSION>` from the release records in the library's `release/<number>` tags, and reads the rule and its asset directory at that tag's commit.
2. Writes them under `local/`. When neither local metadata nor a library the project imports supplies the rule's group, it also creates the local group from the library's group metadata.
3. Adds an `attribution` entry that links to the rule at the tag's commit. Libraries hosted outside GitHub.com and GitLab.com get an attribution entry only when the repository address is an HTTPS URL, and the entry links to that address.
4. When the configuration of the `LIBRARY` source selects the rule and its last sync imported it, adds an `exclude` entry for it with your reason, the fork as `replacedBy`, and the forked version as [`basedOn`](/reference/configuration/#exclude-or-replace-a-rule), so agents read only the fork, and updates list the library's later changes to it. If the source pins the rule, the same edit removes the pin and warns that it did: the fork, not a pinned import, now decides what agents read. If you changed that source's configuration since its last sync, run `code-rules project sync` first.

If the rule links to files in the library's shared `assets/` directory, the fork copies them into its own asset directory and updates the links, because local rules can't depend on library files. For the same reason, the command refuses a fork whose links would point outside the copied files, such as links to declared license or notice files, or raw HTML links. Existing local rules and exclusions are never overwritten. A forked rule has no version; it changes only when you edit it. Run `code-rules project build` afterward. The library's license still applies to the copied text; see [License rules](/guides/license-rules/).

### project add skill

```sh
code-rules project add skill NAME [options]
```

Create a local skill in `.code-rules/local/skills/NAME/`. `NAME` follows the [skill name grammar](/reference/skill-format/#skill-ids), such as `release-checklist`; the skill's ID is `local:skills/NAME`.

| Option | Meaning |
| --- | --- |
| `--description TEXT` | Required. What the skill does and when to use it, at most 1,024 characters. |
| `--type TYPE` | Optional. `playbook` or `review`, recorded as `cr-skill-type`. See [Skill types](/concepts/skill/#skill-types). |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

The command writes `SKILL.md` with the name, description, and type in its frontmatter, and a body with the `code-rules:draft` marker. Complete it and remove the marker before building; a local skill that still has the marker fails sync and build. Local skills cite rules and skills by project ID, such as `fabrica:practices/testing/verify-retry-limits`; see [Local skills](/reference/skill-format/#local-skills). Existing local skills are not overwritten. When a skill the project installs from a source already has the name, the command warns that sync will fail with `skill-name-conflict` until you exclude one of them.

#### Fork a skill

```sh
code-rules project add skill --from SOURCE:SKILL [options]
```

Copy a skill the project installs into `local/skills/` so the project controls its steps. `SOURCE:SKILL` names a configured source and the skill's ID in it, such as `fabrica:skills/fix-bug`. The fork keeps the skill's name, so leave out `NAME`.

| Option | Meaning |
| --- | --- |
| `--from SOURCE:SKILL` | Required for a fork. A skill that the source's last sync installed, whether the project selected it or another skill uses it. |
| `--reason TEXT` | Required. Why the project uses the fork instead of the imported skill, recorded with the exclusion. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

`--description` and `--type` don't apply to a fork. The command:

1. Copies the skill's folder from `vendor/`, exactly as the project installs it, with every file's executable bit, to `local/skills/<name>/`: the version recorded in `vendor/<source-name>/_source.json`, or, for a plain repository, the recorded commit.
2. Rewrites the fork's `cr-rules-used` and `cr-skills-used` to project IDs, such as `fabrica:practices/testing/verify-retry-limits`, because a local skill cites by project ID. For a plain repository, it writes the skills that `skillUses` lists for the skill into `cr-skills-used`.
3. Adds an `exclude` entry for the skill with your reason, `local/skills/<name>` as `replacedBy`, and the forked version as `basedOn`, or, for a plain repository, the full commit SHA. If the source pins the skill, the same edit removes the pin and warns that it did. If you changed that source's configuration since its last sync, run `code-rules project sync` first.

Links in a skill stay inside its folder, so the fork needs no link changes. Existing local skills and exclusions are never overwritten. A forked skill has no version; it changes only when you edit it. Run `code-rules project build` afterward. The source's license still applies to the copied files; see [License rules](/guides/license-rules/#license-skills).

### project sync

```sh
code-rules project sync [options]
```

Import the rule and skill versions recorded for each source, validate their files, and replace `vendor/` and `generated/` in the Code Rules directory with the complete result. Sync also writes each installed skill's folder into the project's install targets, removes the folders of skills it no longer installs, and refreshes an older, unedited managed Code Rules guide.

Accepts the [shared options](#shared-options-and-prompts) only.

Sync needs access to every configured repository and uses your existing Git credentials. It never moves a rule or skill to a newer version on its own: it restores the version recorded in `vendor/<source-name>/_source.json`, or, for a plain repository, the recorded commit. Sync chooses a version only when configuration asks for something the record doesn't have:

| Situation | Version sync imports |
| --- | --- |
| A new source, a changed repository, or a newly selected group, rule, or skill | Each new rule's and skill's newest version, and the library-wide files from the newest library release. For a new plain repository whose `ref` names a branch, the branch's newest commit. |
| A pin added or changed | The pinned version, up or down. |
| A pin removed | The recorded version, unchanged. The next update offers newer versions. |
| `ref` added or changed | The rule as it was at that revision: the version a library release published, or `null` for unreleased changes. |
| `ref` removed | The recorded version when it's a published version; otherwise the newest version. |

When you change `groups`, a group is newly selected if the record didn't import it as a whole, even when the previous value matched it. For example, after a sync with `groups: "*"`, a group that a later library release adds is newly selected once you change `groups` to a list that names it.

Sync trusts only a record it wrote: a record whose [checksum](/reference/provenance/#inspect-the-original-imported-files) doesn't match, or that has none, such as after a hand edit or a merge resolution, fails, and sync writes nothing and never records it again. To recover, restore a record sync wrote with `git checkout -- .code-rules/vendor/<source-name>/_source.json`, or, during a merge conflict, take one side with `git checkout --ours` or `git checkout --theirs` for that file, then run `code-rules project sync`, which applies any configuration changes; or delete `.code-rules/vendor/<source-name>/` and run `code-rules project sync`, which imports the source again, with its unpinned rules at their newest versions. Before importing from a record it wrote, sync also checks it against the library's release records: each recorded rule version must be the version its recorded library release publishes, and each recorded library release must exist. A source using `ref` keeps the commit its `ref` named when it was recorded, even after the tag moves or is gone, so sync checks it against the release records only when choosing versions reads them anyway.

Sync installs each selected skill, every skill it uses, transitively, and every rule an installed library skill cites; see [Select skills](/reference/configuration/#select-skills). It writes a skill folder in a target only when `generated/installed-skills.json` records that Code Rules installed it there, or when no folder with that name exists. It fails, writing nothing, with `skill-folder-not-managed` when a folder with an installed skill's name exists in a target that Code Rules didn't install, and with `skill-name-conflict` when two skills would install under the same name, from any sources, local skills, or both; exclude one of them. See [Who owns the target folders](/reference/sync/#who-owns-the-target-folders). A preset that its library doesn't publish fails with `preset-not-found`.

These changes come from edits you made to configuration, so sync applies them without a preview. To move rules to newer versions, use [project update](#project-update). If a source fails, the previous complete output is preserved. For recovery behavior, see [Sync and recovery](/reference/sync/).

When a source imports unreleased changes through `ref`, sync prints a warning naming the source and its unreleased rules and skills. When an exclusion, individually selected rule, or selected skill names a rule or skill the library retired, or a skill folder a plain repository no longer has, sync warns that the entry no longer does anything. It also warns about an individually selected rule whose group `groups` already selects, since the entry changes nothing. For a plain repository, it warns about each place a skill breaks the [skill format](/reference/skill-format/), and installs the skill as it is. When a library release has no `skills` field because an older Code Rules published it, sync warns that skill versions come from an earlier library release; see [Skills in the release record](/reference/rule-versions/#skills-in-the-release-record).

### project update

```sh
code-rules project update [SOURCE | SOURCE:RULE ...] [options]
```

Preview newer rule and skill versions, new rules and skills, and retirements, then apply them after you confirm, and regenerate guidance and installed skills like sync. With no arguments, every source is updated, and the project's [preset](/reference/configuration/#extend-a-preset) moves to its library's newest library release. `SOURCE`, such as `team`, limits the update to one source. `SOURCE:RULE`, such as `team:techs/react/prefer-server-components`, moves only that rule; nothing else changes, including new rules.

Wherever this section names `SOURCE:RULE`, a skill works the same way, as `SOURCE:SKILL`, such as `fabrica:skills/build-feature`. Moving one skill also installs the skills its new version uses and imports the rules it cites, at their newest versions, and moves nothing that's already installed. A plain repository's skills all come from one commit, so `SOURCE:SKILL` refuses a skill from a plain repository; update the whole source instead.

| Option | Meaning |
| --- | --- |
| `--yes` | Apply the previewed changes without asking. Required to apply changes without a terminal, or with `--json`. |
| `--keep SOURCE:RULE` | Pin this rule or skill at its current version before applying the update, so it stays where it is. Repeat for several. Requires `--reason`. Refused for a skill from a plain repository, which has no versions. |
| `--exclude SOURCE:RULE` | Exclude this new rule or skill before applying the update, so it doesn't join. Repeat for several. Requires `--reason`. |
| `--update-fork SOURCE:RULE` | Replace your local rule or skill for this `replaced` row with a fork of the source's newest version, overwriting your edits, and set its exclusion's `basedOn` to that version, or, for a plain repository, the newest commit. Repeat for several. Takes no `--reason`. See [Replace a fork with the newest version](#replace-a-fork-with-the-newest-version). |
| `--reason TEXT` | The reason recorded with each pin that `--keep` writes and each exclusion that `--exclude` writes. |

The preview lists, for each source:

| Change | Meaning |
| --- | --- |
| `major`, `minor`, or `patch` | The rule's old and new versions, and the summary of every version in between. |
| `new` | A rule the library added to a selected group, at its newest version. |
| `retired` | A rule the library retired, with its last version, its summary, and its replacement when there is one. When the library later retired the replacement too, the preview says so, and names the rule its replacements lead to, if any, instead of recommending the retired one. When your project already replaces the replacement with a local rule, the preview names your local rule instead of recommending the library's. Applying the update drops it. |
| `replaced` | A rule you [replaced with a local rule](/reference/configuration/#exclude-or-replace-a-rule), with your local rule's path. When the exclusion records `basedOn`, the row appears whenever the library's newest version is newer than `basedOn`, and lists every change after it, labeled with its version when there are several, also when a pin keeps the imported copy where it is. Without `basedOn`, it compares with the version the project imports instead: it appears whenever the library's newest version is newer, also when a pin keeps the imported copy where it is, and lists the changes since, which your local rule may already have. Your local rule doesn't change unless you pass `--update-fork`; compare the two to decide whether it needs the same change. |
| `pinned` | A pinned rule that has a newer version. It doesn't move; the preview shows the pin's reason. |
| `removed` | A rule or skill the project stops importing because a preset selection that brought it in was removed, and your configuration doesn't select it. Applying the update drops it. |

Skills from a library appear in the same list, with the same changes, by their skill IDs. A skill that a moving skill newly uses appears as `new`, naming the skill that uses it, and a rule it newly cites appears as `new`, marked reference-only unless the project also selects it; they join only when the skill that brings them in moves. A skill or cited rule that no installed skill needs any longer is dropped.

A plain repository whose `ref` names a branch lists its commit move, such as `main: 3f9c2e1 -> 8a41d07`, and one row for each installed skill folder that differs between the recorded and newest commits:

| Change | Meaning |
| --- | --- |
| `changed` | A skill folder whose files differ, with the paths of the files that changed, including a file whose executable bit changed. |
| `added` | A new skill folder that the source's `skills: "*"` selects. |
| `removed` | An installed skill whose folder no longer exists. Applying the update stops installing it. |

A full update also lists what the preset changed, such as `preset fabrica-defaults: release 6 -> 7`, with each selection, exclusion, pin, or source it added or removed. The rules and skills those changes bring in or drop appear under their sources. See [Update rules](/guides/update/) for examples.

In a terminal, `code-rules project update` shows the preview and asks for confirmation. For each major change and retirement, you can choose to keep the rule at its current version instead; the command then asks for a reason and writes a pin. For each new rule, you can choose to add it or exclude it; excluding asks for a reason and writes an exclusion. For each `replaced` rule, including one whose imported copy a pin keeps, it names the local files that replacing your rule would replace or remove, and you can review the changes later, or replace your rule with a fork of the newest version, as `--update-fork` does. When your answers keep, exclude, or replace rules, it lists them before you confirm, with the files each replaced fork replaces and those it removes. The terminal shows the preview once; the result after you confirm lists the files the update changed. Before each wait on the libraries, such as reading their releases or fetching the update after you confirm, a terminal also shows a short status line, such as `Fetching the update...`, on standard error; JSON output and output that isn't a terminal never include it. Without a terminal, or with `--json`, it shows the preview and writes nothing unless you pass `--yes`. The update applies exactly the versions the preview showed, and writes its pins, exclusions, `basedOn` versions, and replaced forks in the same step that replaces `vendor/` and `generated/`. When the preview has no rule to move, add, or drop, and no shared files to move, a terminal update applies without asking, as sync would, and when that changes no files, says, as a preview would, that no rule updates are available and no files were written.

A preview that writes nothing exits `0`, because available updates aren't an error; so does declining the confirmation. In JSON output, `value.applied` is `false` for a preview and `true` once the update is applied, and `value.sources` lists each source's `rules`, each with its `id`, `change`, versions (`from`, `to`, and, where they apply, `newest` or `lastVersion`), `summaries`, `summaryVersions` (the version each summary belongs to, in the same order), and, where they apply, `replacedBy` (with `replacementRetired: true` and any `currentReplacement` when the library retired the replacement too, and `replacementLocalRule` when your project already replaces that replacement with a local rule), `localRule`, `basedOn`, `pin`, and the `decision` (`keep`, `exclude`, or `update-fork`) that `--keep`, `--exclude`, `--update-fork`, or your answers recorded, with the `reason` of a pin or exclusion. A row's `change` is one of the preview's changes: `major`, `minor`, `patch`, `new`, `retired`, `replaced`, `pinned`, or `removed`. For an `update-fork` decision, a row's `overwrites` lists the local files that the new fork replaces with its own copies, and `removes` those it deletes because the new fork doesn't have them, relative to the Code Rules directory; both are empty for other rows. A `replaced` row's `from` is the version the project imports and `to` the newest version, or `newest` instead when a pin keeps the imported copy; with `basedOn`, its `summaries` are the library's changes after `basedOn`, and otherwise the changes since `from`, not the changes its `localRule` lacks. A source that uses `ref` has its `ref` and no rules. The `added`, `changed`, and `removed` lists are empty for a preview; once applied, they include `config.yaml` when the update wrote pins, exclusions, or `basedOn` versions, and the files of each fork it replaced.

In JSON output, each source in `value.sources` also has a `skills` list, whose rows have the same fields as its `rules`. A plain repository source has `commit`, with the commits `from` and `to`, absent when it doesn't move, and its `skills` rows have a `change` of `added`, `changed`, or `removed`, and `files`, the paths of the changed files relative to the skill's folder, empty for other rows. When the preset moves, `value.preset` has its `name`, the library releases `from` and `to`, and `added` and `removed`, each a list of entries with the `source`, the `field` (`source`, `groups`, `rules`, `skills`, `exclude`, or `pins`), and the `id` or repository address; it's absent when the preset doesn't move.

The preview doesn't list changes to rules or skills you excluded without a replacement. `--keep` and pins in configuration never change which rules are imported. When a rule you exclude or select individually is retired, the entry no longer does anything; update warns about it so you can delete it, and applies the rest of the update. When the update, or a sync, would leave your local rules in a group with no metadata, such as a fork of a retired rule, it writes the group's metadata to `local/<group-id>/_group.yaml` in the same step, lists the file as added, and warns that it did; see [Keep a local rule's group](/reference/sync/#keep-a-local-rules-group).

Sources that use `ref` don't move; change `ref` and run `code-rules project sync` instead. An update of a whole source also moves its [library-wide files](/reference/rule-versions/#what-a-version-covers), such as group metadata, shared assets, and the library's license files, to the newest library release, even when no rule moves; the preview shows it as `Shared files: release 3 -> 4`, and JSON output as the source's `sharedFiles`, with the library releases `from` and `to`, which is absent when they don't move. A `SOURCE:RULE` update leaves them where they are, unless a rule it moves comes from a newer library release: then they move to that library release, since a rule version can rely on the shared files its library release published.

#### Replace a fork with the newest version

For a skill, `--update-fork SOURCE:SKILL` replaces the whole local skill folder that the exclusion's `replacedBy` names with what [`code-rules project add skill --from SOURCE:SKILL`](#fork-a-skill) writes for the source's newest version, or a plain repository's newest commit, and removes files you added to the folder. The rest of this section describes rules; it applies to skills the same way.

`--update-fork SOURCE:RULE` replaces the local rule that a `replaced` row names, the exclusion's `replacedBy`, with a fork of the version the row lists changes up to: the library's newest version, also when a pin keeps the imported copy at an older one. The new fork is exactly what [`code-rules project add rule RULE --from SOURCE@VERSION`](#fork-a-library-rule) writes, with its attribution, its assets, and the shared assets its links reach, placed at the `replacedBy` path, with links rewritten for that path. The update sets the exclusion's `basedOn` to that version, adding it to a replacement you wrote yourself, and installs the fork, `config.yaml`, `vendor/`, and `generated/` in one step, so an interrupted update restores or finishes them together.

Passing `--update-fork` is your consent to overwrite the local rule: your edits to it are lost, so commit them first if you want to compare. The preview, human and JSON, lists every file it replaces and, separately, every file it removes, reading the new fork from the library to tell them apart. The local rule's asset directory is replaced as a whole, including files you added to it, and the shared assets the new version links to are copied into it, as a fork copies them. Nothing outside the rule and its asset directory changes, because each rule owns its asset directory and no other rule may link into it. A pin stays: it governs only the imported copy.

`--update-fork` refuses, as an `invalid-arguments` usage error that exits `2`, a rule the source doesn't exclude with `replacedBy`, an unknown source, a `--reason`, and a rule that `--keep` or `--exclude` also names, before reading any library. It also refuses a rule the preview doesn't list as `replaced`, because the library has nothing newer than the version your rule is based on or the update doesn't include the rule, and a rule the library retired, which has no newest version to fork. Like the rest of the update, it fails with `invalid-release-tag`, writing nothing, when the library release tag that published the fork's version names another commit than when the update previewed it, also when a pin or a `SOURCE:RULE` update keeps the imported copy at an older version. Without `--update-fork`, `--yes` leaves every local rule as it is.

If you merged the library's changes into your local rule by hand, record the version you merged by editing `basedOn` in `.code-rules/config.yaml`, so later updates list only newer changes.

See [Update rules](/guides/update/) for the workflow.

### project build

```sh
code-rules project build [options]
```

Regenerate `generated/` from project configuration, verified imported files, and local rules and skills, and rewrite the installed skill folders in the project's install targets. Build works offline and does not change imported revisions. It also creates a missing managed Code Rules guide or refreshes an older, unedited guide alongside generated output. A manually edited guide stops the build before it changes output.

Accepts the [shared options](#shared-options-and-prompts) only.

Run sync first if the imported repository, selection, pins, or `ref` no longer match configuration, or if the stored library files need repair. Build validates local skills, and fails on an unfinished draft, as sync does.

### project check

```sh
code-rules project check [options]
```

Check generated guidance, installed skills, and the managed Code Rules guide without writing files or contacting repositories. Reports stale, missing, or unexpected output and invalid inputs, including an installed skill folder that differs from what sync would write, a folder that `generated/installed-skills.json` records but that's missing, such as when `.gitignore` excludes the target folders so they were never committed, and a stale `installed-skills.json`, including a source record in `vendor/` that was changed outside `code-rules project sync`, which no command trusts until you restore it or import the source again; see [the source record's checksum](/reference/provenance/#inspect-the-original-imported-files).

Accepts the [shared options](#shared-options-and-prompts) only.

Use check in CI to detect files that need regeneration. It uses recorded commits and rule and skill versions, and does not check whether remote tags moved, a branch a plain repository tracks moved, or newer library releases exist. Success means the managed files agree with their inputs; it does not establish that application code follows the rules.

### project list skills

```sh
code-rules project list skills [options]
```

List the skills the project's sources and preset offer, so you can choose which to install. Lists never write files. The command reads each source's newest library release, or, for a plain repository, the newest commit on the branch its `ref` names, or the tag or commit it names, and needs access to every configured repository.

| Option | Meaning |
| --- | --- |
| `--type TYPE` | Only skills of this type: `playbook` or `review`. |
| `--applies-rules-from GROUP` | Only skills whose `cr-rules-used` names a rule in this group, such as `practices/architecture`. |

```text
$ code-rules project list skills --applies-rules-from practices/architecture
ID                              TYPE      VERSION  STATUS
fabrica:skills/build-feature    playbook  1.3.0    installed
fabrica:skills/write-design-doc playbook  1.1.0    available
skills/review-architecture      review    2.0.1    available, from preset fabrica-defaults

3 skills cite rules in practices/architecture. Skills that cite no rules, including those from plain repositories, show up only without --applies-rules-from.
```

Each skill shows its project ID, its type, when it declares one, and its newest version, or, for a skill from a plain repository, which has no versions, the commit it would come from. `STATUS` is `installed` for a skill the project installs, selected or used by another skill, and `available` for one a source offers that the project doesn't install. `available, from preset <name>` marks a skill in the library that publishes the project's preset, when that library isn't one of the project's sources. Such a skill has no source name yet, so it shows its library ID, such as `skills/review-architecture`; to install it, add that library as a source. Local skills aren't listed. Skills are sorted by project ID. To install one, add it to that source's `skills` and run `code-rules project sync`.

Skills don't belong to groups, so `--applies-rules-from` finds skills by the rules they cite: a skill that cites rules in several groups appears under each of them, and a skill that cites no rules, including every skill from a plain repository, appears only without the option. Both options can be combined.

JSON output lists the skills in `value.skills`, each with its `id`, `status` (`installed` or `available`), and, where they apply, its `type`, `version` or `commit`, and `preset`, the preset's name.

## Library commands

`code-rules library` manages a **library**, an independently maintained collection of rule groups and skills that projects can import. Initialize from the library repository root. Other library commands find that root from subdirectories. Use `--directory PATH` to target another library; see [Working directories](#working-directories).

### library init

```sh
code-rules library init [options]
```

Create `rule-library.yaml`, an authoring README, and a GitHub Actions workflow without overwriting existing authored files. Optionally copy explicitly supplied license terms into the library.

The workflow, `.github/workflows/code-rules.yml`, runs `code-rules library check` on every pull request, using the Code Rules version that created it. A development build, which no release published, writes a workflow that finds the latest release's version when it runs and uses that instead. Either way, the workflow installs only that version's archive, after verifying it against the release's attested `SHA256SUMS`, and prints the version it installed in the job's log and summary. Rerunning `code-rules library init` never changes an existing workflow; to move it to the Code Rules version you run, delete the workflow and run `code-rules library init` again. See [Check changes in CI](/guides/version-rules/#check-changes-in-ci).

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. Init must target the repository root. |
| `--spdx EXPRESSION` | Library SPDX expression. Must be supplied together with `--license-file`. |
| `--license-file PATH` | UTF-8 license text to copy to `LICENSE.md`. Relative to your working directory, even with `--directory`. |
| `--notice-file PATH` | Optional UTF-8 notice text to copy to `NOTICE.md`. Requires both license options. Relative to your working directory. |
| `--non-interactive` | Accepted; library init runs without prompts even when this flag is omitted. |

If you omit the license options, `rule-library.yaml` leaves the license undeclared. Init does not infer license terms, create a Git repository, commit, or publish the library.

### library add group

```sh
code-rules library add group ID [options]
```

Create a group with `_group.yaml` metadata and an authoring README in the library. `ID` is a group path such as `practices/testing` or `techs/typescript`.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--name TEXT` | Required. Group display name. |
| `--description TEXT` | Required. What the group covers. |
| `--when-to-read TEXT` | Required. When an agent should read the group. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Empty groups are valid. Existing group files are preserved.

### library add rule

```sh
code-rules library add rule ID [options]
```

Create a rule in an existing library group. `ID` includes the group path and rule name, such as `techs/javascript/prefer-for-of`, without `.md`.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--title TEXT` | Required. Action-oriented rule title. |
| `--when-to-read TEXT` | Required. When an agent should read the rule. |
| `--impact LEVEL` | Required. One of `CRITICAL`, `HIGH`, `MEDIUM-HIGH`, `MEDIUM`, `LOW-MEDIUM`, or `LOW`. |
| `--impact-description TEXT` | Required. The consequence the rule helps prevent. |
| `--body-file PATH` | Optional UTF-8 Markdown body, without frontmatter. Relative paths start at your working directory. Omit to create an unfinished draft. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Create the group first with `code-rules library add group`; rule creation does not create missing groups. Without `--body-file`, complete the draft and remove its `code-rules:draft` marker before validation. Existing rules are not overwritten. After the first library release, the next steps include adding the new rule's change note with `code-rules library change`. Like `code-rules library check`, it fails with instructions in a shallow clone, or a clone with change notes but no release tags, where it can't tell whether the library has a library release.

### library add skill

```sh
code-rules library add skill NAME [options]
```

Create a skill in the library's `skills/` directory. `NAME` follows the [skill name grammar](/reference/skill-format/#skill-ids), such as `fix-bug`; the skill's ID is `skills/NAME`.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--description TEXT` | Required. What the skill does and when to use it, at most 1,024 characters. |
| `--type TYPE` | Optional. `playbook` or `review`, recorded as `cr-skill-type`. See [Skill types](/concepts/skill/#skill-types). |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

The command writes `skills/NAME/SKILL.md` with the name, description, and type in its frontmatter, and a body with the `code-rules:draft` marker. Complete the draft and remove the marker before validation; see [Write a skill](/guides/write-skills/). An invalid name, or a description that's blank or too long, fails before prompts with exit code `2`. Existing skills are not overwritten, and the ID of a skill a library release retired can't be reused. After the first library release, the next steps include adding the new skill's change note with `code-rules library change`. Like `code-rules library add rule`, it fails with instructions in a shallow clone, or a clone with change notes but no release tags.

### library change

```sh
code-rules library change ID... [options]
```

Write a new [change note](/reference/rule-versions/#change-notes) for one or more rules and skills. `ID` is a rule ID, such as `practices/testing/verify-retry-limits`, or a skill ID, such as `skills/build-feature`; name several to cover related changes in one note. Before the first library release, rules and skills need no notes, so the command fails and says so.

| Option | Meaning |
| --- | --- |
| `--bump LEVEL` | `major`, `minor`, or `patch`. Required for rules and skills that have a version. Not accepted for new or retired ones. |
| `--summary TEXT` | Required. One line describing the change for project maintainers. |
| `--retire` | Record that the rules or skills are retired; see [Retired rules](/reference/rule-versions/#retired-rules) and [Retired skills](/reference/rule-versions/#retired-skills). A rule's Markdown file and asset directory, or a skill's folder, must already be gone. |
| `--replaced-by ID` | Optional with `--retire` and a single `ID`: the rule that replaces a retired rule, or the skill that replaces a retired skill. If it isn't a rule or skill in the library yet, the command warns; add it before the next library release. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

The command writes a new file in `changes/` with a unique name, such as `changes/2026-09-29-verify-retry-limits-7f3a9c.yaml`, and never edits or deletes existing notes. Rule IDs go in the note's `rules` map and skill IDs in its `skills` map. A rule or skill without any version is recorded as `new`. It rejects an ID that isn't a rule or skill in the library, unless `--retire` is supplied for one that has a version. It also rejects a rule that has a version but whose Markdown file and asset directory are unchanged since the latest library release, and a skill whose folder's tracked files and their modes are unchanged: edit it first, then record the change. A rule or skill that was never published can't be retired: delete it, and remove it from any pending note.

`--replaced-by` must be the same kind as the retired ID: a rule for a rule, a skill for a skill. Naming a rule to replace a skill, or the reverse, fails as a usage error with exit code `2`, such as `--replaced-by practices/testing/verify-retries is a rule, but skills/triage-bug is a skill. A skill can only be replaced by a skill.`

See [Choose a version change](/reference/rule-versions/#choose-a-version-change) for picking `LEVEL` for a rule, and [Choose a skill's version change](/reference/rule-versions/#choose-a-skills-version-change) for a skill.

### library check

```sh
code-rules library check [options]
```

Validate `rule-library.yaml`, all groups and rules, all skills, presets, change notes, supporting assets, and declared license and notice files. After the first library release, also check that every changed rule and skill has a change note. Does not change files.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--non-interactive` | Accepted; library check does not prompt. |

Reports group, rule, and skill counts, `value.groupCount`, `value.ruleCount`, and `value.skillCount` in JSON, such as `Library is valid: 3 group(s), 6 rule(s), 3 skill(s).`, and file-specific errors. The human output leaves out the skill count when the library has no skills. Empty groups are valid. Unfinished marked drafts fail. Undeclared licenses produce warnings; invalid declarations, missing declared files, and declared files inside a rule's version (a rule's Markdown file or its asset directory) fail validation. Library check validates the format, not writing quality or legal permissions. Use the [authoring rubric](/reference/rule-authoring/#authoring-rubric) to review guidance quality.

Check validates every skill folder and its `SKILL.md`: the folder's name and contents, the frontmatter fields, the `cr-skill-type`, `cr-rules-used`, and `cr-skills-used` metadata keys, and links. It lists every skill problem it finds and fails with the code `invalid-skill`. It warns when a skill's `license` differs from the library's SPDX expression, and about a file in a skill folder that Git neither tracks nor ignores. [Validation](/reference/skill-format/#validation) lists each check and its message. Check also validates each preset in `presets/`; see [Presets](/reference/library-format/#presets).

After the first library release, check compares each rule's [versioned content](/reference/rule-versions/#what-a-version-covers), meaning its Markdown file and its asset directory, and each skill's, meaning the tracked files in its folder and their modes, with the latest `release/<number>` tag in the current branch's history. Notes added since that tag are pending. Check fails when any of the following is true of a rule, and the same for a skill, with the message naming the skill:

- A rule changed and no pending note names it. The error names the rule and its latest version, and gives the `code-rules library change` command to run.
- A new rule has no pending note.
- A published rule was deleted without a pending note that retires it.
- A retired rule's `replacedBy` isn't a rule in the library, or a retired skill's `replacedBy` isn't a skill in it.
- A rule reuses the ID of a retired rule.
- A pending note names a rule that's unchanged since the latest library release, so the note is stale.
- A pending note names a rule that doesn't exist and isn't being retired.
- A note is invalid, such as an unknown change, a blank summary, a skill ID under `rules`, or neither `rules` nor `skills`.

Check warns when a note that a library release already published was edited, because the edit has no effect. It also warns when such a note was deleted, because notes are never deleted; restore it. When a pending note retires a rule that an earlier library release named as a retired rule's replacement, check warns about it: when the note names a replacement, that updates will point projects still importing the earlier rule on to it; otherwise, that they would be pointed at a retired rule, suggesting a replacement as `replacedBy`. A pending retirement whose `replacedBy` isn't a rule in the library fails.

A skill is compared with the newest library release whose release record has a `skills` field; when a later library release lacks it, because an older Code Rules published it, check warns, as [Skills in the release record](/reference/rule-versions/#skills-in-the-release-record) describes.

This comparison needs the repository's history and tags. Check fails with instructions in a shallow clone, with `shallow-clone`, and in a clone whose commit has change notes but no `release/<number>` tags, such as one made with `git clone --no-tags`, with `missing-release-tags`; fetch the tags with `git fetch --tags`. Check never fetches, so it refuses such a clone; `code-rules library release` fetches the remote's release tags itself. In CI, check out with full history, such as `fetch-depth: 0`. Before the first library release, rules need no notes, and check validates everything else. A library outside a Git repository has no library releases.

When checks pass, the result previews the pending library release: each rule and skill, its change, and its current and next version, in one list sorted by ID. When no rule or skill change is pending and no library-wide file changed, it says there is nothing to publish, as `code-rules library release` does, except before the first library release, which always publishes the library-wide files, even without rules, so check says that instead. When library-wide files, such as group metadata, shared assets, license files, or presets, changed since the latest library release, the preview lists them too, so you know a library release would publish them even when no rule changed. JSON output includes this preview in `value.pendingRelease`, with its `release` number, a `rules` list, and a `libraryFiles` list of the changed library-wide files, which is empty before the first library release. Each rule has its `id` and `change`; `from` and `to`, its version before and after the library release (`from` is absent for a new rule); for a retired rule, `lastVersion` and any `replacedBy` instead; and `summaries`, one per change note that names it, in note order, as the release record and `code-rules project update` name them. `value.pendingRelease.skills` lists pending skill changes in the same form. `code-rules library release` reports the rules and skills it publishes in `value.rules` and `value.skills` the same way.

### library release

```sh
code-rules library release [options]
```

Publish the pending change notes for rules and skills, and any changes to library-wide files, as a [library release](/reference/rule-versions/#library-releases). A library release that changes only skills publishes like any other. See [Version your rules](/guides/version-rules/#publish-a-library-release) for the workflow.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--dry-run` | Show exactly what would be published, including the release notes, without creating a tag or a GitHub Release page. |
| `--no-github-release` | Skip creating the GitHub Release page. |
| `--non-interactive` | Accepted; library release does not prompt. |

Before publishing, the command fetches from the branch's upstream remote. It refuses unless:

- you're on the remote's default branch,
- the branch matches the remote exactly, so the library release contains no unpushed commits and misses none, and
- `code-rules library check` passes.

It then computes each changed rule's and skill's next version from the pending notes, using the largest change when several notes name the same rule or skill. The first library release gives every rule and skill version `1.0.0`. The release record lists skills in its [`skills` field](/reference/rule-versions/#skills-in-the-release-record), and the release notes list skill changes in a section of their own. It creates the annotated `release/<number>` tag on the current commit, with the release notes and [release record](/reference/rule-versions/#release-record) as its message, and pushes it. No files change and nothing is committed. Finally, it creates the GitHub Release page.

`--dry-run` shows the repository, branch, commit, release number, each rule's and skill's change and versions, and the complete release notes.

If a run stops after pushing the tag, such as when the GitHub Release page can't be created, run it again: it finds the tag on the current commit and creates what's missing. A run that stopped before pushing leaves its tag only in your clone; the next run checks it against the library release the commit publishes after the remote's latest one, pushes it only when its release notes and record match exactly, and otherwise refuses with `release-tag-mismatch` and asks you to delete it. A tag that a later Code Rules created in a newer [release record](/reference/rule-versions/#release-record) format instead fails with `unsupported-release-record`: upgrade Code Rules and run it again rather than deleting the tag. The result reports separately whether the tag and the GitHub Release page were created. JSON output gives them as `value.tagCreated` and `value.githubRelease.created`, and describes the library release in `value.release`, `value.rules`, `value.skills`, `value.libraryFiles`, and `value.notes`. Like other optional values that aren't lists, `value.githubRelease` is left out when the library release has no GitHub Release page: on a dry run, with `code-rules library release --no-github-release`, or for a repository that isn't on GitHub.com. When the tag is published but the page can't be created, the command fails with `github-release-failed`, and the JSON response still includes `value`, with `tagCreated` and without `value.githubRelease`, so a script can tell the tag is published and only the page is missing. With no pending notes and no library-wide changes, the command reports that there is nothing to publish, with `value.release` set to `0`, `value.rules`, `value.skills`, and `value.libraryFiles` empty, and `value.tag` and `value.notes` left out, as optional values that don't apply always are, and succeeds. A tool such as `jq` reads a left-out value as `null`.

The tag uses Git's configured identity as its tagger, including the `GIT_COMMITTER_*` environment variables.

When the push fails, the command fails with `push-failed` and deletes the tag it created, so a rerun starts over. Code Rules doesn't show what Git or the server printed, as [Text from Git, servers, and the GitHub CLI](#text-from-git-servers-and-the-github-cli) explains; its message names the cause it recognizes in that text instead:

- a repository rule or tag protection, such as a GitHub ruleset or a GitLab protected tag, with GitHub's error code when the server gave `GH006` or `GH013`,
- the HTTPS server's certificate or the SSH host key couldn't be verified,
- Git couldn't connect to the server,
- the server denied access, or authentication failed,
- a hook on the server, such as a pre-receive or update hook, declined the tag,
- the server refused the tag for a reason Code Rules doesn't recognize, with GitHub's error code when the server gave another one, such as `GH001` for large files, or
- the push failed before the server answered, such as when your pre-push hook stopped it.

To read the server's own message, push a test tag that your tag rules cover yourself, such as `release/0`, which isn't a library release tag:

```sh
git tag release/0
git push origin refs/tags/release/0
git tag --delete release/0
```

If that push succeeds, delete the test tag from the remote with `git push origin --delete refs/tags/release/0`.

`code-rules library release` refuses before creating anything, with exit code `1` and `error.code` in JSON output, when:

| Code | Refusal |
| --- | --- |
| `not-a-repository` | The library isn't the root of a Git repository. |
| `shallow-clone` | The clone has partial history. Fetch it with `git fetch --unshallow --tags`, or check out with `fetch-depth: 0` in CI. |
| `missing-release-tags` | The commit has change notes, but neither the clone nor the remote has `release/<number>` tags. The command fetches the remote's release tags before checking, so it doesn't refuse a clone made with `git clone --no-tags`; that fetch adds them to the clone, also on a dry run. |
| `detached-head` | `HEAD` isn't on a branch. |
| `no-upstream` | The branch has no upstream branch on a remote. |
| `push-destination` | The remote pushes to several URLs, or to another repository than it fetches from. |
| `fetch-failed` | Git couldn't read or fetch from the remote. The message says whether the HTTPS server's certificate or the SSH host key couldn't be verified, Git couldn't connect to the server, or the server denied access or has no such repository, when it recognizes one of them; run `git fetch` with the remote's name to read Git's message. |
| `not-default-branch` | The branch doesn't track the remote's default branch, or the remote reports none. |
| `no-commits` | The library has no commits yet. |
| `branch-differs` | The branch has commits the remote lacks, the remote has commits the branch lacks, or the remote branch doesn't exist. |
| `remote-changed` | Someone pushed to the branch, or changed a release tag, while the command ran. |
| `release-tag-mismatch` | A release tag in the clone differs from the remote's, exists only in the clone without being the next one on this commit, or doesn't publish what this commit would. |
| `invalid-release-tag` | A release tag reachable from the commit isn't a valid annotated tag with a release record. |
| `unsupported-release-record` | A release tag uses a newer release record format. Upgrade Code Rules. |
| `uncommitted-changes` | Library files differ from the checked-out commit. |
| `change-notes` | Change notes don't match the rule or skill changes, as `code-rules library check` reports; other check failures keep their own codes, such as `invalid-skill`. |
| `version-limit` | A change would take a rule's or skill's version past the largest version number. |
| `git-identity` | Git doesn't know your name and email for the tag's tagger. |
| `github-cli-missing` | A GitHub Release page applies, but `gh` isn't installed. |
| `github-cli-signed-out` | A GitHub Release page applies, but `gh` isn't signed in to GitHub.com. |
| `release-too-large` | The release tag would exceed 8 MiB. |
| `limit-exceeded` | The clone has more than 20,000 release tags, or `gh` printed more than its output limit. |
| `changed-input` | Library files changed while the command read them. |

After it creates the tag, it can fail with:

| Code | Failure |
| --- | --- |
| `tag-failed` | Git couldn't create the tag, such as when tag signing fails. |
| `push-failed` | The push failed, including when the remote refused it, with the cause described above. The command deletes the tag it created. |
| `release-conflict` | Someone else published the same `release/<number>` first. |
| `github-cli-failed` | `gh` couldn't run. |
| `github-release-failed` | `gh` couldn't look up or create the GitHub Release page. The message names the cause it recognizes: GitHub's rate limit, `gh` signed out or its credentials refused, permission denied, or a repository GitHub can't find. Otherwise, when the lookup failed, it says to run `gh release view` to read `gh`'s message; when creating the page failed, it suggests checks that change nothing, `gh auth status` and `gh repo view`, and running the command again, which creates the page with its generated notes only if it's still missing. The tag is published; run the command again to create the page. |

**GitHub Release pages** are created with the [GitHub CLI](https://cli.github.com/), `gh`, for repositories on GitHub.com. Each library release gets one GitHub Release page on its `release/<number>` tag, with the release notes as its body. Before changing anything, `code-rules library release` checks that `gh` is installed and signed in, and refuses if it isn't, unless you pass `--no-github-release`. `code-rules library release --dry-run` checks it too, and refuses the same way. Repositories hosted elsewhere get tags only. The page's URL in the result is built from the repository and the tag, never taken from what `gh` printed.

<span id="help-and-version"></span>

## Help, version, and license

```sh
code-rules --help
code-rules COMMAND --help
code-rules --version
code-rules --license
```

Use `-h` or `--help` to inspect command syntax and options. For example, run `code-rules project --help` to find project commands or `code-rules library add rule --help` for rule authoring options.

| Option | Meaning |
| --- | --- |
| `-h`, `--help` | Show help for the selected command. |
| `-v`, `--version` | At the root, print the executable's version. |
| `--license` | At the root, print the full embedded MIT license and copyright notice. |
| `--json` | Return help, version, or license text in `value.text` inside a JSON response. |

Running `code-rules` or a command group such as `code-rules library` without a subcommand displays help. The `code-rules project`, `code-rules project add`, `code-rules project list`, `code-rules library`, and `code-rules library add` groups organize commands; they do not perform operations themselves. Help never prompts or writes files.

The root `--version` flag prints the tool version. To choose which versions of a library's rules and skills to import, see [Choose versions](/reference/configuration/#choose-versions).

## Working directories

In Git repositories, run `code-rules project init` or `code-rules library init` from the repository root. Initialization from a subdirectory fails without writing files. This also applies to the target of `code-rules library init --directory`.

Other commands find the nearest ancestor containing a `.git` directory or file, including worktrees and submodules. Project commands use that root's `.code-rules/config.yaml`; library commands use its `rule-library.yaml`. They stop at that repository boundary, even if its configuration is missing. A nested `.code-rules/` does not override the root's configuration.

Outside Git, commands use the current directory, or the directory selected by `--directory` for library commands. They do not search parent directories for configuration. Custom project configuration locations are not supported.

Relative input paths such as `--body-file`, `--license-file`, and `--notice-file` resolve from the directory where you run the command. Paths inside project configuration remain relative to `.code-rules/`.

## Shared options and prompts

Every command accepts `--json`, which prints one JSON response and disables prompts. Every command also accepts `-h` / `--help`.

Authoring commands accept `--non-interactive`. Without it, commands can prompt for missing required metadata or source selections when running in a terminal. With `--non-interactive`, `--json`, or no terminal, missing required inputs cause an error. Positional arguments such as `ID` and `ALIAS` must always be supplied.

Invalid interactive answers repeat the same question while retaining earlier answers; for a choice, such as `[yes/no]`, the error lists the valid answers. Without a terminal, or with `--json` or `--non-interactive`, a missing required input fails with exit code `2` and names the flag to pass, such as `--summary`. Explicit flags are validated without prompting for replacement values. Existing groups, rules, skills, and source names fail before prompts.

Both init commands, both check commands, `code-rules project list skills`, and `code-rules library release` run without prompts. Of these, `code-rules library check` and `code-rules library release` accept `--non-interactive`; `code-rules project check`, `code-rules project sync`, `code-rules project update`, `code-rules project build`, and `code-rules project list skills` do not need or accept it.

String options accept one value and cannot be repeated, except `--groups`, `--rules`, `--skills`, `--skill-targets`, `--keep`, `--exclude`, and `--update-fork`, which accept repeated values. For a value beginning with `-`, use the equals form, such as `--description='-prefixed text'`.

Scaffolding commands validate paths and detect collisions before writing. They preserve existing authored files and roll back failed creation attempts. They do not publish content or convert arbitrary third-party material. For the authoring workflow, see [Set up your first project](/start-here/set-up-project/) or [Create your first library](/start-here/create-library/).

## Command output

Human-readable output is the default. Add `--json` for scripts:

```sh
code-rules project check --json
```

JSON mode writes one response to stdout:

| Field | Meaning |
| --- | --- |
| `ok` | `true` for success; `false` for failure or an out-of-date project check. |
| `value` | The command's result, when available. An out-of-date check still includes its report here. |
| `error` | On failure, an object with `kind` (`usage`, `validation`, `operation`, `cancelled`, or `out-of-date` for a stale project check) and `message`, plus `location` when available. Domain failures include a stable `code`, such as `needs-init`, `missing-group`, or `guide-edited`, and every `usage` error has the code `invalid-arguments`. |

Every list in a value is always present, empty when there's nothing to report, including `warnings`. Enumerated values, such as statuses, kinds, changes, and codes, are kebab-case. Only optional values that aren't lists, such as a `ref` or a `githubRelease` object, are left out when they don't apply.

Project check returns `value.status` as `up-to-date` or `out-of-date`, and a `value.problems` list. Each problem has a `kind` (`missing-file`, `stale-contents`, `unexpected-file`, or `outdated-readme`), `path`, `message`, and `nextSteps`, a list of repair steps in the same form as authoring results use. Paths are relative to the Code Rules directory, except paths in a skill install target, which are relative to the project root. Generated guidance, installed skills, and the managed Code Rules guide must all be current for success.

Authoring results include `value.nextSteps`, an ordered list of steps, each an `instruction` and its copyable `commands`. Human output shows those steps after initialization and rule or group creation.

Sync, update, and build report `added`, `changed`, and `removed` file lists, with paths relative to the Code Rules directory, or to `generated/` for build, except paths in a skill install target, such as `.claude/skills/build-feature/SKILL.md`, which are relative to the project root. Authoring commands, such as init, add group, add rule, add skill, add library, add repository, and library change, report the absolute paths of the files they created in `added` and of those they modified, such as `config.yaml`, in `changed`. Their human output lists the same files under `Added:` and `Changed:`, relative to the working directory when they're inside it. Update also reports each source's rule changes in `value.sources`. Sync and update list their warnings, such as for an unreleased `ref`, an entry naming a retired rule, or local group metadata they wrote, in `value.warnings`, which is empty when there are none. When they refresh the managed Code Rules guide, `value.guide` reports its path relative to the Code Rules directory and whether it was `created`. Help, version, and license return their text in `value.text`.

In human mode, operational errors go to stderr. An out-of-date check prints its status, problems, and next steps on stdout. Human output, prompts, and errors show control characters, which could make a terminal clear or rewrite what it displays, as visible escapes: ESC as `\x1b`, other controls from `\x00` to `\x1f` and `\x7f` the same way, C1 controls as `\u0080` to `\u009f`, and bytes that aren't valid UTF-8 as `\xNN`. Libraries supply much of that text, such as change summaries and rule IDs. JSON output encodes text as JSON strings instead. In JSON mode, errors go in the response; stderr is reserved for failures writing that response. Unreleased preview builds also print a non-production warning with their source commit to stderr before every command, including help, version, and JSON commands. JSON output on stdout is unchanged. See [Test a PR build](https://github.com/fabricahq/code-rules/blob/main/CONTRIBUTING.md#test-a-pr-build) in the contributing guide.

## Text from Git, servers, and the GitHub CLI

Code Rules never shows text that Git, a Git server, or the GitHub CLI, `gh`, produced: not in human output, JSON output, or errors. That includes Git's error messages, the lines a server or its hooks print, the reason in Git's `! [remote rejected]` line, the default branch a server advertises, and everything `gh` prints. Such text can hold anything, including credentials Code Rules can't know about, such as one a Git credential helper, `GIT_ASKPASS`, or `http.extraHeader` supplies, so Code Rules doesn't try to find and remove them. It reads the text privately to choose one of its own fixed messages and error codes, which name the likely cause and what to check. A message may name a GitHub error code it recognizes, such as `GH013`, because those are a fixed set. To read the original text, run the Git or `gh` command yourself; the message or this reference says which.

Code Rules also never shows credentials in a URL it displays. A repository address it shows, such as a library's remote in `code-rules library release`, has any password, query, and fragment removed, and its user name too unless it's an SSH user name, which names an account. An address it can't parse is replaced by a note, including one with an authority but no scheme, such as `//host/rules`, and a `user@host:path` address with a colon before its `@`, which could start a password. The GitHub Release page URL is built from the repository and the tag, not taken from `gh`.

Code Rules shows some values that Git computes, after checking their form: commit and object IDs, which must be hexadecimal object IDs; commit counts, which must be nonnegative integers; and object types, which must be one of Git's four. A value in any other form fails with code `git-failed` and a fixed message instead.

Code Rules shows as they are:

- names you chose locally, such as branch and remote names,
- the repository addresses in project configuration, which can't contain credentials, and
- library content, such as rule files, change notes, group descriptions, and release records.

Human output still shows control characters in that text as visible escapes, as [Command output](#command-output) describes.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | An operation failed, inputs are invalid, or a project check found stale output. |
| `2` | Invalid command usage, such as an unknown command, unsupported option, missing required CLI input, or options the command can't combine, such as `code-rules library change --bump` for a new rule, or a `code-rules project update` decision the preview doesn't offer, such as `--keep` for a rule the update doesn't move. |
| `130` | The command was interrupted, such as with Ctrl-C, including at a prompt. It says it was cancelled; a command interrupted before writing files writes none, and one interrupted while replacing files is recovered by the next command, as [Sync and recovery](/reference/sync/) describes. JSON output reports `error.kind` as `cancelled`. |

Usage errors point to the relevant help page. Operational errors identify the affected file or rule when available and describe the problem. A failure of `code-rules project sync` or `code-rules project update` that wrote nothing, including an update's refusal of its arguments after the command line is parsed, ends with `No files were written.`, or, when the command first recovered an interrupted earlier command, which restores or finishes that command's files, says so instead, as an update's preview and a declined or interrupted update do too, and names the source, as in `source team:`, when the problem doesn't already. An interrupted one says `cancelled; no files were written`, or, after such a recovery, that it was cancelled and what the recovery did. For repair commands and interrupted updates, see [Sync and recovery](/reference/sync/).

## Migrating from earlier commands

Only the `project` and `library` command trees are supported. Earlier command paths now fail with an unknown-command error.
Update existing scripts to use the scoped commands below. Use `-h` or `--help` for help; there is no `help` subcommand.

| Earlier command | Canonical command |
| --- | --- |
| `code-rules init` | `code-rules project init` |
| `code-rules add source` | `code-rules project add library` |
| `code-rules local add group` | `code-rules project add group` |
| `code-rules local add rule` | `code-rules project add rule` |
| `code-rules sync` | `code-rules project sync` |
| `code-rules build` | `code-rules project build` |
| `code-rules check` | `code-rules project check` |

Library command paths are unchanged. For configuration fields that changed, see [Configuration](/reference/configuration/#fields).
