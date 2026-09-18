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
