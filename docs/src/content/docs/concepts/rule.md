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

A rule is a Markdown file with two parts:

- **Metadata at the top** gives the rule a title and tells agents when to read it and how important it is.
- **Guidance in the body** explains what to do and when it applies. Add rationale, examples, and ways to check the result when they help make the rule clear.

For the required fields, see [Rule and library format](/reference/rule-library-format/#rule-metadata). The [rubric and template](/reference/rule-authoring/) offer writing advice. To write your own, follow [Write a rule](/guides/write-rules/).

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

When you build or sync a project, Code Rules combines its local and imported rules into generated group files. Each group file contains the rules in full, or summaries with links to the full rules when the group is large. The project's exclusions and replacements determine which rules appear.

Agents read the generated guidance in this order:

1. Open `RULES.md` to find the groups relevant to the task.
2. Read those groups and their rules, following links to full rule files where needed.
3. Choose which rules apply to the task, using each rule's conditions and exceptions, and follow them.

For example, an agent changing retry behavior would read the Testing group, then apply the rule about verifying retry limits.

See [Group](/concepts/groups/) for how groups guide reading, or [Project files](/reference/files/#example-follow-a-rule-from-the-index) to follow an example from `RULES.md` to a rule.
