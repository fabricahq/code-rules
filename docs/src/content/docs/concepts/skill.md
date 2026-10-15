---
title: "Skill"
description: "A reusable procedure that tells an agent what to do, step by step, which Syllabus installs where agents find it."
---

:::note[Preview]
Skills come in the next Syllabus release. Skill levels, versions, borrowed skills, renames, exclusions, and the rules a skill cites aren't supported yet.
:::

A **skill** is an [Agent Skills](https://agentskills.io) folder: a `SKILL.md` file that tells an agent how to carry out one kind of task, plus any scripts, references, or assets it bundles.

Rules say what good work looks like. Skills say what to do and in what order. A skill that builds a feature might tell the agent to read the spec, write a failing test, implement, and verify the result. The rules say how each of those steps should be done well.

Syllabus installs skills into your repository and records where each one came from. It doesn't run skills, decide when an agent uses one, or check that the agent followed it.

## What a skill contains

A skill lives in a folder named after it:

```text
build-feature/
  SKILL.md
  scripts/
    verify.sh
```

`SKILL.md` starts with frontmatter, followed by the instructions:

```markdown
---
name: build-feature
description: Build a feature from an agreed spec, one slice at a time. Use when starting a new feature.
metadata:
  syllabus-skill-type: playbook
---

1. Read the spec and name the slice you're building.
2. Write a failing test for the slice.
...
```

- `name` and `description` come from the Agent Skills format. Agents find a skill by its `name`, and read the description to decide when to use it. The `name` is also the skill's **install name**, the name of its installed folders; without one, the folder's name is used. It uses 1 to 64 lowercase letters, digits, and hyphens, without a leading, trailing, or doubled hyphen.
- `metadata.syllabus-skill-type` optionally labels the skill: `playbook`, for ordered steps that carry out a task, or `review`, for examining existing work against stated criteria. Syllabus shows the label in `SKILLS.md` and treats every skill the same way.

Other frontmatter fields are kept as they are.

## Where skills come from

A project gets skills from three places:

- **Libraries.** A [library](/concepts/libraries/) publishes the skills in its `.syllabus/skills/` folder, one folder per skill. Every skill a library publishes is offered: a project installs the ones it names in `skills.include`, or all of them with `"*"`.
- **Other repositories.** Any Git repository with skill folders and no library manifest, such as a third party's skill collection. Every folder in it that holds a `SKILL.md` is a skill. Because it has no library releases, its source needs a `ref`, a tag or full commit SHA.
- **Your project.** Skills you author live in `.syllabus/skills/<name>/`.

A skill's ID is its folder's path in the repository that holds it, such as `skills/build-feature` for a library's skill or `tools/review` for a skill in another repository. In a project, a source's skill is qualified by the source name, such as `fabrica:skills/build-feature`; an authored skill keeps its bare ID, such as `skills/release-checklist`.

All of a source's skills come from one revision, the library release or commit the project recorded for that source, so skills that call each other always match. Sync never moves that revision; `syllabus project update` moves a library source to its newest library release.

A project can [exclude](/guides/install-skills/#exclude-a-skill) a source's skill it doesn't want, with a reason, or [fork](/guides/install-skills/#fork-a-skill) it into an authored skill that installs in its place under the same name. Updates list the source's later changes to a skill the project forked, so the project can decide whether its fork needs them.

See [Install skills](/guides/install-skills/) to add skills to a project.

## How agents use installed skills

Build and sync write a copy of each skill into the folders your agents read, the [skill targets](/reference/configuration/#skill-targets): `.claude/skills/` for Claude Code and `.agents/skills/` for agents that follow the shared convention. You commit these copies, so every checkout and every fresh cloud session has the same skills. Each copy's `SKILL.md` carries a line saying not to edit it, and naming the skill and the commit it came from.

Syllabus also writes `.syllabus/generated/SKILLS.md`, which lists every skill the project installs: what it does, where it came from, and where it's installed. `RULES.md` points to it, so agents in tools without native skill support can find skills there.

Agents find skills by name, so two skills with the same install name can't both be installed; a project [renames](/guides/install-skills/#rename-a-skill) one of them to keep both. A rename changes only the installed name: text that mentions the skill's old name still uses it. Syllabus writes only into the skill targets in your repository, never into your home directory, and never replaces a folder there that it didn't install.

When a skill and a rule disagree, agents should follow the rule and report the conflict, so someone can fix the skill.
