---
title: "Rule rubric and template"
description: "The shared standard for writing and reviewing clear, scoped, verifiable rules."
---

Start with the Markdown template, then check the rule against the authoring rubric.
Authors, reviewing agents, and the planned Code Rules authoring skill use the same rubric.
One shared template covers technologies, practices, and project-specific rules. Metadata has a defined format; authors adapt the body to the rule.


## Markdown template

Start with this structure and replace the placeholders.
Add examples where they clarify the obligation, and omit sections that add no useful guidance.
The [metadata reference](/reference/files/#rule-metadata) defines the frontmatter fields and allowed impact values.

````md
---
title: <Short action-oriented title>
whenToRead: "When planning, implementing, reviewing, or diagnosing [specific work]."
impact: <Allowed impact value>
impactDescription: <Consequence this rule addresses>
tags: <Relevant topics, separated by commas>
---

## <Short action-oriented title>

<State one concrete obligation.>

### Implementation

<Describe decisions or steps that help an agent write compliant code.>
<Include relevant exceptions and boundaries.>

### Rationale

<Explain the failure or tradeoff this rule addresses.>

### Incorrect example

<Show a plausible mistake that violates this rule.>
<Explain why it fails in the stated situation.>

### Correct example

<Show the preferred approach in the same situation.>
<Explain how it satisfies the rule and clarify illustrative choices.>

### Validation

<Describe observable evidence or checks that establish compliance.>
<Identify plausible situations that are insufficient evidence of a violation.>
````

## When to read, implement, and validate

Every author considers three questions separately:

| Question | Where to answer |
| --- | --- |
| Is this relevant to my work? | Required `whenToRead` metadata describes the activity that should trigger reading, including before code exists. |
| How should I implement it? | The body supplies useful decisions, procedures, or examples. Add an Implementation section when it contributes information. |
| How can I check compliance? | The body supplies observable evidence, checks, and boundaries for findings. Add a Validation section when useful. |

For a function-design rule, `whenToRead` might say: “When planning, writing, changing, or reviewing a function that coordinates several steps.”
A trigger such as “when orchestration is obscured by parsing” requires recognizing the mistake first and misses prospective implementation.
Check each trigger against both a planned change and a review or diagnosis of existing code.
Reading a rule does not imply a violation; its full body defines the obligation and exceptions.

Consider implementation and validation separately, but omit extra sections when the obligation or examples already answer their questions.
Delete unused template prompts; authors need not explain omitted sections.
Use specific evidence or a check command instead of “verify that this rule is followed.”
Make examples consistent with applicable technology and project conventions, and use a suitable language for code snippets.
A practice rule's example language does not limit its scope.

Preserve source attribution and any required notices when adapting existing material.
Use the [complete example](/guides/write-rules/#example-rule) to see the template filled in.

## Authoring rubric

The rubric defines what makes a useful rule; filling every template heading does not establish compliance.
Evaluate each rule against every criterion.
Revise missing or unclear guidance where the criterion applies; additional headings are useful only when they add information.

| Criterion | What to look for |
| --- | --- |
| One obligation | The rule states one independently reviewable expectation. Split obligations that a project might adopt or replace separately. |
| Clear action | The rule tells the agent what to do. Replace vague instructions such as "use good error handling" with observable behavior. |
| Discoverable relevance | Describe the activity in `whenToRead` so agents can select the rule before writing code or identifying a violation. |
| Explicit scope | State the obligation's conditions and exceptions in the full body. An example's language does not implicitly limit a practice rule to that language. |
| Useful rationale | Explain the failure or tradeoff the obligation addresses. Keep context separate from the obligation so it does not introduce hidden requirements. |
| Concrete examples | Show the expected behavior when prose alone leaves room for interpretation. Label illustrative choices so they do not become accidental requirements. |
| Counterexamples | Show a plausible violation and explain what goes wrong. Pair it with a correct example of the same situation, keeping unrelated details consistent. Label both clearly. |
| Verifiable compliance | Explain what code, behavior, test, or other evidence would demonstrate compliance. Match the check to the obligation; not every rule needs an automated test. |
| Honest claims | Support factual claims and retain source attribution where needed. Distinguish an organization's preference from a universal requirement. |
| Complete meaning | Make the rule understandable with its stated context. Keep the obligation, applicability, and verification visible instead of hiding them behind unexplained references. |

A rule can govern plans, code, tests, documentation, or another engineering artifact.
The rubric checks the quality of the instruction; the library owner chooses the engineering policy.

### Make counterexamples useful

Show a realistic mistake, not an obviously broken example that teaches little.
Explain the failure and place the corrected approach nearby.
Keep the contrast focused on the rule so an agent can see which change matters.
For a practice rule, label any language or framework choice as illustrative unless the rule depends on it.

## Review a proposed rule

For each unmet criterion, identify the passage, explain the ambiguity or missing evidence, and suggest a concrete revision.
A heading's presence alone does not satisfy a criterion.
For example, "Validation: review the code" does not explain what the reviewer should check.

Keep rubric findings separate from disagreements about the underlying engineering policy.
When the intended policy is unclear, ask the author instead of inventing an obligation.
