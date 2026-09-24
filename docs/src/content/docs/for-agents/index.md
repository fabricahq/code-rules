---
title: "Plan, write, and review"
description: "A suggested workflow for agents using the project\u2019s resolved engineering rules."
---

This page suggests an agent workflow for using the project's committed resolved rules during planning, implementation, and review.
Adapt it to your project, or use your own prompts and tooling. Code Rules supplies the rule files; it does not run this workflow or require a particular validation or enforcement method.
See [product scope](/start-here/overview/#what-code-rules-does-not-do).

## Project instructions

To use this workflow, add the following section to the project's existing `AGENTS.md`, or use it as a starting point for a direct agent prompt:

```markdown
Before planning, implementing, or reviewing a change, read `.code-rules/generated/RULES.md`.
Compare each group's When to read this group cue with your task, the affected behavior, and the code. Use its Description to understand the scope.
Open every relevant or plausibly relevant group.
Read every rule in each opened group completely, including every page of a split index. Read full definitions inline or follow every Read full rule link.
Then use each rule's When to read cue, guidance, and exceptions to determine whether it applies.
Complete truncated reads. Revisit rule selection when scope changes and reload needed rules after compaction.
Follow every applicable rule regardless of impact, including its exceptions.
After reading the complete rule, use Implementation guidance when planning or changing code and Validation guidance when reviewing, testing, or diagnosing behavior, when those sections are present.
Use both when the task includes both activities. These sections support the rule’s guidance; they do not replace it.
When reviewing, select groups independently and cite rule IDs with evidence for findings. Assess finding severity from concrete consequences.
Report missing relevant groups as coverage gaps.
```

The generated index is an explicit entry point.
Do not rely on automatic discovery of nested `AGENTS.md` files to load the rules.

## Plan and implement

1. Read the task and the generated index.
2. Identify affected technologies and engineering practices.
3. Compare each group's **When to read this group** cue with the work. Use its **Description** to understand scope. Open every relevant or plausibly relevant group.
4. Read every rule in each opened group completely, including all pages and linked full definitions. Then assess applicability using each rule's **When to read** cue, guidance, and exceptions.
5. Account for applicable obligations in the plan and implementation.
6. Revisit selection if the work expands.

### Example: retry a failed request

For retries in a TypeScript service, consider `techs/typescript`, `practices/testing`, `practices/observability`, and `practices/error-handling`.
Inspect the intended behavior, dependencies, imports, surrounding code, and changed files.
Practice groups can apply even when no test files or logging packages change.

Open each relevant or plausibly relevant group, read every rule in it completely, then apply the rules whose conditions match the work. Respect their exceptions.
A group match alone is not evidence of a violation.

A behavior change can require testing rules before anyone edits a test file.
When an obligation depends on local contracts, read project context alongside the rules.

Completion means the relevant rules informed the work, including their scope and exceptions.
Reading the index alone is not enough.

## Review independently

Select groups from the requested behavior and implementation, rather than accepting the writing agent's selection as complete.
For each finding, cite the resolved rule ID, applicable condition, observed evidence, and practical consequence.

Impact describes the consequence a rule helps prevent; it does not determine applicability, override exceptions, or set finding severity.
Read **Why it matters** for context and assess the actual consequence of each finding.

Separate confirmed failures from hypotheses that need verification.
`code-rules project check` establishes file consistency, not application compliance.

## Handle gaps and conflicts

If a relevant group is missing, report the missing coverage.
If resolved rules conflict, identify both IDs and ask the project owner to resolve the intended policy.
Use [Resolve conflicting rules](/guides/conflicting-guidance/) for the review criteria and explicit resolution options.
Keep the pinned ruleset during ordinary work; adopting upstream changes is a separate project update.

<span id="write-or-review-rules-themselves"></span>

## Manage and author rules

When the task changes rules or their configuration, use the matching guide below. The guides provide the working directory, commands, files to edit, and checks for each task.

| Task | Guide |
| --- | --- |
| Add an original rule to an existing project | [Write a rule](/guides/write-rules/#add-a-rule-to-an-existing-project) |
| Declare terms for rules you publish | [License a library](/guides/license-rules/) |
| Turn external guidance into a rule with retained attribution and terms | [Import rules from another source](/guides/select-rules/#from-another-source) |
| Import groups from an existing library | [Import rules](/guides/select-rules/) |
| Add project guidance or exclude and replace imported rules | [Customize imported rules](/guides/customize/) |
| Publish a collection of rules for projects to import | [Create a library](/start-here/create-library/) |
| Adopt newer library versions or change imported groups | [Update rules](/guides/update/) |
| Investigate incompatible instructions and record the owner's decision | [Resolve conflicting rules](/guides/conflicting-guidance/) |

When the task changes a rule, follow the target library or project's authoring conventions and the [required file format](/reference/rule-library-format/).
The [rubric and template](/reference/rule-authoring/) offer writing advice, not a pass/fail checklist.

Use relevant rubric criteria to suggest improvements with a concrete passage and revision. A valid short rule does not need every template section.
Separate unclear wording from unresolved engineering policy; ask the owner to resolve the latter.
