---
title: "What is Code Rules?"
description: "Share engineering standards across projects and give agents clear, consistent expectations."
---

**Code Rules gives your projects shared, versioned libraries of engineering standards.**
A project chooses the rules it needs, records its exceptions, and commits the effective rules for agents to read.
Those expectations guide planning, implementation, and review.

## Why use it?

Your projects may share a language and testing philosophy while making different choices about compatibility or architecture.
Copying instructions by hand makes those differences hard to maintain.
Loading a shared document directly can introduce new obligations without a project choosing to adopt them.

Code Rules makes that choice explicit.
Choose each library's commit or tag, select its groups, and review updates like any other change to the project.

## What belongs in a rule library?

Rules cover both specific technologies and broader engineering practices:

- **Technologies:** TypeScript, React, Go, and Playwright.
- **Practices:** testing, observability, error handling, and architecture.

A [rule](/concepts/rule/) describes an expectation and the conditions under which it applies.
A practice rule can use a TypeScript example without becoming a TypeScript-only rule.

## How rules are written

Each rule is a Markdown file.
A shared [rubric and template](/reference/rule-authoring/) help authors make the obligation clear, scoped, and verifiable.
The planned [Code Rules authoring skill](/guides/write-rules/) uses those references to help agents draft, revise, and review rules.

## What a project controls

Your project chooses its libraries, selects each commit or tag, and selects groups from each source.
Your project can add local rules, exclude inherited rules with a reason, or replace a rule completely.
The importer combines those choices into one effective file per group.

Writing and reviewing agents use the same files.
Each agent still decides which groups and individual rules apply to the work at hand.

## Where Code Rules fits

The public tool defines formats and handles imports.
Libraries own engineering opinions and can be public or private.
Projects own their exceptions and the version they adopt.

Product vision, architecture facts, and domain context remain in your project documentation.
Your software factory owns review orchestration, approvals, and merge policy.
Code Rules provides the expectations those systems can use.

## Start exploring

1. [Walk through using rules in a project](/guides/use-rules/).
2. Learn how a [Group](/concepts/groups/) organizes rules by technology or practice.
3. [Adapt a rule](/guides/select-rules/#adapt-the-import-to-your-project) for a project's needs.
4. Read the [agent instructions](/for-agents/) for planning, writing, and review.
