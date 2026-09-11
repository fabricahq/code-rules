# Writing useful when-to-read guidance

Research date: 2026-09-11.

The question is how to help agents choose relevant groups and rules before implementation and during review. We checked three primary sources about agent skill discovery, the closest documented analogue to selecting a rule from an index.

## What the sources support

- The Agent Skills specification separates discovery metadata from instructions loaded after selection. Descriptions should identify the task and include specific terms that help an agent recognize relevant work. This supports putting selection cues in `whenToRead`, where the agent sees them before opening the full rule. [Agent Skills specification](https://agentskills.io/specification#description-field)
- Anthropic recommends specific triggers and contexts, concrete descriptions, and enough detail to distinguish relevant tasks. It also recommends concise instructions and improving them against representative evaluations. Brevity means removing unnecessary explanation, not removing useful selection details. [Skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions)
- Anthropic's evaluation guidance explicitly covers expected activation, expected non-activation, ambiguous cases, and coexistence with other skills. It recommends testing the models actually used. This supports checking both missed rules and unnecessary selections. [Skills for enterprise](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/enterprise#evaluating-skills-before-deployment)

These are specification and vendor recommendations, not controlled evidence that this exact wording improves Code Rules compliance. The sources do not establish an optimal word count. Applying their discovery advice to engineering rules is our design inference; selection and correct application still need testing in our own workflow.

## Guidance for Code Rules authors

1. Name the work and its relevant situation. Prefer recognizable actions and objects, such as coordinating parsing, validation, and persistence, over broad labels such as “complex code.”
2. Include prospective work. A cue should work before the function or defect exists. Name planning, writing, changing, reviewing, or diagnosis only where those phases are relevant.
3. Keep examples that clarify scope. A short “such as” list helps translate an abstract concern into task details. It illustrates the category rather than defining an exhaustive checklist.
4. Add a boundary when it prevents a plausible selection mistake. Keep compliance exceptions and detailed remedies in the body unless they also change whether the rule is worth reading.
5. Distinguish reading from a finding. Selecting a rule means its guidance may matter; it does not prove that code violates it or needs a particular refactor.
6. Match the level of selection. Group cues cover the work represented by their rules; individual cues identify the specific concern. Practice cues should follow intended behavior, not only changed file extensions.

For example:

> Before planning, writing, changing, or reviewing a function that coordinates multiple steps, such as parsing input, validating it, calling another operation, or constructing a result.

The full rule can then explain when extraction helps and why a short, cohesive function can already comply.

## Lightweight selection checks

Try an intended implementation task before code exists, a review task, an unrelated task, and a nearby boundary case. For this example: designing a parse-validate-save operation and reviewing one should select the rule; changing only a document title should not. Reviewing a short, cohesive multistep function should select it without inventing a violation. For testing guidance, include a behavior change that touches no test file.

These are proposed checks, not completed agent evaluations. Record the group/rule selected, whether its full text was read, and whether the implementation or finding actually followed it. Revise cues against observed misses or over-selection rather than making every description longer.
