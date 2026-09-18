---
title: "What is Code Rules?"
description: "The package manager for engineering rules that agents follow when they write and review code."
---

**Code Rules is the package manager for your engineering rules.**

You write the rules you want agents to follow, or you import them from a library. Each project adopts the ones it needs, and ultimately every agent reads the same guidance.

## What traditional package managers do

When you write TypeScript or JavaScript, you install packages from `npm`. For Go, you use `go mod`. For Rust, you use Cargo and `crates.io`. Almost every language has a tool like this.

A package manager *pulls in third-party code in an organized way.* You name the library and the version you want. The package manager makes that code available to you and to everyone else on the project.

## Agents need the same thing for guidance

AI-native software engineering adds a new layer to the stack. We humans tell the agent what to build, and (ideally) how to build it. Agents write the actual code.

But how can we build confidence that agents are writing code the way we want? Will they write deep modules over shallow modules? Hide complexity behind simple interfaces? Pull complexity down instead of onto callers?

And what about language-specific practices? If we're writing Go code, will it faithfully follow our team's preferences?

Ultimately, we want agents to be equipped with the context of all our team's best practices when writing or validating code.

What fragments of context we give it, on which projects, and in which scenarios is what Code Rules is meant to solve. Better yet, if someone else has already written good code rules, we should be able to take advantage of them instead of reinventing the wheel!

## What a rule is

A [rule](/concepts/rule/) is one of those practices, written as a Markdown file.
It tells an agent what to do, when the instruction applies, and what evidence would show the work follows it.

Your team already has practices like these in reviews, prompts, and docs. Some cover a language or framework. Others cover testing, errors, or how you design systems.

For example, a team might write:

> When adding or changing bounded retries, test that requests stop at the configured limit.

An implementing agent gets a concrete test to write.
A reviewing agent can check that the test would catch an extra retry.
The rule applies to retry behavior. The rule does not require every operation to add retries.

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
