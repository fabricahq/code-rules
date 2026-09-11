---
title: "Rule rubric and template"
description: "The shared standard for writing and reviewing clear, scoped, verifiable rules."
---

Start with the Markdown template, then check the rule against the authoring rubric.
Authors, reviewing agents, and the planned Code Rules authoring skill use the same rubric.


## Markdown template

Start with this structure and replace the placeholders.
Add examples where they clarify the obligation, and omit sections that add no useful guidance.
The [metadata reference](/reference/files/#rule-metadata) defines the frontmatter fields and allowed impact values.

````md
---
title: <Short action-oriented title>
impact: <Allowed impact value>
impactDescription: <Consequence this rule addresses>
tags: <Relevant topics, separated by commas>
whenToApply: <When an agent should apply this rule>
---

## <Short action-oriented title>

<State one concrete obligation.>

### Applicability

<When does this rule apply?>
<What exceptions or boundaries matter?>

### Rationale

<Explain the failure or tradeoff this rule addresses.>

### Incorrect example

<Show a plausible mistake that violates this rule.>
<Explain why it fails in the stated situation.>

### Correct example

<Show the preferred approach in the same situation.>
<Explain how it satisfies the rule and clarify illustrative choices.>

### Verification

<Describe the evidence that demonstrates compliance.>
````

Preserve source attribution and any required notices when adapting existing material.
Use the [complete example](/guides/write-rules/#example-rule) to see the template filled in.

## Authoring rubric

The rubric defines what makes a useful rule; filling every template heading does not establish compliance.
Evaluate each rule against every criterion.
Revise any unmet criterion, or explain why it does not apply.

| Criterion | What to look for |
| --- | --- |
| One obligation | The rule states one independently reviewable expectation. Split obligations that a project might adopt or replace separately. |
| Clear action | The rule tells the agent what to do. Replace vague instructions such as "use good error handling" with observable behavior. |
| Explicit scope | State the conditions that trigger the rule and any exceptions. An example's language does not implicitly limit a practice rule to that language. |
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
For example, "Verification: review the code" does not explain what the reviewer should check.

Keep rubric findings separate from disagreements about the underlying engineering policy.
When the intended policy is unclear, ask the author instead of inventing an obligation.
