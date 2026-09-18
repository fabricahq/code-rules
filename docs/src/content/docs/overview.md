---
title: "What is Code Rules?"
description: "Share engineering standards across projects and give agents clear, consistent expectations."
---

**Code Rules manages your engineering rules and practices so that your agents code the way you want them to.**
A project selects versioned rule libraries, records local additions and exceptions, and commits its resolved rules alongside the code.
Teams can review and adopt rule updates over time, then use those rules during planning, implementation, and review.

## Why use it?

Your projects may share a language and testing philosophy while making different choices about compatibility or architecture.
Copying instructions by hand makes those differences hard to maintain.
Loading a shared document directly can introduce new obligations without a project choosing to adopt them.

Code Rules makes that choice explicit.
Choose each library's exact revision or version constraint, select its groups, and review updates like any other change to the project.

## What belongs in a rule library?

Rules cover both specific technologies and broader engineering practices:

- **Technologies:** TypeScript, React, Go, and Playwright.
- **Practices:** testing, observability, error handling, and architecture.

A [rule](/concepts/rule/) describes an expectation and the conditions under which it applies.
A practice rule can use a TypeScript example without becoming a TypeScript-only rule.

## How rules are written

Each rule is a Markdown file.
A shared [rubric and template](/reference/rule-authoring/) help authors make the obligation clear, scoped, and verifiable.
Use those references with any agent or editor. An installable [Code Rules authoring skill](/guides/write-rules/) is planned.

## How research shaped rule delivery

We researched how coding agents discover and apply rules, reviewing agent-tool documentation, published studies, and reported tests of reading limits.
The evidence highlighted two risks: large rule bundles can exceed reading limits, while selective loading can miss relevant rules.
That informed our design: small groups include full rules, while larger groups use compact applicability indexes with explicit links to full definitions.
The configurable size threshold is a delivery choice, not a measured guarantee of compliance.
Our suggested agent workflow considers both technologies and practices before writing code, then independently selects relevant rules during review.
Reading a rule does not prove compliance; review still needs concrete evidence.
We found no controlled comparison establishing one delivery format as universally best.
The [agent instructions](/for-agents/) describe how to use this approach.

## What a project controls

Your project chooses its libraries, selects an exact revision or version constraint, and selects groups from each source.
Your project can add local rules, exclude inherited rules with a reason, or replace a rule completely.
Builds resolves those choices into group pages and individual files containing each resolved rule's full text. Small groups include complete rules; larger groups use applicability summaries with explicit reading links.

Writing and reviewing agents use the same files.
Each agent still decides which groups and individual rules apply to the work at hand.

## Scope: rule management and delivery

Code Rules associates a codebase with managed, version-controlled, updatable engineering rules. It defines library and configuration formats, manages adopted revisions and project exceptions, and generates readable files with traceable origins.
Libraries supply the engineering opinions; each project chooses which ones to adopt.

Your project decides how to get agents to apply those rules and how to validate or enforce compliance. Code Rules does not prescribe an agent, review process, validation tool, or enforcement mechanism, and it does not run code reviews or enforce application compliance.
You can prompt an agent directly, add instructions to the project's `AGENTS.md`, or build or integrate separate tooling such as review agents, linters, tests, and CI gates.
The [agent instructions](/for-agents/) are a suggested integration you can adapt to your workflow.

The offline builder validates rule inputs and generates their effective output. The `check` command checks that generated rules match their inputs; it does not check whether application code follows those rules.
Product vision, architecture facts, and domain context remain in your project documentation.
See [Project status](/status/) for which rule-management capabilities are implemented and which are still planned.

## Start exploring

1. [Walk through using rules in a project](/guides/use-rules/).
2. Learn how a [Group](/concepts/groups/) organizes rules by technology or practice.
3. [Adapt a rule](/guides/select-rules/#adapt-the-import-to-your-project) for a project's needs.
4. Read the [agent instructions](/for-agents/) for planning, writing, and review.
