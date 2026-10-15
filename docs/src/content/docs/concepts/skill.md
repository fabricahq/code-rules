---
title: "Skill"
description: "A reusable procedure that tells an agent what to do, step by step, which Code Rules installs where agents find it."
---

:::note[Preview]
Skills come in the next Code Rules release. Skill levels, versions, borrowed skills, renames, and exclusions aren't supported yet, and a skill you author can use a source's skill only when the source already installs it. A skill links only to the cited rules the project already includes: Code Rules doesn't yet take a cited rule the project's rule `include` leaves out.
:::

A **skill** is an [Agent Skills](https://agentskills.io) folder: a `SKILL.md` file that tells an agent how to carry out one kind of task, plus any scripts, references, or assets it bundles.

Rules say what good work looks like. Skills say what to do and in what order. A skill that builds a feature might tell the agent to read the spec, write a failing test, implement, and verify the result. The rules say how each of those steps should be done well.

Code Rules installs skills into your repository and records where each one came from. It doesn't run skills, decide when an agent uses one, or check that the agent followed it.

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
  syllabus-rules-used: |
    practices/testing/test-behavior-not-implementation
    practices/verification/verify-on-the-surface-users-touch
  syllabus-skills-used: |
    skills/tdd
---

1. Read the spec and name the slice you're building.
2. Write a failing test for the slice.
...
```

- `name` and `description` come from the Agent Skills format. Agents find a skill by its `name`, and read the description to decide when to use it. The `name` is also the skill's **install name**, the name of its installed folders. It uses 1 to 64 lowercase letters, digits, and hyphens, without a leading, trailing, or doubled hyphen. In a skill your project or library authors, `name` is required and must equal the folder's name; a skill from another repository that has none installs under its folder's name.
- `metadata.syllabus-skill-type` optionally labels the skill: `playbook`, for ordered steps that carry out a task, or `review`, for examining existing work against stated criteria. Code Rules shows the label in `SKILLS.md` and treats every skill the same way.
- `metadata.syllabus-rules-used` lists the rules the skill applies, one rule ID per line.
- `metadata.syllabus-skills-used` lists the other skills it calls, one skill ID per line. Installing a source's skill also installs every skill it uses, and the skills those use, from the same source; see [Install the skills a skill uses](/guides/install-skills/#install-the-skills-a-skill-uses).

All three `syllabus-` keys are optional, and each value is a string, as the Agent Skills format requires, so write the two lists as YAML block strings that start with `|`. Other `syllabus-` keys are reserved for Code Rules and rejected. Other frontmatter fields and `metadata` keys are kept as they are.

## Rules a skill cites

Keep opinions in rules and order in skills. When a skill needs a practice followed, it cites the rule in `syllabus-rules-used` rather than restating it, so a project's exclusions and replacements, and a rule's new versions, reach every skill that cites it without a change to the skill.

A library's skill cites rules in its library by their library rule ID, such as `practices/testing/test-behavior-not-implementation`, without a source name, because each project chooses its own. A skill you author in a project cites your authored rules by their bare ID and a source's rules by their project ID, such as `fabrica:practices/testing/test-behavior-not-implementation`. `syllabus-skills-used` names skills the same way: `skills/tdd` in the same repository, or, in a project, `fabrica:skills/tdd` from a source.

Cited rules are for reference: the installed skill links to them, but they don't join `RULES.md`, so installing a skill never widens the rules that apply to all of your project's work. See [How agents use installed skills](#how-agents-use-installed-skills).

## Where skills come from

A project gets skills from three places:

- **Libraries.** A [library](/concepts/libraries/) publishes the skills in its `.code-rules/skills/` folder, one folder per skill. Every skill a library publishes is offered: a project installs the ones it names in `skills.include`, or all of them with `"*"`.
- **Other repositories.** Any Git repository with skill folders and no library manifest, such as a third party's skill collection. Every folder in it that holds a `SKILL.md` is a skill. Because it has no library releases, its source needs a `ref`, a tag or full commit SHA.
- **Your project.** Skills you author live in `.code-rules/skills/<name>/`.

A skill's ID is its folder's path in the repository that holds it, such as `skills/build-feature` for a library's skill or `tools/review` for a skill in another repository. In a project, a source's skill is qualified by the source name, such as `fabrica:skills/build-feature`; an authored skill keeps its bare ID, such as `skills/release-checklist`.

All of a source's skills come from one revision, the library release or commit the project recorded for that source, so skills that call each other always match. Sync never moves that revision; `code-rules project update` moves a library source to its newest library release.

A project can [exclude](/guides/install-skills/#exclude-a-skill) a source's skill it doesn't want, with a reason, or [fork](/guides/install-skills/#fork-a-skill) it into an authored skill that installs in its place under the same name. Updates list the source's later changes to a skill the project forked, so the project can decide whether its fork needs them.

See [Install skills](/guides/install-skills/) to add skills to a project.

## How agents use installed skills

Build and sync write a copy of each skill into the folders your agents read, the [skill targets](/reference/configuration/#skill-targets): `.claude/skills/` for Claude Code and `.agents/skills/` for agents that follow the shared convention. You commit these copies, so every checkout and every fresh cloud session has the same skills. Each copy's `SKILL.md` carries a line saying not to edit it, and naming the skill and the commit it came from.

When a skill cites rules, its installed `SKILL.md` ends with a generated section, **Rules this skill applies**, that links to your project's version of every rule the skill cites, including an authored rule that replaces one, and says when the project excluded a cited rule, doesn't include it, or its library retired it. Agents read those rules in full before the step that applies them.

Code Rules also writes `.code-rules/generated/SKILLS.md`, which lists every skill the project installs: what it does, where it came from, where it's installed, how many rules it applies, which skills it uses, and why it's installed. `RULES.md` points to it, so agents in tools without native skill support can find skills there.

`code-rules project check` fails when an installed copy differs from what build would write, such as after an edit to it, and when a skill you author cites a rule its library retired.

Agents find skills by name, so two skills with the same install name can't both be installed. Code Rules writes only into the skill targets in your repository, never into your home directory, and never replaces a folder there that it didn't install.

A person can also install **personal skills** outside any repository, which their agents load in every repository. When a personal skill has the same name as a skill your project installs, each agent decides which one to use: Claude Code uses the personal skill, Codex shows both, and VS Code uses the project's. Code Rules warns when it finds such a pair; see [Personal skills with the same name](/reference/sync/#personal-skills-with-the-same-name).

When a skill and a rule disagree, agents should follow the rule and report the conflict, so someone can fix the skill.
