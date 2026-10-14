---
title: "Install skills"
description: "Install skills from libraries and plain repositories into the folders your agents read, and manage their versions, dependencies, and exceptions."
---

A **[skill](/concepts/skill/)** tells an agent how to carry out one kind of task, step by step, and which rules each step applies. Installing skills with Code Rules gives every checkout of your project the same skills, at the versions you adopt, beside the rules they cite.

In this guide, you'll see which skills your sources offer, install some from a library, and find out where they land. You'll also exclude or fork a skill, write a local one, install skills from a plain Git repository, and commit the result.

Start with a project that already has a `.code-rules/` directory. If you haven't created one, follow [Set up your first project](/start-here/set-up-project/). To choose rules as well, see [Import rules](/guides/select-rules/).

## Check your install targets

Code Rules installs a copy of each skill into the folders your agents read. `skillTargets` in `.code-rules/config.yaml` names them:

```yaml
schemaVersion: 2
skillTargets: [claude, agents]
```

`claude` installs into `.claude/skills/`, which Claude Code reads. `agents` installs into `.agents/skills/`, which Codex, Cursor, and other agents that follow the shared convention read. `code-rules project init` writes both, and nothing is written to either folder until the project installs a skill.

A project initialized with an earlier version of Code Rules has no `skillTargets`, which means no targets. Add the field, and set `schemaVersion: 2`, which any skill field requires. With no targets, Code Rules still imports your skills, checks them, and lists them in `.code-rules/generated/SKILLS.md`, but writes no installed copies. See [Install targets](/reference/configuration/#install-targets).

## See what's available

List the skills your sources offer:

```sh
code-rules project list skills
```

```text
ID                             TYPE      VERSION  STATUS
fabrica:skills/build-feature   playbook  1.3.0    installed
fabrica:skills/review-change   review    3.0.0    available
fabrica:skills/tdd             playbook  1.1.0    installed
pstack:skills/architect                  8a41d07  installed
pstack:skills/sketch                     8a41d07  available
skills/plan-migration          playbook  2.1.0    available, from preset fabrica-defaults
```

Each row shows a skill's ID, its [type](/concepts/skill/#skill-types) when it has one, and its newest version, or, for a plain repository, the newest commit on the branch its `ref` names. `installed` means the project installs it, as a selection or a dependency; `available` means a source offers it and the project doesn't install it. When your project [extends a preset](/guides/presets/), skills in the preset's own library are listed too, marked `available, from preset <name>`; to install one, add that library as a source.

Narrow the list:

- `--type review` lists only review skills, and `--type playbook` only playbooks.
- `--applies-rules-from practices/testing` lists only skills whose `fg-rules-used` names a rule in that group. Skills that cite no rules, including every skill from a plain repository, appear only without this option.
- `--json` returns the list for scripts.

To see a library you haven't added yet, browse its `skills/` folder. See [project list skills](/reference/cli/#project-list-skills) for every option.

## Install skills from a library

Add a library with the skills you want:

```sh
code-rules project add library fabrica \
  --repository https://github.com/fabricahq/public-rules.git \
  --skills skills/build-feature \
  --non-interactive
```

A source can select skills without any groups or rules. For a library your project already imports, add the skill IDs to its `skills` list in `.code-rules/config.yaml` instead:

```yaml
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    groups:
      - practices/testing
    skills:
      - skills/build-feature
```

To install every skill the library offers, now and when it adds more, set `skills: "*"`. Skills don't belong to groups: selecting `practices/testing` installs no skill, and selecting a skill selects no group.

Then sync:

```sh
code-rules project sync
```

Sync installs every skill from the source's newest library release and records that release and each skill's version in `.code-rules/vendor/fabrica/_source.json`. All of a source's skills come from that one release. Later syncs keep them there; adopt newer versions with [`code-rules project update`](/guides/update/#update-skills). See [Select skills](/reference/configuration/#select-skills) for the exact rules.

## What lands where

After sync, each installed skill appears in several places:

| Path | What it is | Who edits it |
| --- | --- | --- |
| `.claude/skills/build-feature/` and `.agents/skills/build-feature/` | The installed copies your agents load, one folder per target. | Code Rules |
| `.code-rules/vendor/fabrica/skills/build-feature/` | The original files, as the library published that version. | Code Rules |
| `.code-rules/generated/SKILLS.md` | One entry per installed skill: its description, ID and version, type, installed paths, how many rules it applies, the skills it uses, and whether you selected it or another skill uses it. | Code Rules |
| `.code-rules/generated/installed-skills.json` | Which folders in the targets Code Rules installed. | Code Rules |

An installed copy matches the library's folder, scripts and their executable bits included, with two additions to `SKILL.md`:

- A marker comment as the first line after the frontmatter, naming the skill's project ID and version, such as `<!-- code-rules:installed fabrica:skills/build-feature 1.3.0. Do not edit; fork it with code-rules project add skill --from. -->`.
- When the skill cites rules, a generated section at the end, **Rules this skill applies**, which links to your project's version of each rule the skill cites, and a **Skills this skill uses** line when it uses others.

Don't edit installed copies: the next sync replaces them, and `code-rules project check` reports the difference. To change a skill, [fork it](#fork-a-skill).

Code Rules writes only the skill folders it installs. Other folders in `.claude/skills/` and `.agents/skills/`, such as a skill another tool installed, stay as they are. When one of them has the same name as a skill you select, sync fails with `skill-folder-not-managed` instead of overwriting it. See [Installed skills](/reference/files/#installed-skills) and [Who owns the target folders](/reference/sync/#who-owns-the-target-folders).

`generated/RULES.md` gains one line pointing to `SKILLS.md`, so agents whose tools don't load skills from the target folders still find them through the instructions in your `AGENTS.md`.

## Dependencies and cited rules

A skill can use other skills and cite rules. When you install it, Code Rules installs those too:

- **Skills it uses.** Every skill in its `fg-skills-used` list, and the skills those use, from the same library. `SKILLS.md` says each was installed because another skill uses it.
- **Rules it cites.** Every rule in its `fg-rules-used` list, from the same library, even from groups you don't select. These cited rules are reference-only: each gets its generated rule file, and the installed skill links to it, but `RULES.md` doesn't list it. Installing a skill never widens the rules your agents apply to all work.

If you exclude or replace a cited rule, the installed skill follows your choice: it links to your replacement, or says the rule is excluded by this project, with your reason. A cited rule your project also selects through a group is an ordinary imported rule.

Two skills with the same name can't both be installed, because agents find skills by name. Sync fails with `skill-name-conflict`, naming both; [exclude](#exclude-a-skill) one of them.

## Exclude a skill

Exclude a skill you don't want, such as one that `"*"`, your preset, or another skill brings in:

```yaml
sources:
  fabrica:
    exclude:
      skills/write-changelog:
        reason: Release Planner writes our release notes.
```

These are partial snippets, not complete source definitions. An excluded skill isn't installed, but it's still imported into `vendor/`, so updates can show you its changes. When another installed skill uses it, sync warns that the skill it uses isn't installed.

You can't pin a skill to a version the way you pin a rule: all of a source's skills come from one library release and move together when the source updates. To keep a skill as it is, [fork it](#fork-a-skill), or set the source's `ref` to a release tag or commit, which holds every rule and skill from that source. See [Update skills](/guides/update/#update-skills).

## Fork a skill

To change what an imported skill says, fork it into your project:

```sh
code-rules project add skill \
  --from fabrica:skills/fix-bug \
  --reason 'We use our own bug workflow.' \
  --non-interactive
```

The command copies the version your project installs into `.code-rules/local/skills/fix-bug/`, and adds an exclusion that names the fork:

```yaml
exclude:
  skills/fix-bug:
    reason: We use our own bug workflow.
    replacedBy: local/skills/fix-bug
    basedOn: "1.0.0"
```

Edit the fork, then run `code-rules project build`. Agents get your fork in place of the library's skill, installed under the same name. With `basedOn`, `code-rules project update` lists the library's later changes to the skill, so you can decide whether your fork needs them, or replace it with a fork of the newest version with `--update-fork fabrica:skills/fix-bug`. The library's license still applies to the copied text. See [project add skill](/reference/cli/#project-add-skill).

## Write a local skill

Create a skill for this project alone:

```sh
code-rules project add skill review-migration \
  --description 'Review a database migration for safety before it merges. Use when a change adds or edits a migration.' \
  --type review \
  --non-interactive
```

The command writes a draft `.code-rules/local/skills/review-migration/SKILL.md`. Local skills have no versions; they change when you edit them. In `fg-rules-used`, a local skill cites rules by project ID, such as `fabrica:practices/testing/test-behavior-not-implementation` or `local:practices/testing/test-project-contracts`, because it can cite rules from any source. Run `code-rules project build` to install it. Follow [Write a skill](/guides/write-skills/) for how to write one that works.

## Install skills from a plain repository

Not every useful skill lives in a library. A **plain repository** is any Git repository that holds skill folders without a `rule-library.yaml`, such as a third party's collection. Add one with `code-rules project add repository`:

```sh
code-rules project add repository pstack \
  --repository https://github.com/cursor/plugins.git \
  --ref main \
  --skills-root pstack/skills \
  --skills skills/architect \
  --non-interactive
```

This adds a source to your configuration:

```yaml
sources:
  pstack:
    repository: https://github.com/cursor/plugins.git
    ref: main
    skillsRoot: pstack/skills
    skills:
      - skills/architect
```

Then run `code-rules project sync`.

- **Track a branch or stay on one revision.** `ref` is required. When it names a branch, sync installs the newest commit on the branch and records it; later syncs keep that commit, and `code-rules project update` moves to the branch's newest commit and shows which skill folders and files changed. When it names a tag or full commit SHA, the source never moves.
- **Point at the skills.** `skillsRoot` is the folder that holds the skill folders, `skills` by default. Each skill's ID is `skills/<folder-name>`, whatever the root.
- **Declare dependencies.** A plain repository's skills declare none that Code Rules reads. When one uses another, list it under `skillUses`, so Code Rules installs both:

  ```yaml
  skillUses:
    skills/architect:
      - skills/how
  ```

- **Expect warnings, not failures.** Code Rules checks these skills leniently: where one doesn't follow the Agent Skills format, sync warns and installs it as it is. A plain repository has no rules, versions, or pins, so `groups`, `rules`, and `pins` are rejected on it.
- **Check the license.** Code Rules looks for terms in the skill's `license` field, then a `LICENSE*` file in its folder, then `LICENSE*` and `NOTICE*` files at the repository root, and records what it finds in provenance. When a skill has none, sync fails with `license-not-found`; ask the owner to add a license, or stop selecting the skill.

Code Rules never runs a skill's scripts. Before you install or update a skill from a repository you don't control, read its files, scripts included: your agents may run them. To help, the `code-rules project update` preview flags each new or changed script and warns you to read it; see [Update skills](/guides/update/#update-skills). See [Plain repositories](/reference/configuration/#plain-repositories).

## Commit the installed skills

Commit `.code-rules/` and the installed skill folders together:

```sh
git add .code-rules .claude/skills .agents/skills
```

Committed copies give every checkout, CI job, and fresh cloud session the same skills without running sync. Run `code-rules project check` in CI: it fails when an installed copy differs from what sync would write, when a folder Code Rules installed is missing, or when `generated/installed-skills.json` is out of date.

If your `.gitignore` excludes a target folder, such as with `/.claude/`, check reports the folders as missing in every fresh checkout. Add an exception for the skills folder. Git can't re-include a folder inside an excluded folder, so first change the ignore line to exclude the folder's contents instead:

```text
/.claude/*
!/.claude/skills/
```

This keeps ignoring the rest of `.claude/`, such as local settings, while committing installed skills.
