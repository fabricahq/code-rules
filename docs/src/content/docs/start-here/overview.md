---
title: "What is Code Rules?"
description: "The package manager for the agent skills and engineering rules agents follow when they write and review code."
---

Code Rules is the package manager for your agent skills and rules. It records where each skill came from and which version you have, updates it when you choose, shares it across your organization, and installs it for Claude Code, Codex, Cursor, and other agents at once. It also versions, updates, and shares rules: the granular engineering practices that your agents and skills apply.

:::caution[Preview]
Skill support isn't in a released version of Code Rules yet. This page describes how it will work.
:::

Create a `.code-rules` directory in your repo, then take skills and rules from a library, or write your own. Code Rules installs the skills where your agents look for them, and agents configured to read your project guidance follow the same rules.

## Why we built Code Rules

Most people already guide their coding agents with skills. When people talk about skills for AI agents, they usually mean folders that follow the open [Agent Skills](https://agentskills.io) standard: a `SKILL.md` file of instructions that an agent loads when a task calls for it. Skills work well, but managing them doesn't. And skills alone aren't enough.

### Skills need a package manager

Everyone is installing skills from everywhere: a teammate's repository, a vendor's collection, a folder copied from the last project. Before long, you run into problems like these:

- **Unclear provenance.** Once a skill folder is copied in, there's no record of where it came from, who maintains it, or what license it carries.
- **No updates.** You can't tell which version of a skill a repository has, or whether a newer one exists. An improvement made in one place never reaches the other copies unless someone copies it over by hand.
- **No way to share across an organization.** Each repository has its own copy, and the copies drift apart. Nothing says which skills every project should have.
- **One agent at a time.** Claude Code reads skills from `.claude/skills/`, while Codex and Cursor read `.agents/skills/`. Using more than one agent means keeping several copies in step.

We've seen this before! Teams used to copy third-party code into each project by hand, and ran into every one of these issues. That's exactly the problem **package managers** like `npm` and `go mod` solve: shared code lives in one place, each project declares what it depends on and pins a version, and each project adopts updates on its own schedule.

Code Rules does the same for skills. Each project declares where its skills come from in one configuration file. Code Rules records each skill's source and exact revision, moves it to a newer one only when you run `code-rules project update`, and installs it into the folders Claude Code, Codex, Cursor, and other agents read. An organization can publish shared skills in its own `.code-rules` repository, and require or recommend them for every project that uses it.

If you publish skills, you publish them in a [library](/concepts/libraries/), in a format tools can read: a manifest that lists the skills it offers and their levels, a version for each skill, a change note for each change, and metadata that says which rules a skill applies. You can still take skills from any Git repository of skill folders.

### Skills aren't enough

Skills are coarse-grained. They're good for playbooks, techniques, and coordinating a whole task, like building a feature or reviewing a pull request. But much of what makes code good is smaller than a task: test the failure path, keep the context of an error, label every form control. Put those practices inside a skill, and a project can't take one without the rest, or keep a practice the same across every skill that needs it.

So Code Rules adds [rules](/concepts/rule/). A rule is one engineering practice in a Markdown file: what to do, when it applies, and how to check the work. Rules live in rule groups, such as Go or testing, and agents read only the rule groups that match their task. A skill cites the rules each step applies instead of restating them. When a project excludes or replaces a rule, or a rule gets a new version, every skill that cites it follows.

Code Rules manages rules and skills together: for libraries that publish them, organizations that standardize on them, and projects that use them.

## How Code Rules works

Code Rules is built around just a few simple concepts.

### Rules

You write a single best practice as a [rule](/concepts/rule/), written as a Markdown file. A rule tells an agent what to do, when it applies, and how to apply it.

### Rule groups

You organize rules into [rule groups](/concepts/groups/). There are two kinds of rule groups: technology groups, for a _technology_ like Go, Tailwind, Next.js, Typescript, or Playwright, and practice groups, for a _practice_ like testing, logging, or error handling.

### Skills

A [skill](/concepts/skill/) is an Agent Skills folder with a reusable procedure, such as how to build a feature or review a pull request. Rules say what good work looks like; skills say what to do and in what order, citing the rules each step applies.

### Projects

You declare the rules and skills you want in a [project](/concepts/project/), specifically in the `/.code-rules` folder in your project repo. You (or your agent) can use the `code-rules` CLI to initialize a new project, and add your own rules, rule groups, and skills. We recommend committing `/.code-rules`, and the skills Code Rules installs, to version control.

### Libraries

Finally, you probably want to include some pre-existing rules and skills from a [library](/concepts/libraries/), a repository that publishes rule groups and skills for other projects. A library can require or recommend skills for every project that uses it. You can also install skills from a Git repository that isn't a library, such as a third party's skill collection. Libraries can optionally declare license terms, which Code Rules carries into your project.

### RULES.md

Code Rules combines your own rules and the ones you include from sources into guidance under `.code-rules/generated/`. Agents start with `RULES.md`, an index that tells them which rule groups to read.

An abbreviated `RULES.md` entry:

```md
### Testing

**When to read this group:** When changing behavior or fixing bugs.

**Open group:** [Testing](groups/practices/testing.md)
```

The linked rule group page contains the rules in full, or links to their full definitions. For example, it might tell an agent to add a regression test when fixing a bug. See [the generated file layout](/reference/files/) for details.

### Installed skills

Code Rules writes each installed skill into `.claude/skills/` and `.agents/skills/`, where coding agents find skills, and lists them in `.code-rules/generated/SKILLS.md`. Each installed skill that cites rules ends with links to your project's version of every rule it applies. See [Skill](/concepts/skill/#how-agents-use-installed-skills).

## What Code Rules does not do

Code Rules assembles rules and skills from multiple sources into a cohesive, agent-friendly collection of files in your repo, and checks that those files are valid and current.

Code Rules installs skills into your repository, not on your computer. It doesn't manage the personal skills a person installs for all of their repositories, though it warns when one has the same name as a skill your project installs.

That is where Code Rules stops. It installs and checks skills, but it doesn't run them, decide when your agents use them, or check that your agents followed them or your rules. At least for now, how your agents consume rules and skills is up to you, though we'll share best practices as we discover them.
