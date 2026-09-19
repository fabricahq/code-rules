---
title: "Rule"
description: "A best practice expressed as one clear, scoped, verifiable engineering instruction."
---

A **rule** expresses one engineering best practice as a Markdown file.
It tells an agent what to do, when the instruction applies, and what evidence would show that the work follows it.
A rule can guide code, tests, plans, documentation, or other engineering work.

## One best practice, one rule

For example, a team might declare:

> When adding or changing bounded retries, test that requests stop at the configured limit.

That rule gives an implementing agent a concrete test to write.
A reviewing agent can check whether the test would catch an extra retry.
The rule applies to retry behavior; it does not require every operation to add retries.

Keep independently adoptable obligations in separate files.
A project should be able to import, exclude, or replace one rule without also changing unrelated expectations.

## What a rule contains

The Markdown body states the obligation and its applicability.
Rationale explains why it matters, examples clarify the intended behavior, and verification guidance tells an agent what evidence to look for.
Required metadata records the title, `whenToRead`, impact, and the consequence the rule addresses.
Optional tags supply search terms; they do not select rules.

The [rubric and template](/reference/rule-authoring/) define the authoring standard.
Agents can follow them directly. An installable [Code Rules skill](/guides/write-rules/) is planned.

## Where a rule lives

Each rule has one home in a [rule group](/concepts/groups/).
For example, a library can store the retry rule at:

```text
practices/testing/verify-retry-limits.md
```

Keep a rule local to a project, or publish it in a library that other projects import.
Library rules are versioned with their repository; a project's exact ref or version constraint selects the version it imports.
A project can add local rules or explicitly exclude and replace imported rules.

## How agents identify and use it

The library-relative path without `.md` identifies a rule within its library.
Generated files add the configured source name, such as `fabrica:practices/testing/verify-retry-limits`.
That prefix distinguishes matching paths from different libraries.

Agents select rules from group pages, which include complete definitions for small groups and link to individual resolved rule files for larger groups. Imported and local choices have already been resolved.
Each rule keeps its identity and provenance even when several rules share a group applicability index.
Agents select relevant groups, then apply each individual rule's conditions and exceptions.

See [Group](/concepts/groups/) for how rules are organized, or [Write a rule](/guides/write-rules/) to create one.
