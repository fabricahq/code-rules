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
whenToRead: "Before [relevant activities] involving [specific behavior or artifact], such as [representative cases, if helpful]."
impact: <Level matching the credible consequence within this rule's scope>
impactDescription: <Specific consequence the rule helps prevent, supporting the impact level>
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

Reading a rule does not imply a violation; its full body defines the obligation and exceptions.

Consider implementation and validation separately, but omit extra sections when the obligation or examples already answer their questions.
Delete unused template prompts; authors need not explain omitted sections.
Use specific evidence or a check command instead of “verify that this rule is followed.”
Make examples consistent with applicable technology and project conventions, and use a suitable language for code snippets.
A practice rule's example language does not limit its scope.

Preserve source attribution and any required notices when adapting existing material.
Use the [complete example](/guides/write-rules/#example-rule) to see the template filled in.

## Write whenToRead guidance that helps selection

Write a cue an agent can match against its task before opening the full rule.
Use the same guidance for group metadata, with broader cues covering the group's concerns.

- **Name the work and its object.** Identify the behavior, artifact, interface, or technology involved. “When coding” does not distinguish relevant work.
- **Include prospective work.** Use activities such as planning or writing where relevant. A cue that requires spotting a violation first will miss implementation tasks.
- **Add recognizable examples when they clarify scope.** Terms such as parsing input, validation, external calls, and result construction explain what “multiple steps” means. Use “such as” to keep examples illustrative.
- **Cover distinct situations that need the guidance.** Include review or diagnosis when relevant. For practices, describe behavior as well as files: testing guidance can apply without test-file edits.
- **State a boundary when it prevents a likely selection mistake.** Include a technology or context restriction if the rule depends on it. Keep detailed obligations and exceptions in the body.
- **Keep useful detail; remove repetition.** Prefer a focused sentence or two. Add words that help distinguish relevant tasks, without imposing a fixed word limit or listing every synonym.

In the template, replace the activity and scope placeholders and omit the “such as” clause when examples add nothing.
The goal is a recognizable trigger, not the shortest possible description.

For a function-design rule, use:

> Before planning, writing, changing, or reviewing a function that coordinates multiple steps, such as parsing input, validating it, calling another operation, or constructing a result.

“When orchestration is obscured by parsing” is too late: an agent must recognize the defect before deciding to read the rule.
The fuller cue also covers a well-structured function. Its full rule determines whether any change is warranted.

Before accepting a cue, check it against representative tasks:

1. A planned change where no code exists yet should trigger reading.
2. A review or diagnosis within scope should trigger reading, including when the relevant test or other artifact is missing.
3. A relevant task phrased differently should still match; examples should not become an exhaustive checklist.
4. A nearby task outside the rule's scope should be distinguishable. For the function-design example, changing only a color token would not trigger it.

Check selection separately from compliance: a cohesive multi-step function should still lead to reading, without automatically producing a finding.
When evaluating with an agent, record both missed relevant rules and unnecessary selections, then revise the cue based on those cases.
These checks are an authoring practice, not a required metadata field or proof that every agent will select correctly.

We adapted these recommendations from the [Agent Skills specification](https://agentskills.io/specification#description-field) and Anthropic's [authoring](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions) and [evaluation](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/enterprise#evaluating-skills-before-deployment) guidance.
Those sources address skill discovery; applying them to rule selection is a design inference, not a measured improvement in Code Rules compliance.

## Describe impact through consequences

`impact` describes the significance of the consequence a rule helps prevent.
Choose the level based on credible consequences within the rule's scope, rather than an imaginable worst case.
Use `impactDescription` to name that consequence in a focused sentence. Generated pages display it as **Why it matters**.

| Impact | Consequence the rule addresses |
| --- | --- |
| CRITICAL | Severe harm, such as irreversible data loss or a major security breach. |
| HIGH | Substantial correctness, reliability, or maintainability problems. |
| MEDIUM | Meaningful but bounded defects or recurring development friction. |
| LOW | Local clarity or consistency improvements with limited consequences. |

`MEDIUM-HIGH` and `LOW-MEDIUM` sit between the adjacent anchors.
The description should support the chosen level; “important for quality” does not explain a consequence.

For Express Operations as Meaningful Steps, a useful description is:

> Mixing orchestration with low-level details can hide important decisions and make behavior harder to verify or change.

Agents use `whenToRead` to select rules and the full body to understand obligations and exceptions.
Read and follow every applicable rule, regardless of impact.
Impact does not determine applicability, override exceptions, resolve conflicting rules, or set a review finding's severity.
Assess each finding from concrete evidence and the consequences of that specific violation.
A high-impact design rule does not require extracting every multi-step function or make every readability finding high severity.

## Authoring rubric

The rubric defines what makes a useful rule; filling every template heading does not establish compliance.
Evaluate each rule against every criterion.
Revise missing or unclear guidance where the criterion applies; additional headings are useful only when they add information.

| Criterion | What to look for |
| --- | --- |
| One obligation | The rule states one independently reviewable expectation. Split obligations that a project might adopt or replace separately. |
| Clear action | The rule tells the agent what to do. Replace vague instructions such as "use good error handling" with observable behavior. |
| Discoverable relevance | Use the [selection guidance](#write-whentoread-guidance-that-helps-selection) to name recognizable work, include useful examples, and check both relevant and out-of-scope tasks. |
| Explicit scope | State the obligation's conditions and exceptions in the full body. An example's language does not implicitly limit a practice rule to that language. |
| Supported impact | Use the [impact guidance](#describe-impact-through-consequences) to choose a level and name a credible consequence. Keep finding severity dependent on evidence. |
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
