---
title: "What is Code Rules?"
description: "The package manager for engineering rules that agents follow when they write and review code."
---

Code Rules is the package manager for your engineering rules.

## In a nutshell

To use Code Rules:

1. Run `code-rules project init` from your project root to create `.code-rules/config.yaml`.
2. Write the rules you want agents to follow, or import existing rules from a rule library.
3. Run `code-rules project sync` and Code Rules will use your configuration and rules to generate a collection of Markdown documents optimized for agents to read.

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

- **Drift.** Each project has its own version of your best practices.
- **No versioning.** You can't tell which version of a rule a given repo has.
- **Improvements don't spread.** You improve a rule, but only one project gets the update.
- **Unclear provenance.** Once a rule is pasted in, there's no record of where it came from.
- **Hand-merging.** Combining guidance from several sources often means merging by hand.

We've seen this before! Teams used to copy third-party code into each project by hand, and ran into every one of these issues. That's exactly the problem **package managers** like `npm` and `go mod` solve: shared code lives in one place, each project declares what it depends on and pins a version, and each project adopts updates on its own schedule.

Code Rules does the same, but for agent guidance.

## How Code Rules works

Code Rules is built around just a few simple concepts.

### Rules

You write a single best practice as a [rule](/concepts/rule/), written as a Markdown file. A rule tells an agent what to do, when it applies, and how to apply it.

### Groups

You organize rules into [groups](/concepts/groups/), where a group represents either a _technology_ like Go, Tailwind, Next.js, Typescript, or Playwright, or a _practice_ like testing, logging, or error handling.

### Projects

Your [project](/concepts/project/) is the codebase whose rules Code Rules manages. Its `.code-rules/` directory holds the configuration and rules. You (or your agent) can use the CLI to set up Code Rules in your project and add local rules and groups. We recommend committing `.code-rules/` to version control.

### Libraries

Finally, you probably want to import some pre-existing rules from a [library](/concepts/libraries/), a collection of rule groups maintained separately for projects to reuse. Libraries can optionally declare license terms, which Code Rules carries into your project.

### RULES.md

Code Rules combines project-only and imported rules into guidance under `.code-rules/generated/`. Agents start with `RULES.md`, an index that tells them which groups to read.

An abbreviated `RULES.md` entry:

```md
### Testing

**When to read this group:** When changing behavior or fixing bugs.

**Open group:** [Testing](groups/practices/testing.md)
```

The linked group page contains the rules in full, or links to their full definitions. For example, it might tell an agent to add a regression test when fixing a bug. See [the generated file layout](/reference/files/) for details.

## What Code Rules does not do

Code Rules assembles collections of rules from multiple sources (libraries) into a cohesive, agent-friendly collection of local files in your repo.

That is where Code Rules stops. It is not opinionated about how your agents consume these rules. At least for now, that part is up to you, though we'll share best practices as we discover them.
