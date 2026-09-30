---
title: "Plan, write, and review"
description: "A suggested workflow for agents using the project\u2019s resolved engineering rules."
---

This page suggests an agent workflow for using the project's committed resolved rules during planning, implementation, and review.
Adapt it to your project, or use your own prompts and tooling. Code Rules supplies the rule files; it does not run this workflow or require a particular validation or enforcement method.
See [product scope](/start-here/overview/#what-code-rules-does-not-do).

## Project instructions

To use this workflow, add the following section to the project's existing `AGENTS.md`, `CLAUDE.md`, or the instruction file your agent reads, or use it as a starting point for a direct agent prompt. `code-rules project init` also writes it into the project's `.code-rules/README.md`.

```markdown
## Engineering rules

Before planning, implementing, reviewing, testing, or debugging a change:

1. Read `.code-rules/generated/RULES.md` and follow its instructions to
   select relevant groups and read their rules in full, including linked
   files and additional index pages.
2. Follow the applicable rules and their exceptions while doing the work.
3. Before finishing, check your work against those rules and run the
   relevant validation. Briefly report what you verified and any gaps.

If required rule files are unavailable or give conflicting instructions,
report the issue rather than silently skipping them or choosing a policy.
```

`.code-rules/generated/RULES.md` then tells the agent how to select groups, read rules, and report findings, so these instructions stay short.
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
For each finding, cite the resolved rule ID and its version when the group page shows one, the applicable condition, observed evidence, and practical consequence.

Impact describes the consequence a rule helps prevent; it does not determine applicability, override exceptions, or set finding severity.
Read **Why it matters** for context and assess the actual consequence of each finding.

Separate confirmed failures from hypotheses that need verification.
`code-rules project check` establishes file consistency, not application compliance.

## Handle gaps and conflicts

If a relevant group is missing, report the missing coverage.
If resolved rules conflict, identify both IDs and ask the project owner to resolve the intended policy.
Use [Resolve conflicting rules](/guides/conflicting-guidance/) for the review criteria and explicit resolution options.
Keep the recorded ruleset during ordinary work. Adopting upstream changes is a separate step, `code-rules project update`, and major changes need the project owner's consent; don't run it as part of another task.
When asked to keep an imported rule at its current version, read the rule's version from `.code-rules/vendor/<source-name>/_source.json` and add a pin under that source's `pins`, with the version and the reason you were given, such as `practices/testing/verify-retry-limits: {version: "1.3.0", reason: "…"}`. Then run `code-rules project sync`.

## Write or review rules themselves

When the task changes a rule, use the [authoring rubric and template](/reference/rule-authoring/).
Follow those documents directly. The [Code Rules authoring skill](/guides/write-rules/#write-with-the-skill) is planned and is not available yet.

Read the target library's conventions and evaluate each rule against every rubric criterion.
In a library that has published its first library release, add or update the rule's change note with `code-rules library change` in the same change, and choose `major` whenever work that complied with the previous version could fail the new one. See [Record changes to a library rule](/reference/rule-authoring/#record-changes-to-a-library-rule).
Report unmet criteria with the relevant passage and a concrete revision.
Separate unclear wording from unresolved engineering policy; ask the author to resolve the latter.
