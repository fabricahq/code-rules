---
title: "Rule rubric and template"
description: "Recommended guidance for writing and reviewing clear, useful rules."
---

A useful rule tells an agent what to do, when it applies, and how to check its work. Use the template and rubric below to make that guidance clear.

These are **writing recommendations, not format requirements**. Keep the parts that help explain your rule. Your project or library can set its own conventions.
Code Rules validates [metadata and file structure](/reference/rule-library-format/), not writing quality. For the creation commands, follow [Write a rule](/guides/write-rules/).

## What goes in a rule

- **Metadata:** YAML fields at the top give the rule a title, a reading cue, and an impact level.
- **Body:** Markdown states the instruction. Add scope, exceptions, rationale, examples, and checks when they help someone apply it.

## When to read, implement, and validate

Before writing, answer three questions:

| Question | Where to answer it |
| --- | --- |
| When should an agent read this? | Describe recognizable work in `whenToRead`, including planning before code exists. |
| What should the agent do? | State the instruction and its exceptions in the body. |
| How can someone check the result? | Name observable evidence or a specific check. |

Reading a relevant rule does not mean the code violates it. The full instruction, conditions, and exceptions determine whether a change is needed.

## Markdown template

The CLI creates a draft from this structure. Replace the prompts with your guidance and remove sections that add no useful information.
Finish the draft and remove its `<!-- code-rules:draft -->` marker before building or checking it.

````md
---
title: <Short action-oriented title>
whenToRead: 'Before [relevant activities] involving [specific behavior or artifact], such as [representative cases, if helpful].'
impact: <Level matching the credible consequence within this rule's scope>
impactDescription: <Specific consequence the rule helps prevent, supporting the impact level>
---

## <Short action-oriented title>

<State one concrete obligation.>
<Name version, runtime, or surrounding-contract prerequisites when they affect the advice.>

### Implementation

<Describe decisions or steps that help an agent write compliant code.>
<Include relevant exceptions and boundaries.>

### Rationale

<Explain the failure or tradeoff this rule addresses.>
<Explain the mechanism connecting the action to that consequence; cite supporting contracts where needed.>

### Examples

<Repeat the application block for each distinct known use of the rule.>
<Cover differences in behavior, constraints, or implementation choices; combine cases when the same pair explains them without losing useful guidance.>

#### Application: <Name the situation>

<Explain when this application occurs and which conditions matter.>

**Incorrect (counterexample):**

<Show a plausible mistake that violates the rule in this situation.>
<Explain what goes wrong.>

**Correct:**

<Show the preferred approach in the same situation, keeping unrelated details consistent.>
<Explain how it satisfies the rule and clarify illustrative choices.>
<When overapplication is likely, also show a similar-looking valid case and explain why it needs no change.>

### Validation

<Describe observable evidence or checks that establish compliance.>
<Identify plausible situations that are insufficient evidence of a violation.>
<Name any surrounding code or contracts the reviewer must inspect before deciding.>
````


For accepted fields and values, see [Rule metadata](/reference/rule-library-format/#rule-metadata). For a filled-in example, see [Write a rule](/guides/write-rules/#example-rule).

## Write whenToRead guidance that helps selection

Describe work an agent can recognize **before reading the rule**. Avoid cues that require discovering the problem first.

**Too dependent on knowing the problem:**

> When orchestration is obscured by parsing.

**Names the work:**

> Before planning, writing, or reviewing a function that coordinates multiple steps, such as parsing input, validating it, or calling another operation.

A group's cue describes an area of work; a rule's cue identifies more specific situations. Practice rules can apply even when no related file changes.
For example, testing guidance matters when planning a behavior change, before anyone edits a test.

Try the cue against a relevant task, the same task phrased differently, and a nearby task outside its scope. Revise missed or unnecessary selections.
Keep detailed obligations and exceptions in the body.

These recommendations adapt the [Agent Skills description guidance](https://agentskills.io/specification#description-field) and Anthropic's [authoring](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions) and [evaluation](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/enterprise#evaluating-skills-before-deployment) guidance to rule discovery. They are authoring advice, not a guarantee that agents select every relevant rule.

## Describe impact through consequences

Choose `impact` based on a credible consequence within the rule's scope. Use `impactDescription` to explain that consequence in a sentence.

| Impact | Consequence the rule addresses |
| --- | --- |
| CRITICAL | Severe harm, such as irreversible data loss or a major security breach. |
| HIGH | Substantial correctness, reliability, or maintainability problems. |
| MEDIUM | Meaningful but bounded defects or recurring development friction. |
| LOW | Local clarity or consistency improvements with limited consequences. |

`MEDIUM-HIGH` and `LOW-MEDIUM` sit between adjacent levels.

“Important for quality” is too vague. “Hidden failures prevent callers from recovering” explains why an error-handling rule matters.

Agents should follow every applicable rule regardless of impact. Impact does not resolve conflicts or set review severity; assess each finding from its actual consequences.

## Write examples that explain the difference

Show a plausible mistake beside the preferred approach. Keep unrelated details consistent and explain the decisive difference.

Add examples when different contexts change how to follow the rule or which exception matters. One pair is enough when it explains the rule fully.
If readers might overapply the rule, show a similar-looking case that needs no change.

Label illustrative choices. A practice rule demonstrated in Go does not automatically apply only to Go.

## Explain how to check the rule

Name evidence a reviewer can inspect. “Send an oversized upload and check that the error names the size limit” is more useful than “review the code.”

State prerequisites that affect the advice, such as a runtime version or caller contract. Identify surrounding code a reviewer must inspect before reporting a violation.
Distinguish a permitted exception from a case where the evidence is insufficient.

Not every rule needs an automated test. Rules can govern plans, code, documentation, or other work; choose a check that matches the instruction.

## Supporting material

Keep obligations and exceptions in the rule itself. Put optional diagrams, sample data, and longer explanations in [supporting assets](/reference/rule-library-format/#supporting-assets).
Rules must remain independently selectable, so don't depend on another rule document or its private assets.

When adapting external material, preserve its source credits and required notices. Follow [Import rules from another source](/guides/select-rules/#from-another-source).

## Authoring rubric

Use the questions that fit your rule. A short rule can be complete without every template section.

| Criterion | Question |
| --- | --- |
| One obligation | Can a project adopt or replace this expectation independently? |
| Clear action | Does the rule say what to do, rather than only “use good practices”? |
| Discoverable relevance | Can an agent recognize relevant work from `whenToRead`? |
| Explicit scope | Are the conditions and exceptions clear? |
| Supported impact | Does the description justify the impact level? |
| Useful rationale | Does the explanation connect the instruction to a real consequence? |
| Concrete examples | Do examples show the important difference and cover distinct applications? |
| Verifiable compliance | Does the rule name evidence someone can check? |
| Honest claims | Are technical claims supported and team preferences identified as preferences? |
| Complete meaning | Can someone apply the rule without guessing at missing context? |

## Review a proposed rule

When you find unclear or incomplete guidance:

1. Identify the passage.
2. Explain what is ambiguous, missing, or unsupported.
3. Suggest a concrete revision.

Keep writing improvements separate from policy disagreements. The project or library owner decides which engineering practices to require.
