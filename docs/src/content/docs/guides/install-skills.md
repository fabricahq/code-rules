---
title: "Install skills"
description: "Install skills from libraries and other repositories into the folders your agents read, and manage their levels, names, and exceptions."
---

:::caution[Preview]
Skill support isn't in a released version of Code Rules yet. This guide describes how it will work.
:::

A **[skill](/concepts/skill/)** tells an agent how to carry out one kind of task, step by step, and which rules each step applies. Installing skills with Code Rules gives every checkout of your project the same skills, from the revisions you choose, beside the rules they cite.

In this guide, you'll see which skills your sources offer, install a library's required and recommended skills, and find out where they land. You'll also install skills from a repository that isn't a library, resolve a name clash, exclude or fork a skill, write your own, and commit the result.

Start with a project that already has a `.code-rules/` directory. If you haven't created one, follow [Set up your first project](/start-here/set-up-project/). To choose rules as well, see [Include rules](/guides/select-rules/).

## Check your skill targets

Code Rules installs a copy of each skill into the folders your agents read. `skillTargets` in `.code-rules/config.yaml` names them:

```yaml
schemaVersion: 2
skillTargets: [claude, agents]
```

`claude` installs into `.claude/skills/`, which Claude Code reads. `agents` installs into `.agents/skills/`, which Codex, Cursor, and other agents that follow the shared convention read. `code-rules project init` writes both, and nothing is written to either folder until the project installs a skill.

