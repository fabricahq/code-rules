---
title: "What is Code Rules?"
description: "The package manager for engineering rules that agents follow when they write and review code."
---

Code Rules is the package manager for your engineering rules.

You write the rules you want agents to follow, or you adopt them from a library. Each project picks the rules it needs, and every agent working on that project reads the same guidance.

## What traditional package managers do

Almost every codebase depends on some code it did not write. Before "code-native" package managers, teams would manually copy third-party code into each project. The different copies inevitably drifted apart, nobody knew which version of the imported code they had, and "upstream" fixes were painful to adopt.

Traditional package managers like `npm` and `go mod` solve this. Shared code lives in one place, typically a git repo. Each project names the library or module to import, and pins the version it wants. When the library improves, projects import updates on their own schedule.

## Agents need a package manager for guidance

With AI-led coding, we depend heavily on _prompts_ we did not write, or do not want to write every time.

Today that guidance lives in an AGENTS.md file. Portions of it get copied from repo to repo and edited a little each time. Over time, every project has its own collection of best practices..

In short, today's tools for assembling context give us limited control to construct exactly the prompt we want.

## How Code Rules works

Code Rules is the package manager for your engineering rules.

### Rules

You write a single best practice as a [rule](/concepts/rule/), written as a Markdown file. A rule tells an agent what to do, when it applies, and how to apply it.

### Groups

You organize rules into [groups](/concepts/groups/), where a group represents either a _technology_ like Go, Tailwind, Next.js, Typescript, or Playwright, or a _practice_ like testing, logging, or error handling.

### Projects

You declare the rules you want in a [project](/concepts/project/), typically in a `/.code-rules` folder in your project repo. You (or your agent) can use the `code-rules` CLI to initialize a new project, and add local rules and groups. We recommend that you commit `/.code-rules` to version control.

### Libraries

Finally, you probably want to import some rules from a [library](/concepts/libraries/), which is a collection of rules and groups maintained by a third-party. Libraries can declare license terms, which Code Rules carries into your project. If a library has no license declaration, inspect its terms before using or sharing its rules.

## What Code Rules does not do

Code Rules assembles collections of rules from multiple sources (libaries) into a cohesive, agent-friendly collection of local files in your repo.

That is where Code Rules stops. It is not opinionated about how your agents consume these rules. At least for now, that part is up to you, though we'll share best practices as we discover them.
