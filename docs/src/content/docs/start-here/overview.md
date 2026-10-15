---
title: "What is Code Rules?"
description: "The package manager for the agent skills and engineering rules agents follow when they write and review code."
---

Code Rules is the package manager for your agent skills and rules.

Create a `.code-rules` directory in your repo, then write the rules you want agents to follow, or take rules and skills from a library. Agents configured to read your project guidance can then use the same rules, and Code Rules installs the skills where your agents look for them.

## Why we built Code Rules

We've all heard the buzz:

"Code is cheap!"
"Coding is solved!"
"The code is no longer the blocker!"

But if coding is so cheap and trivial, why do agents so often produce a mess of unmaintainable code? Worse, why do they so often cut corners on the practices serious software depends on, like logging, error handling, testing, and observability?

The reality is that agents are _capable_ of writing testable, maintainable, well-organized code, but _they don't do it by default._ They do it when you tell them how. That makes the guidance you give your agents one of the biggest levers on the quality of the code they write.

We wrote Code Rules to help individuals and teams manage that guidance, whether for a single project or at scale.

### Why managing agent guidance is hard

Once you've written good guidance for your agents, you want it in every project. So you copy chunks of your `AGENTS.md` into the next repo, tweak them a little, and move on. Then you do it again. Before long, you run into problems like these:

- **Drift.** Every copy has been tweaked, and each project has its own version of your best practices.
- **No versioning.** You can't tell which version of a rule a given repo has, or whether it's current.
- **Improvements don't spread.** When you improve a rule in one project, the other projects never get it unless someone copies it over by hand.
- **Unclear provenance.** Once a rule is pasted in, there's no record of where it came from, who maintains it, or what license it carries.
- **Hand-merging.** Combining guidance from several sources means merging it into one file by hand, with no clean way to take only the parts you want.

We've seen this before! Teams used to copy third-party code into each project by hand, and ran into every one of these issues. That's exactly the problem **package managers** like `npm` and `go mod` solve: shared code lives in one place, each project declares what it depends on and pins a version, and each project adopts updates on its own schedule.

Code Rules does the same for agent guidance.

## How Code Rules works

Code Rules is built around just a few simple concepts.

### Rules

You write a single best practice as a [rule](/concepts/rule/), written as a Markdown file. A rule tells an agent what to do, when it applies, and how to apply it.

### Rule groups

You organize rules into [rule groups](/concepts/groups/). There are two kinds of rule group: technology groups, for a _technology_ like Go, Tailwind, Next.js, Typescript, or Playwright, and practice groups, for a _practice_ like testing, logging, or error handling.

### Skills

A [skill](/concepts/skill/) is a reusable procedure, such as how to build a feature or review a pull request. Rules say what good work looks like; skills say what to do and in what order, citing the rules each step applies.

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

That is where Code Rules stops. It installs and checks skills, but it doesn't run them, decide when your agents use them, or check that your agents followed them or your rules. At least for now, how your agents consume rules and skills is up to you, though we'll share best practices as we discover them.
