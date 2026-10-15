---
title: "Install skills"
description: "Install skills from a library, another repository, or your own project into the folders your agents read."
---

:::note[Preview]
Skills come in the next Code Rules release. Sync installs the skills a source's skill uses; a skill you author in `.code-rules/skills/` can use a source's skill only when the source already installs it.
:::

Code Rules installs [skills](/concepts/skill/) from your sources and from your own project into the folders your agents read. This guide shows how to choose where they go, add skills from a library or another repository, write your own, exclude or fork a source's skill, and change what's installed.

## Choose where skills are installed

`code-rules project init` writes `skillTargets: [claude, agents]` to a new configuration, so every skill is installed into both `.claude/skills/` and `.agents/skills/`. To choose other targets, pass `--skill-targets` once for each, or `none`:

```sh
code-rules project init --skill-targets claude
```

Running init again on an existing project changes `skillTargets` only when you pass `--skill-targets`. You can also edit `skillTargets` in `.code-rules/config.yaml`; see [Skill targets](/reference/configuration/#skill-targets). With no skill targets, the project still records its skills and lists them in `.code-rules/generated/SKILLS.md`, and sync warns that it installed no copy.

## Install skills from a library

Name the skills by ID, the folder's path in the library, such as `skills/build-feature`, or include every skill the library offers with `*`:

```sh
code-rules project add source fabrica \
  --repository https://github.com/fabricahq/rules.git \
  --rules 'practices/testing/*' \
  --skills skills/build-feature
code-rules project sync
```

The repository and skill ID are illustrative; replace them with a source you can access. Sync installs each skill from the same library release as the source's rules, so skills that call each other always match, and writes `.claude/skills/build-feature/` and `.agents/skills/build-feature/`. Commit the installed copies with the rest of `.code-rules/`.

To install another skill later, add its ID to the source's `skills.include` and run `code-rules project sync`. Sync never moves the source to a newer library release for it: when the recorded library release doesn't have the skill, sync fails with `skill-not-in-revision` and names the update that takes it:

```sh
code-rules project update fabrica
```

Code Rules never runs a skill's scripts, but your agents may. When an update moves a library's skills, its preview flags each executable script it adds or changes, including every script of a skill it newly installs, and warns you to read them before you confirm; see [Review skill scripts](/guides/update/#review-skill-scripts).

## Install skills from another repository

A repository that isn't a library, such as a third party's skill collection, works as a source too. Every folder in it that holds a `SKILL.md` is a skill, named by its path from the repository root. Because it has no library releases, set `--ref` to a tag or full commit SHA:

```sh
code-rules project add source review-tools \
  --repository https://github.com/example/review-skills.git \
  --ref v2.1.0 \
  --skills tools/review
code-rules project sync
```

Without `--ref`, sync fails with `ref-required`. `*` installs every skill in the repository except those in folders whose names start with a dot, such as another tool's installed copies. To take a newer revision, change `ref` and sync. Review a third party's skills, including any scripts they bundle, before installing them: agents follow their instructions.

## Install the skills a skill uses

A skill can call other skills. When its `SKILL.md` lists them in `syllabus-skills-used`, sync installs them with it, and the skills they use in turn, from the same source and revision, so you include only the skill you want:

```yaml
sources:
  pstack:
    repository: https://github.com/example/pstack.git
    ref: v1.4.0
    skills:
      include: [skills/poteto-mode]
```

`generated/SKILLS.md` lists each skill a skill uses, and says when one was installed only because another skill uses it, such as `**Installed because:** used by poteto-mode`.

When a skill calls another without listing it in `syllabus-skills-used`, as many third-party skills do, declare the call in the source's `skills.uses`, so sync installs both:

```yaml
    skills:
      include: [skills/poteto-mode]
      uses:
        skills/architect: [skills/how]
```

Code Rules doesn't read skill text to find calls it wasn't told about. See [Declare the skills a skill uses](/reference/configuration/#declare-the-skills-a-skill-uses).

## Write your own skill

Create a folder in `.code-rules/skills/` named after the skill, with its `SKILL.md`:

```text
.code-rules/skills/release-checklist/SKILL.md
```

```markdown
---
name: release-checklist
description: Prepare a release. Use when asked to cut a release.
metadata:
  syllabus-skill-type: playbook
  syllabus-rules-used: |
    practices/releases/write-release-notes
    fabrica:practices/testing/test-behavior-not-implementation
---

1. Check that the main branch is green.
2. ...
```

`name` must equal the folder's name. The `metadata` keys are optional:

- `syllabus-skill-type` labels the skill `playbook` or `review`.
- `syllabus-rules-used` lists the rules the skill applies, one per line, in a block string that starts with `|`. A bare ID, such as `practices/releases/write-release-notes`, names a rule you author in `.code-rules/rules/`; a source's rule takes its source name, such as `fabrica:practices/testing/test-behavior-not-implementation`.
- `syllabus-skills-used` lists the skills it calls, in the same form, such as `skills/tag-release` or `fabrica:skills/tdd`.

See [What a skill contains](/concepts/skill/#what-a-skill-contains). Then install it:

```sh
code-rules project build
```

Build works offline and installs authored skills alongside your sources' skills. It fails, naming the skill's `SKILL.md`, when the frontmatter breaks one of these requirements or cites a rule the project doesn't have. Each installed copy ends with a list of links to the project's version of every rule the skill cites; those rules don't join `RULES.md`. Edit the skill in `.code-rules/skills/`, never in an installed copy, and run build again after each change: `code-rules project check` fails while an installed copy differs from what build would write.

## Exclude a skill

To leave out a skill that `"*"` brings in, exclude it with a reason:

```yaml
sources:
  fabrica:
    skills:
      include: ["*"]
      exclude:
        - id: skills/write-changelog
          reason: Release Planner writes our release notes.
```

This is a partial snippet, not a complete source definition. Run `code-rules project sync`: the skill isn't installed, and `.code-rules/generated/SKILLS.md` lists it with your reason. It stays in `vendor/`, so updates can compare it with a replacement. See [Exclude or replace a skill](/reference/configuration/#exclude-or-replace-a-skill).

You can't pin a skill to a version: all of a source's skills come from one revision and move together when the source updates. To keep a skill as it is, fork it.

## Fork a skill

To change what a source's skill says, fork it into your project:

```sh
code-rules project add skill \
  --from fabrica:skills/fix-bug \
  --reason 'We use our own bug workflow.' \
  --non-interactive
```

The command copies the skill as your project installs it, with its scripts' executable bits, into `.code-rules/skills/fix-bug/`, and adds an exclusion that names the fork and the commit it copied:

```yaml
skills:
  exclude:
    - id: skills/fix-bug
      reason: We use our own bug workflow.
      replacedBy: skills/fix-bug
      basedOn: "3f9c2a7b1d4e6f8091a2b3c4d5e6f708192a3b4c"
```

:::note[Preview]
Skills have no versions yet, so `basedOn` is the commit the fork copied, for a library's skill too.
:::

Edit the fork, then run `code-rules project build`. Agents get your fork, under the same name, in place of the source's skill, and `SKILLS.md` says which skill it replaces and why. When the source later changes the skill, `code-rules project update` lists it as `replaced`, with the files that changed since the commit your fork is based on, so you can decide whether your fork needs them; the update never changes your fork. After you merge changes by hand, set `basedOn` to the commit you merged. The source's license still applies to the copied files. See [project add skill](/reference/cli/#project-add-skill).

To replace a skill with one you wrote yourself, name your skill's ID, such as `skills/our-bug-workflow`, as the exclusion's `replacedBy`, without `basedOn`.

## Stop installing a skill

Remove the skill's ID from `skills.include`, or delete an authored skill's folder, and run `code-rules project sync` or `code-rules project build`. Code Rules removes the installed copies, and a skill target's folder once nothing is left in it.

## Personal skills with the same name

:::note[Preview]
The personal skill warning comes in the next Code Rules release, with skills.
:::

People also install skills for themselves, outside any repository, such as in `~/.claude/skills/`. When one of those **personal skills** has the same name as a skill your project installs, each agent decides which one to use, and some use the personal one without saying so. `code-rules project sync`, `update`, and `check` look in your personal skill folders and warn about each pair, naming both paths and what each agent does:

```text
Warning: your personal skill ~/.claude/skills/build-feature has the same name as this project's build-feature (fabrica:skills/build-feature, installed at .claude/skills/build-feature).
  Claude Code uses your personal skill.
  Codex shows both.
  VS Code uses this project's skill.
Rename or remove your personal skill to use the project's.
```

Code Rules never writes to your home directory, and the warning doesn't fail the command. See [Personal skills with the same name](/reference/sync/#personal-skills-with-the-same-name).

## When install fails

| Failure | What to do |
| --- | --- |
| `skill-folder-not-managed` | A folder in a skill target already has the skill's install name, and Code Rules didn't install it, such as a skill you put there by hand. Move or delete that folder, perhaps into `.code-rules/skills/` to make it an authored skill, or stop including the skill. |
| `skill-name-clash` | Two skills you include have the same install name. Stop including or exclude one of them. |
| `skill-not-in-revision` | The revision the source record locks doesn't have a skill you named. Run the update the message names, set `ref` to a revision that has it, or check the ID. |
| `ref-required` | The source's repository has no library releases. Set its `ref`. |
| A skill uses a skill that can't be found | A skill's `syllabus-skills-used`, or your `skills.uses`, names a skill the source's revision doesn't have. The message names both skills. Correct `skills.uses`, ask the source's maintainer to correct `syllabus-skills-used`, or stop including the skill. When `skills.uses` names a skill a newer revision has, sync fails with `skill-not-in-revision` instead. |
| Skills use each other in a cycle | The message lists the cycle, such as `skills/a -> skills/b -> skills/a`. Remove one of the uses. |
| An invalid authored skill | Its `name` doesn't match its folder, a `metadata` key is invalid, or `syllabus-rules-used` names a rule the project doesn't have. Correct the `SKILL.md` the message names. |

## When check fails for a skill

`code-rules project check` reports an installed copy that's missing, edited, or left behind by a skill the project no longer installs, as `missing-skill`, `modified-skill`, or `unexpected-skill`, naming the skill and its folder. Run `code-rules project build` to reinstall every skill from its source; it replaces edits to the installed copies, so move any you want to keep into a skill you author first.

When a skill you author cites a rule its library retired, update, sync, and build warn, and check fails, naming the skill and the rule. Remove the rule from `syllabus-rules-used`, or cite the rule that replaces it.

For every file sync, update, and build change, see [Sync and recovery](/reference/sync/#skills).
