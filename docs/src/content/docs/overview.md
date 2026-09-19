---
title: "What is Code Rules?"
description: "The package manager for engineering rules that agents follow when they write and review code."
---

Code Rules is the package manager for your engineering rules.

You write the rules you want agents to follow, or you adopt them from a library. Each project picks the rules it needs, and every agent working on that project reads the same guidance.

## What traditional package managers do

Almost every codebase depends on some code it did not write. Before "code-native" package managers, teams would manually copy third-party code into each project. The different copies inevitably drifted apart, nobody knew which version of the imported code they had, and "upstream" fixes were painful to adopt.

Traditional package managers like `npm` and `go mod` solve this. Shared code lives in one place, typically a git repo. Each project names the library or module to import, and pins the version it wants. When the library improves, projects import updates on their own schedule.

## Agents need the same thing for guidance

With AI-led coding, we depend heavily on _prompts_ we did not write, or do not want to write every time.

Today that guidance lives in an AGENTS.md file. Portions of it get copied from repo to repo and edited a little each time. Over time, every project has its own collection of best practices..

In short, today's tools for assembling context give us limited control to construct exactly the prompt we want.

## Introducing Code Rules!

Code Rules is the package manager for your engineering rules.

### Rules

You write a single best practice as a [rule](/concepts/rule/), written as a Markdown file. A rule tells an agent what to do and when it applies, and gives both examples and counter-examples.

### Groups

You organize rules into [groups](/concepts/groups/), where a group represents either a _technology_ like Go, Tailwind, Next.js, Typescript, or Playwright, or a _practice_ like testing, logging, or error handling.

You write your "local" rules in a `/.code-rules` folder in your repo, using the `code-rules` CLI for scaffolding and guidance. If you wish, you can stop there.

Or you can import existing rules from

## Adopt the rules a project needs

Projects may share a language and a testing philosophy, and still disagree about compatibility or architecture.
Copying instructions by hand makes those differences hard to maintain.
Loading a shared document wholesale can add obligations the project never chose.

Code Rules makes that choice explicit.
You pick each library's revision or version constraint, select its groups, and review updates like any other change.

A project can add local rules, exclude an imported rule with a reason, or replace one completely.
Build turns those choices into group pages and full rule files. Small groups include complete rules. Larger groups use short summaries with links to the full text.

Writing and reviewing agents use the same files.
Each agent still decides which groups and rules apply to the work at hand.

## What belongs in a library

Rules cover both specific technologies and broader engineering practices:

- **Technologies:** TypeScript, React, Go, and Playwright.
- **Practices:** testing, observability, error handling, and architecture.

A practice rule can use a TypeScript example without becoming a TypeScript-only rule.

## How rules are written

Each rule is a Markdown file.
A shared [rubric and template](/reference/rule-authoring/) help authors make the obligation clear, scoped, and verifiable.
Use those references with any agent or editor. An installable [Code Rules authoring skill](/guides/write-rules/) is planned.

## How agents read the rules

We researched how coding agents discover and apply rules: vendor docs, published studies, and reported tests of reading limits.
Large rule bundles can exceed those limits. Selective loading can miss a relevant rule.

That shaped the delivery format. Small groups include full rules. Larger groups use short applicability indexes with explicit links to the full text.
The size threshold is a delivery choice, not a measured guarantee of compliance.
The suggested workflow has agents consider both technologies and practices before they write code, then select rules independently during review.

Reading a rule does not prove compliance. Review still needs concrete evidence.
We found no controlled comparison that crowns one delivery format as best.
The [agent instructions](/for-agents/) describe how to use this approach.

## What Code Rules does not do

Code Rules manages which versioned rules a project adopts and generates readable files with traceable origins.
Libraries supply the engineering opinions. Each project chooses which ones to adopt.

Your project decides how agents apply those rules, and how you validate or enforce them.
Code Rules does not pick an agent, run reviews, or check whether application code follows the rules.
You can prompt an agent directly, add instructions to `AGENTS.md`, or plug in your own review agents, linters, tests, and CI gates.
The [agent instructions](/for-agents/) are a suggested integration you can adapt.

The `check` command confirms generated rules match their inputs. It does not inspect application code.
Product vision, architecture facts, and domain context stay in your project documentation.
See [Project status](/status/) for what the CLI implements and what is still planned.

## Start exploring

1. [Walk through using rules in a project](/guides/use-rules/).
2. Learn how a [Group](/concepts/groups/) organizes rules by technology or practice.
3. [Adapt a rule](/guides/select-rules/#adapt-the-import-to-your-project) for a project's needs.
4. Read the [agent instructions](/for-agents/) for planning, writing, and review.
