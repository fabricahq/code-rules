---
title: "Skill"
description: "A reusable procedure that tells an agent what to do, step by step, and which rules each step applies."
---

A **skill** is an [Agent Skills](https://agentskills.io) folder: a `SKILL.md` file that tells an agent how to carry out one kind of task, plus any scripts, references, or assets it bundles.

Rules say what good work looks like. Skills say what to do and in what order. A skill that builds a feature might tell the agent to read the spec, decide where the code lives, write a failing test, implement, and verify the result in the browser. The rules it applies say how each of those steps should be done well.

Code Rules installs skills into your project, keeps them versioned, and checks them. It doesn't run skills, decide when an agent uses one, or check that the agent followed it.

## What a skill contains

A skill lives in a folder named after it:

```text
build-feature/
  SKILL.md
  references/
    slicing.md
  scripts/
    verify.sh
```

`SKILL.md` starts with metadata, followed by the instructions:

```markdown
---
name: build-feature
description: Build a feature from an agreed spec, one slice at a time. Use when starting implementation of a new feature or a change to an existing one.
metadata:
  cr-skill-type: playbook
  cr-rules-used: |
    practices/testing/test-behavior-not-implementation
    practices/verification/verify-on-the-surface-users-touch
  cr-skills-used: |
    skills/tdd
---

1. Read the spec and name the slice you're building.
2. Decide where the code lives before writing it.
...
```

- `name` and `description` come from the Agent Skills format. Agents read the description to decide when to use the skill.
- `cr-skill-type` labels what kind of skill it is. See [Skill types](#skill-types).
- `cr-rules-used` lists the rules the skill applies.
- `cr-skills-used` lists other skills it uses.

All three `cr-` keys are optional. For every field and check, see [Skill format](/reference/skill-format/). To write one, follow [Write a skill](/guides/write-skills/).

## Skill types

A skill can declare one type:

| Type | What the skill does | Examples |
| --- | --- | --- |
| `playbook` | Ordered steps for carrying out one kind of engineering task, citing the rules each step applies. | Build a feature, fix a bug, migrate a dependency |
| `review` | Examines existing work, such as a diff, plan, test suite, or rendered UI, against stated criteria and reports findings. | Review a pull request, audit a test suite |

The test is the skill's main job: a playbook changes the work, and a review judges work that already exists. A review may offer to fix what it found, and a playbook may include a review step.

A skill that fits neither type leaves the field out. The type is a label for people and catalogs, such as `code-rules project list skills --type review`. Code Rules installs, checks, and updates typed and untyped skills the same way.

## Skills and rules

Keep opinions in rules and order in skills. When a skill needs a practice followed, it cites the rule rather than restating it:

- A project can exclude or replace the rule, and every skill that cites it picks up the project's choice.
- When the rule gets a new version, every skill that cites it applies the new version, without a change to the skill.
- Reviewers check work against the same rules the implementing agent followed.

Skills don't belong to [groups](/concepts/groups/), and selecting a group never installs a skill. When you install a skill, Code Rules also imports the rules it cites, even from groups your project didn't select. Those rules are for reference: the skill links to them, but they don't join `RULES.md`, so installing a skill never widens the rules that apply to all of your project's work.

## Where skills come from

A project gets skills from three places:

- **Libraries.** A [library](/concepts/libraries/) can hold skills in `skills/` beside its rule groups. Library skills are versioned and checked strictly.
- **Plain repositories.** Any Git repository with skill folders, such as a third party's skill collection, without a `rule-library.yaml`. Its `ref` either tracks a branch or stays on one revision, and Code Rules records the exact commit.
- **Your project.** Local skills live in `.code-rules/local/skills/`.

When you select a skill, Code Rules also installs every skill it uses, from the same source.

A skill's ID is its path in the library, `skills/<name>`. In a project, it's qualified by the source name you chose, such as `fabrica:skills/build-feature`. Configuration uses these IDs to select, exclude, and replace skills, the same way it does for rules.

See [Install skills](/guides/install-skills/) to add skills to a project.

## How a skill is versioned

A library skill has its own semantic version, which covers its whole folder and follows the same levels as a [rule's](/concepts/rule/#how-a-rule-is-versioned):

- **Major:** the skill now produces or requires something different, such as a removed or reordered step that people rely on, a new required input, or a new rule or skill it uses.
- **Minor:** the skill adds steps or guidance without breaking how it was used.
- **Patch:** the skill only clarifies its wording.

Library authors record each change in a change note, and a library release publishes the new versions. Unlike rules, a project doesn't choose skill versions one by one: all of a source's skills come from the same library release, the one that supplies the source's library-wide files. A skill's version tells you what changed. Projects see it in their generated files and in the `code-rules project update` preview, and every installed skill moves when its source updates. To hold a skill back, exclude it, [fork it](/guides/install-skills/#fork-a-skill), or set the source's `ref`, which holds everything from that source. Local skills and skills from plain repositories have no versions.

See [Rule and skill versions](/reference/rule-versions/) for the details.

## How agents use installed skills

When you sync, Code Rules writes a copy of each selected skill into the folders your agents read: `.claude/skills/` for Claude Code and `.agents/skills/` for Codex, Cursor, and other agents that follow the shared convention. You commit these copies, so every checkout and every fresh cloud session has the same skills.

When a skill cites rules, its installed `SKILL.md` ends with a generated section, **Rules this skill applies**, that points at your project's version of every rule the skill cites, including any local rule that replaces one. Agents read those rules in full before the step that uses them.

Code Rules also writes `.code-rules/generated/SKILLS.md`, which lists every installed skill, its version, how many rules it applies, and the skills it uses. Agents in tools without native skill support can read it there.

When a skill and a rule disagree, agents should follow the rule and report the conflict, so someone can fix the skill. See [For agents](/for-agents/).
