---
title: "Plan, write, and review"
description: "A suggested workflow for agents using the project\u2019s effective engineering rules."
---

This page suggests an agent workflow for using the project's committed resolved rules during planning, implementation, and review.
Adapt it to your project, or use your own prompts and tooling. Code Rules supplies the rule files; it does not run this workflow or require a particular validation or enforcement method.
See [product scope](/overview/#scope-rule-management-and-delivery).

## Project instructions

To use this workflow, add the following section to the project's existing `AGENTS.md`, or use it as a starting point for a direct agent prompt:

```markdown
Before planning, implementing, or reviewing a change, read `.code-rules/generated/RULES.md`.
Select technology and practice groups using the task, affected behavior, and code.
Inspect the selected group pages, including every part of a split index, and use each rule's whenToRead guidance to select rules.
Read full definitions where included, or follow each applicable rule's Read full rule link.
Read every relevant or plausibly relevant definition completely before relying on it.
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
3. Inspect matching group pages, including all parts of any summary index.
4. Use `whenToRead` to select relevant or plausibly relevant rules. Read their complete definitions inline or through explicit **Read full rule** links.
5. Account for applicable obligations in the plan and implementation.
6. Revisit selection if the work expands.

### Example: retry a failed request

For retries in a TypeScript service, consider `techs/typescript`, `practices/testing`, `practices/observability`, and `practices/error-handling`.
Inspect the intended behavior, dependencies, imports, surrounding code, and changed files.
Practice groups can apply even when no test files or logging packages change.

Inspect a group index when its guidance matches the work, then read the relevant full rules and apply their conditions and exceptions.
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
The planned `code-rules check` command will establish file consistency, not application compliance.

## Handle gaps and conflicts

If a relevant group is missing, report the missing coverage.
If resolved rules conflict, identify both IDs and ask the project owner to resolve the intended policy.
Use [Conflicting guidance](/guides/conflicting-guidance/) for the review criteria and explicit resolution options.
Keep the pinned ruleset during ordinary work; adopting upstream changes is a separate project update.

## Write or review rules themselves

When the task changes a rule, use the [authoring rubric and template](/reference/rule-authoring/).
The planned [Code Rules authoring skill](/guides/write-rules/#write-with-the-skill) guides this workflow.
Until the skill ships, follow those documents directly.

Read the target library's conventions and evaluate each rule against every rubric criterion.
Report unmet criteria with the relevant passage and a concrete revision.
Separate unclear wording from unresolved engineering policy; ask the author to resolve the latter.
