---
title: "Decide what belongs in a rule"
description: "Choose between deterministic and stochastic guidance, then between a rule and a skill."
---

Rules are one way to guide agents. Scripts, linters, tests, and skills guide them too. To decide where a piece of guidance belongs, answer two questions:

1. Should it be **deterministic** or **stochastic**?
2. If it's stochastic, should it be a **rule** or a **skill**?

## Deterministic or stochastic

**Deterministic** guidance is code. It gives the same result every time, whoever runs it. **Stochastic** guidance is instructions that an agent interprets, so its results vary with the agent and from run to run.

Each kind can help agents implement work or validate it:

| | Implement | Validate |
| --- | --- | --- |
| **Deterministic** | Scripts | Lint rules, type checks, tests |
| **Stochastic** | Rules, skills | Rules |

### Prefer deterministic guidance

If you can state exactly what's right in every case, write code. Code means exactly what it says, so humans, agents, and CI all get the same result. It costs an agent no attention to run, and a machine can verify the outcome.

Use stochastic guidance when you can't state exactly what's right, because the cases are open-ended or depend on context. One sentence such as "Charge customers only through the billing service" covers situations you never listed. Spelling out every case in English instead is tedious, and the result is still ambiguous.

When a tool becomes able to check part of a rule, move that part into the check. Keep the rest in the rule: the reason behind the practice, and the cases the tool can't see.

## Rule or skill

Rules and skills are both stochastic: reusable instructions, with supporting resources, that an agent interprets. They differ in two ways:

| | Rule | Skill |
| --- | --- | --- |
| **Scope** | One assertion about the work | Many instructions for one kind of task |
| **Use** | Implement and validate | Implement |

A rule makes one assertion, such as "Change prices only in the plan catalog." Because it's a single assertion, a reviewer can check work against it, so the same rule guides both the agent that writes the code and the agent that reviews it. A project can also adopt, exclude, replace, or [version](/guides/version-rules/) one rule without changing the others.

A skill bundles the instructions for a task, such as adding a pricing plan. It guides an agent through the work, but it makes no single assertion to check the result against. Validate the result against rules instead.

To choose between them:

- **Use a rule for an assertion that should hold in every task it applies to.** "Change prices only in the plan catalog" applies whether an agent is adding a pricing plan, fixing a refund bug, or reviewing a pull request.
- **Use a skill for the steps of one kind of task.** "Add a pricing plan" tells an agent what to do, in order, when it takes on that task.
- **Don't repeat rules in skills.** Agents read the rules that apply to a task through `RULES.md`, so the pricing-plan skill can leave out where prices live. The rule also covers tasks the skill never sees, such as a refund fix.

## Common guidance smells

These signs suggest that guidance is in the wrong place.

#### A rule reads like a specification

If a rule lists exact steps that never vary, or exact conditions a tool could test, move that part into a script or check.

#### A check is full of exceptions

If a lint rule needs constant suppressions, it's trying to encode judgment. Write a rule instead.

#### A skill says "always" or "never"

That's a rule hidden in a skill. Extract it so it applies outside the skill and reviewers can check it.

#### A rule makes several assertions

Split it into rules that can each be followed, checked, and changed on their own. If the assertions are the steps of one kind of task, it's a skill instead, or a script if the steps never vary.

#### A skill holds a single assertion

That's a rule. Turn it into one, so it applies to every task it covers and reviewers can check work against it.

## Example: a billing module

One part of a project usually needs several kinds of guidance:

- **Script:** `bun run billing:webhooks` replays the payment provider's webhooks locally.
- **Checks:** `bun run billing:check` validates the plan catalog, and a lint rule rejects imports of the provider's SDK outside the billing module.
- **Rules:** "Charge customers only through the billing service" explains why and covers what the linter can't see, such as raw HTTP requests to the provider's API. "Run `bun run billing:check` before finishing" makes sure agents use the check.
- **Skill:** "Add a pricing plan" walks an agent through the task and relies on the rules above.

Rules for one part of a project belong in an [area group](/concepts/groups/#areas).

## Where READMEs fit

A README should explain how something works, not tell anyone what to do. That makes it context for all of the guidance above rather than guidance itself. Keep explanation in READMEs, and state expectations as rules:

- **Agents read rules before they plan.** They find them through the generated `RULES.md`. A README in a folder is often read only after an agent opens that folder, when its plan is already set.
- **Rules aren't tied to one folder.** Guidance about one part of a project often spans several folders, or even repositories.
- **Rules are addressable.** Each rule has an ID, so a reviewer can cite it.

Code Rules manages rules. Scripts, checks, skills, and READMEs stay in your project's own tooling.

When a rule is the right choice, [write it](/guides/write-rules/).