To record which skills your sources require before you choose agent folders, set `skillTargets: []`. Code Rules still records and checks every skill and lists it in `.code-rules/generated/SKILLS.md`, but writes no installed copies, and sync warns that it installed nothing. See [Skill targets](/reference/configuration/#skill-targets).

## Add a library

Add your organization's library as a source:

```sh
code-rules project add source fabrica \
  --repository https://github.com/fabricahq/.code-rules.git \
  --skills recommended \
  --non-interactive
```

This records the source in `.code-rules/config.yaml`:

```yaml
sources:
  fabrica:
    repository: https://github.com/fabricahq/.code-rules.git
    skills:
      include: [recommended]
```

Then sync:

```sh
code-rules project sync
```

What a library's skills install depends on the **level** the library sets for each one:

- **Required** skills always install. Even a source with no `include` at all installs them, now and after every update that adds one.
- **Recommended** skills install because you included `recommended`.
- **Offered** skills install only when you name them, or include `"*"` for everything the library offers.

To add an offered skill, name its ID beside `recommended`:

```yaml
skills:
  include: [recommended, skills/deploy-preview]
```

Including a rule group's rules never installs a skill, and including a skill never includes a rule group's rules. See [Skill selectors](/reference/configuration/#skill-selectors).

Sync takes the source's newest library release, records it in `.code-rules/vendor/fabrica/_source.json`, and installs every skill from that one release. Later syncs keep that release, even for skills you include later: a skill that isn't in the recorded release makes sync fail and name `code-rules project update fabrica`, the command that moves the source. See [Update skills](/guides/update/#update-skills).

## See what's available

List every skill your sources offer, with its level:

```sh
code-rules project list skills
```

```text
NAME            ID                                               LEVEL        SET BY             VERSION  STATUS
build-feature   fabrica:skills/build-feature                     recommended  fabrica release 7  1.3.0    installed
deploy-preview  fabrica:skills/deploy-preview                    offered      fabrica release 7  1.0.0    installed
review-change   fabrica:skills/review-change                     offered      fabrica release 7  3.0.0    available
grilling        fabrica:mattpocock:skills/productivity/grilling  offered      fabrica release 7  8a41d07  available
ship-service    fabrica:skills/ship-service                      required     fabrica release 7  1.0.0    installed
tdd             fabrica:skills/tdd                               offered      fabrica release 7  1.1.0    installed
to-spec         fabrica:mattpocock:skills/engineering/to-spec    recommended  fabrica release 7  8a41d07  installed
```

Each row shows a skill's install name, its ID, its level and the library release that set it, its version, or a commit for a skill without versions, and whether the project installs it. `installed` covers skills you included, required skills, and skills another installed skill calls; `available` means a source offers it and the project doesn't install it.

The `mattpocock:` skills are **borrowed**: Fabrica's library lists them at a level, but their files come from the original repository, at a commit your project resolves when it updates. See [Borrowed skills](/reference/configuration/#borrowed-skills).

Narrow the list:

- `--type review` lists only review skills, and `--type playbook` only playbooks.
- `--applies-rules-from practices/testing` lists only skills whose `syllabus-rules-used` names a rule in that rule group.
- `--json` returns the list for scripts.

See [project list skills](/reference/cli/#project-list-skills) for every option.

## What lands where

After sync, each installed skill appears in several places:

| Path | What it is | Who edits it |
| --- | --- | --- |
| `.claude/skills/build-feature/` and `.agents/skills/build-feature/` | The installed copies your agents load, one folder per target. | Code Rules |
| `.code-rules/vendor/fabrica/skills/build-feature/` | The original files, as the library published that version. | Code Rules |
| `.code-rules/generated/SKILLS.md` | One entry per installed skill: its description, ID and version, level and the library that set it, type, installed paths, how many rules it applies, the skills it calls, and why it's installed. | Code Rules |
| `.code-rules/generated/installed-skills.json` | Which folders in the targets Code Rules installed. | Code Rules |

An installed copy matches the library's folder, scripts and their executable bits included, with two additions to `SKILL.md`:

- A marker comment as the first line after the frontmatter, naming the skill's project ID and version, such as `<!-- code-rules:installed fabrica:skills/build-feature 1.3.0. Do not edit; fork it with code-rules project add skill --from. -->`.
- When the skill cites rules, a generated section at the end, **Rules this skill applies**, which links to your project's version of each rule the skill cites, and a **Skills this skill uses** line when it calls others.

Don't edit installed copies: the next sync replaces them, and `code-rules project check` reports the difference. To change a skill, [fork it](#fork-a-skill).

Code Rules writes only the skill folders it installs. Other folders in `.claude/skills/` and `.agents/skills/`, such as a skill another tool installed, stay as they are. When one of them has the name of a skill you install, sync fails with `skill-folder-not-managed` instead of overwriting it. See [Installed skills](/reference/files/#installed-skills) and [Who owns the target folders](/reference/sync/#who-owns-the-target-folders).

`generated/RULES.md` gains one line pointing to `SKILLS.md`, so agents whose tools don't load skills from the target folders still find them through the instructions in your `AGENTS.md`.

## Skills a skill calls, and the rules it cites

A skill can call other skills and cite rules. When you install it, Code Rules installs those too:

- **Skills it calls.** Every skill in its `syllabus-skills-used` list, and the skills those call, from the same repository and revision. `SKILLS.md` says each was installed because another skill uses it.
- **Rules it cites.** Every rule in its `syllabus-rules-used` list, from the same library, even ones your rule `include` doesn't name. These cited rules are reference-only: each gets its generated rule file, and the installed skill links to it, but `RULES.md` doesn't list it. Installing a skill never widens the rules your agents apply to all work.

If you exclude or replace a cited rule, the installed skill follows your choice: it links to your replacement, or says the rule is excluded by this project, with your reason. A cited rule your project also includes is an ordinary rule.

When a skill calls another without saying so in `syllabus-skills-used`, as many third-party skills do, declare the call under `skills.uses`, so Code Rules installs both:

```yaml
skills:
  uses:
    skills/architect: [skills/how]
```

Code Rules doesn't search skill text for calls it wasn't told about. See [Declare the skills a skill calls](/reference/configuration/#declare-the-skills-a-skill-calls).

## Install skills from a repository that isn't a library

Not every useful skill lives in a library. Any Git repository that holds skill folders works as a source, such as a third party's collection. Add it the same way:

```sh
code-rules project add source pstack \
  --repository https://github.com/example/pstack.git \
  --ref main \
  --skills skills/architect \
  --non-interactive
```

```yaml
sources:
  pstack:
    repository: https://github.com/example/pstack.git
    ref: main
    skills:
      include: [skills/architect]
```

Then run `code-rules project sync`. You don't declare what kind of source it is: Code Rules finds that the repository has no library manifest, so it's a **non-library source**.

- **Every skill is offered.** Nothing installs until you name it, or include `"*"` for every skill in the repository.
- **A ref is required.** A non-library source has no library releases, so `ref` must name a branch, tag, or commit. With a branch, sync installs the newest commit on it the first time and records it; later syncs keep that commit, and `code-rules project update` moves to the branch's newest commit and shows which skill folders and files changed.
- **IDs are paths.** Each folder with a `SKILL.md` is a skill, and its ID is the folder's path in the repository, such as `skills/architect`.
- **Expect warnings, not failures.** Code Rules checks these skills leniently: where one doesn't follow the Agent Skills format, sync warns and installs it as it is. A non-library source has no rules or versions, so `rules` is rejected on it.
- **Check the license.** Code Rules looks for terms in the skill's `license` field, then a `LICENSE*` file in its folder, then `LICENSE*` and `NOTICE*` files at the repository root, and records what it finds in provenance. When a skill has none, sync fails with `license-not-found`; ask the owner to add a license, or stop including the skill.

Code Rules never runs a skill's scripts. Before you install or update a skill from a repository you don't control, read its files, scripts included: your agents may run them. To help, the `code-rules project update` preview flags each new or changed executable script and warns you to read it; see [Update skills](/guides/update/#update-skills).

## Try a newer version of a borrowed skill

A library chooses the `ref` it tracks for each repository it borrows skills from, and your project resolves that ref when it updates. To take a borrowed skill from another revision, declare the same repository as your own source, with your own `ref`:

```yaml
sources:
  fabrica:
    repository: https://github.com/fabricahq/.code-rules.git
    skills:
      include: [recommended]
  mattpocock:
    repository: https://github.com/mattpocock/skills.git
    ref: v2.0.0
```

The skills Fabrica borrows from that repository now come from `v2.0.0`, still at the level Fabrica sets. Repository identity decides the match, not the source name. Remove the source to go back to the library's ref.

## Resolve a name clash

Agents find skills by name, so two installed skills can't share one. When two sources ship skills with the same name, sync fails with `skill-name-clash` and writes nothing:

```text
Error: skill-name-clash: fabrica:skills/review-change and pstack:skills/review-change both install as review-change. Rename one with skills.rename, or exclude one. No files were written.
```

Rename one of them:

```yaml
sources:
  fabrica:
    repository: https://github.com/fabricahq/.code-rules.git
    skills:
      include: [recommended, skills/review-change]
      rename:
        skills/review-change: fabrica-review
```

The installed copy is now `fabrica-review`, in `.claude/skills/fabrica-review/`. Its ID stays `fabrica:skills/review-change`. Code Rules doesn't rewrite the old name where other skills mention it, so every sync that applies the rename warns:

```text
Warning: fabrica:skills/review-change is installed as fabrica-review. Mentions of review-change in skill text aren't rewritten; check the skills that call it by that name.
```

Check whether any installed skill tells the agent to use `review-change`. You can rename a required skill too, because it stays installed.

## Exclude a skill

Exclude a skill you don't want, such as one that `recommended` or `"*"` brings in, with a reason:

```yaml
sources:
  fabrica:
    skills:
      include: [recommended]
      exclude:
        - id: skills/write-changelog
          reason: Release Planner writes our release notes.
```

These are partial snippets, not complete source definitions. An excluded skill isn't installed, but it stays in `vendor/`, so updates can show you its changes. When another installed skill calls it, sync warns that the skill it calls isn't installed.

You can't exclude or fork a [required skill](/reference/configuration/#skill-levels): sync fails and names the library that requires it. Required skills are your organization's floor; to diverge from one, stop using the source.

You can't pin a skill to a version either: all of a source's skills come from one revision and move together when the source updates. To keep a skill as it is, [fork it](#fork-a-skill), or set the source's `ref` to a library release tag or commit, which holds every rule and skill from that source. See [Update skills](/guides/update/#update-skills).

## Fork a skill

To change what an included skill says, fork it into your project:

```sh
code-rules project add skill \
  --from fabrica:skills/fix-bug \
  --reason 'We use our own bug workflow.' \
  --non-interactive
```

The command copies the version your project installs into `.code-rules/skills/fix-bug/`, and adds an exclusion that names the fork:

```yaml
skills:
  exclude:
    - id: skills/fix-bug
      reason: We use our own bug workflow.
      replacedBy: skills/fix-bug
      basedOn: "1.0.0"
```

Edit the fork, then run `code-rules project build`. Agents get your fork in place of the library's skill. With `basedOn`, `code-rules project update` lists the library's later changes to the skill, so you can decide whether your fork needs them, or replace it with a fork of the newest version with `--update-fork fabrica:skills/fix-bug`. The library's license still applies to the copied text. See [project add skill](/reference/cli/#project-add-skill).

## Write your own skill

Create a skill for this project alone:

```sh
code-rules project add skill review-migration \
  --description 'Review a database migration for safety before it merges. Use when a change adds or edits a migration.' \
  --type review \
  --non-interactive
```

The command writes a draft `.code-rules/skills/review-migration/SKILL.md`. Authored skills have no versions; they change when you edit them. In `syllabus-rules-used`, an authored skill cites rules by project ID, such as `fabrica:practices/testing/test-behavior-not-implementation` or `practices/testing/test-project-contracts` for one of your own, because it can cite rules from any source. Run `code-rules project build` to install it. Follow [Write a skill](/guides/write-skills/) for how to write one that works.

## Personal skills with the same name

People also install skills for themselves, outside any repository, such as in `~/.claude/skills/`. When one of those **personal skills** has the same name as a skill your project installs, each agent decides which to use, and some use the personal one without saying so. `code-rules project sync`, `update`, and `check` read the known personal skill folders and warn about each pair, naming both paths and what each agent does:

```text
Warning: your personal skill ~/.claude/skills/build-feature has the same name as this project's build-feature (fabrica:skills/build-feature, installed at .claude/skills/build-feature).
  Claude Code uses your personal skill.
  Codex shows both.
  VS Code uses this project's skill.
Rename or remove your personal skill to use the project's.
```

Code Rules never writes to your home directory, and the warning doesn't fail the command. See [Personal skills with the same name](/reference/sync/#personal-skills-with-the-same-name).

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
