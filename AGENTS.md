# Code Rules agent guide

Use this file to understand the product we are building and the decisions it should guide.
The [README](README.md) owns repository setup and validation commands.
The [project status](docs/src/content/docs/status.md) distinguishes working capabilities from proposed interfaces.

## Mission

Help teams turn engineering best practices into shared rules that agents can follow and reviewers can check.

Teams should be able to turn a best practice into a rule, share it across projects, and expect agents to apply it when relevant.
Success means fewer repeated corrections, clearer engineering decisions, and reviews grounded in the team's declared expectations.
Generating a document is a means to that outcome.

## Vision

Code Rules is the shared engineering-standards layer for software factories.
A software factory uses agents and repeatable workflows to carry software from an idea through implementation, validation, and delivery.
Code Rules supplies the standards those workflows apply.

A useful best practice can begin in one project, become a shared organization rule, and reach other projects through deliberate imports.
Projects can combine several canonical libraries while retaining local requirements and explicit exceptions.
Implementing and reviewing agents work from the same resolved rules, with clear applicability and traceable sources.

The authoring experience is part of the product.
A shared rubric, Markdown template, and authoring skill help people and agents create instructions that are clear, scoped, and verifiable.
Correct examples and counterexamples explain both the intended behavior and plausible mistakes.
Reviewing agents use the same rubric to improve the rules themselves.

Code Rules belongs to Fabrica's collection of tools for building software factories.
Code Rules should work across agent tools and workflows without requiring the Fabrica app or a hosted service.

## Ideal customer profile

Our initial target is a technical founder, engineering lead, or platform engineer using coding agents to ship and maintain software.
The strongest fit is a team with several repositories that share technologies and engineering practices but also have project-specific requirements.

These teams already have opinions about quality.
Their standards are scattered across documents, prompts, code reviews, and individual experience.
They repeatedly correct agents for the same mistakes and want consistent implementation and review without restating every expectation in every task.

The human maintainer decides which policies to adopt and owns exceptions.
Agents are direct consumers of the product: they need to find relevant rules, interpret their scope, and cite evidence when reviewing work.
Serve both audiences with readable files and explicit conventions.

A single-project user should still benefit from authoring and applying local rules before adopting a shared library.
Treat this customer profile as an initial product hypothesis, not evidence of validated demand or a commitment to a pricing model.

## Core concepts

### Rule

A rule expresses one independently adoptable engineering expectation in a Markdown file.
It states what to do, when it applies, and what evidence would demonstrate compliance.
Rules can govern code, tests, plans, documentation, and other engineering work.

A rule retains its identity and provenance when imported or included in an aggregate.
See [Rule](docs/src/content/docs/concepts/rule.md) for the concept and [the rubric and template](docs/src/content/docs/reference/rule-authoring.md) for the authoring standard.

### Group

A group collects related rules and explains when an agent should read them.
Technology groups live under `techs/` and cover named languages, frameworks, tools, platforms, or protocols.
Practice groups live under `practices/` and cover concerns such as testing, observability, error handling, and architecture.

Groups guide selection; individual rules determine applicability.
A testing rule may matter even when a change touches no test files.
See [Group](docs/src/content/docs/concepts/groups.md).

### Library

A library is a versioned collection of rule groups published in a Git repository.
Libraries own their engineering opinions and can be public or private.
An organization can publish shared defaults in `<organization>/.code-rules`; a project explicitly chooses which libraries and groups to import.

Projects may import multiple sources, select a commit or tag for each, and add or override rules locally.
Source-qualified IDs distinguish rules from different libraries; they do not resolve contradictory instructions.
See [Library](docs/src/content/docs/concepts/libraries.md) and [Configuration](docs/src/content/docs/reference/configuration.md).

## Product principles

- **Make the relevant guidance available before agents act.** The generated index helps agents select groups from the task, behavior, and technologies involved.
- **Make shared standards adaptable.** Projects own their adopted versions, local rules, exclusions, and replacements; source order does not silently establish policy precedence.
- **Make changes deliberate and traceable.** Imports record resolved revisions and preserve attribution and licensing information; generated files combine the project's active rules by group.
- **Keep deterministic work in the tool.** The CLI resolves imports, validates structure, and generates consistent files; agents interpret applicability, assess implementations, and investigate conflicting guidance.
- **Distinguish evidence from guarantees.** File consistency does not establish application compliance, and an agent review with no findings does not prove the absence of violations.
- **Keep the files useful on their own.** Markdown, Git, and committed aggregates let people inspect changes and agents work offline without a proprietary runtime.

## Ownership and boundaries

This public repository owns the Code Rules conventions, importer, authoring tools, and documentation.
Independently owned libraries supply the engineering policies.
Keep Fabrica's private rule corpus separate; public examples must be original or authorized for redistribution.

Projects retain their product vision, domain knowledge, and architecture context.
The surrounding software factory owns agent execution, review orchestration, approvals, and merge gates.
Code Rules provides rule files and review instructions those systems can use.

## Working in this repository

The docs are a working design preview; the CLI and installable authoring skill have not shipped.
Describe proposed behavior honestly, and consult [project status](docs/src/content/docs/status.md) before claiming availability.

When a product decision changes, update the owning concept, guide, or reference and its examples together.
Keep configuration details and command contracts in those documents rather than duplicating them here.
After docs changes, run the README's validation command.
When layout or interaction changes, inspect the rendered pages.

Keep this file focused on enduring product context and decisions useful to almost every agent session.
Revise or remove stale guidance instead of accumulating a history of decisions.
