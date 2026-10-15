---
title: "Install skills"
description: "Install skills from a library, another repository, or your own project into the folders your agents read."
---

:::note[Preview]
Skills come in the next Code Rules release. Code Rules checks the form of `fg-skills-used` but doesn't yet install the skills it names.
:::

Code Rules installs [skills](/concepts/skill/) from your sources and from your own project into the folders your agents read. This guide shows how to choose where they go, add skills from a library or another repository, write your own, and change what's installed.

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
  fg-skill-type: playbook
  fg-rules-used: |
    practices/releases/write-release-notes
    fabrica:practices/testing/test-behavior-not-implementation
---

1. Check that the main branch is green.
2. ...
```

`name` must equal the folder's name. The `metadata` keys are optional:

- `fg-skill-type` labels the skill `playbook` or `review`.
- `fg-rules-used` lists the rules the skill applies, one per line, in a block string that starts with `|`. A bare ID, such as `practices/releases/write-release-notes`, names a rule you author in `.code-rules/rules/`; a source's rule takes its source name, such as `fabrica:practices/testing/test-behavior-not-implementation`.
- `fg-skills-used` lists the skills it calls, in the same form, such as `skills/tag-release` or `fabrica:skills/tdd`.

See [What a skill contains](/concepts/skill/#what-a-skill-contains). Then install it:

```sh
code-rules project build
```

Build works offline and installs authored skills alongside your sources' skills. It fails, naming the skill's `SKILL.md`, when the frontmatter breaks one of these requirements or cites a rule the project doesn't have. Each installed copy ends with a list of links to the project's version of every rule the skill cites; those rules don't join `RULES.md`. Edit the skill in `.code-rules/skills/`, never in an installed copy, and run build again after each change: `code-rules project check` fails while an installed copy differs from what build would write.

## Stop installing a skill

Remove the skill's ID from `skills.include`, or delete an authored skill's folder, and run `code-rules project sync` or `code-rules project build`. Code Rules removes the installed copies, and a skill target's folder once nothing is left in it.

## When install fails

| Failure | What to do |
| --- | --- |
| `skill-folder-not-managed` | A folder in a skill target already has the skill's install name, and Code Rules didn't install it, such as a skill you put there by hand. Move or delete that folder, perhaps into `.code-rules/skills/` to make it an authored skill, or stop including the skill. |
| `skill-name-clash` | Two skills you include have the same install name. Stop including one of them. |
| `skill-not-in-revision` | The revision the source record locks doesn't have a skill you named. Run the update the message names, set `ref` to a revision that has it, or check the ID. |
| `ref-required` | The source's repository has no library releases. Set its `ref`. |
| An invalid authored skill | Its `name` doesn't match its folder, a `metadata` key is invalid, or `fg-rules-used` names a rule the project doesn't have. Correct the `SKILL.md` the message names. |

## When check fails for a skill

`code-rules project check` reports an installed copy that's missing, edited, or left behind by a skill the project no longer installs, as `missing-skill`, `modified-skill`, or `unexpected-skill`, naming the skill and its folder. Run `code-rules project build` to reinstall every skill from its source; it replaces edits to the installed copies, so move any you want to keep into a skill you author first.

When a skill you author cites a rule its library retired, update, sync, and build warn, and check fails, naming the skill and the rule. Remove the rule from `fg-rules-used`, or cite the rule that replaces it.

For every file sync, update, and build change, see [Sync and recovery](/reference/sync/#skills).
