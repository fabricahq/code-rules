---
title: "Make a rule"
description: "Author a focused engineering obligation with applicability, examples, and verification guidance."
---

Write each rule in Markdown with a rubric that helps agents make it clear, scoped, and verifiable.
Use the Code Rules skill to turn a best practice into a rule, starting from a shared template.

## Rubric, template, and skill

- The [authoring rubric](/reference/rule-authoring/#authoring-rubric) defines what makes a useful rule and how to review it.
- The [Markdown template](/reference/rule-authoring/#markdown-template) gives each rule a consistent starting structure.
- The Code Rules authoring skill guides an agent through drafting, revising, and reviewing rules using those two references.

The rubric is authoritative.
The skill references it rather than maintaining a separate copy of the authoring guidance.
You can also use the rubric and template without a skill, with any agent or editor.

:::note[Skill availability]
The installable authoring skill is planned and has not shipped.
The rubric and template are available here as part of the design preview.
The workflow below describes how the skill is intended to work.
:::

## Write with the skill

1. Give the agent a best practice, its intended scope, and any supporting context or sources.
2. The skill reads the rubric, template, and target library's conventions.
3. The agent drafts one rule per file, specifying `whenToRead` and separately considering implementation and validation guidance.
4. The agent checks the draft against every rubric criterion and revises unclear passages.
5. The agent presents the rule and any unresolved policy questions for review.

The same skill can revise an existing rule or review a proposed rule against the rubric.
It should ask about missing policy decisions instead of silently choosing them for the library owner.

## Choose a group

Use a technology group when the obligation depends on a named technology.
Use a practice group when it transfers across technologies.
Place project-specific contracts in the applicable project's local rules.

For a new group, add [group metadata](/reference/files/#group-metadata) that helps agents recognize relevant work.

## Example rule

This original example illustrates the proposed format:

````md
---
title: Verify retry limits
whenToRead: "When planning, implementing, reviewing, or diagnosing bounded retry behavior."
impact: HIGH
impactDescription: prevents a transient failure from causing unbounded requests
tags: testing, retries
---

## Verify retry limits

When a change adds bounded retries, test that requests stop after the configured limit.

### Implementation

Write a deterministic test that keeps the request failing until the configured attempt limit.
It does not require a retry mechanism where none is needed.

### Rationale

A retry limit prevents repeated failures from producing unbounded requests.
A test should catch changes that accidentally bypass that limit.

### Incorrect example

Make the first request fail and the second succeed, then assert success.
This verifies recovery but never reaches the retry limit.
The test would still pass if the operation allowed unlimited attempts.

### Correct example

For a limit of three attempts, make every attempt fail.
Assert that the operation stops after three calls and returns the documented failure.

### Validation

Run the test with the stop condition removed in a disposable checkout.
The test should fail.
````

## Keep identity stable

A rule's library ID is its relative path without `.md`.
Consuming projects qualify it with their configured source name, such as `fabrica:practices/testing/verify-retry-limits`.
Renaming or moving the file changes its ID in the first release.
Consumers must update exclusions and replacements that referenced the old path.

Keep source attribution in the rule's metadata or Markdown body and preserve any required notices.
Builds carries that attribution into the individual generated rule file; no separate attribution file is required.
The existing nested `source:` metadata describes provenance, not an override target.
When publishing or adapting rules, follow [License rules](/guides/license-rules/) to make permissions and attribution explicit.
