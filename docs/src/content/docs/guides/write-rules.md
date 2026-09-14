---
title: "Make a rule"
description: "Author a focused engineering obligation with applicability, examples, and verification guidance."
---

Write each rule in Markdown with a rubric that helps agents make it clear, scoped, and verifiable.
Use the Code Rules skill to turn a best practice into a rule, starting from a shared template.
To set up a shared library and create groups before adding rules, follow [Create a rule library](/guides/create-library/).

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


## Adapt third-party rules

For a step-by-step example with inspectable license provenance, follow [Adapt a third-party rule](/guides/adapt-rules/).

The original repository does not have to change. The definition you give Code Rules must use its input format: a technology or practice group with `_group.json`, a Markdown rule with the required metadata, and retained supporting files. The body follows the flexible authoring rubric; it does not need every template heading.

For an existing compatible Code Rules library, use the normal import workflow. For a linter rule, style guide, skill, or other document that is not a compatible library:

1. Identify the exact source revision and establish permission to copy, adapt, and redistribute the material for your intended use. Preserve the applicable license and notices.
2. Create an adapted definition under `local/techs/<group>/` or `local/practices/<group>/`, or in a separately maintained compatible library. Preserve the original separately when useful for reviewing future updates.
3. Add the required metadata and an activity-based `whenToRead` cue. Preserve the obligation, important conditions, exceptions, and examples; explain deliberate changes. A detector's analysis limitations do not automatically become exceptions to a written rule.
4. Declare `licenses` and `attribution` using the [file reference](/reference/files/#rule-specific-licenses-and-attribution). Use a commit-pinned source URL and describe the adaptation. Keep retained terms in the owning root's `licenses/` folder.
5. Review the adaptation against the [authoring rubric](/reference/rule-authoring/), then generate and inspect the effective rule, license links, and provenance. Commit the adapted source and retained notices together.

An agent can help prepare the adaptation, but Code Rules does not currently convert arbitrary repositories automatically. Do not place modified material in `vendor/` and claim it is an unchanged snapshot of the upstream commit. For a local adaptation, the effective origin is local and external provenance is recorded through attribution. A separately published adapted library has its own repository and version while retaining the earlier attribution chain.

For later updates, compare the original pinned material with the new source, then deliberately revise the adaptation. Updating a source citation alone does not establish that the adapted rule incorporates the newer guidance.
