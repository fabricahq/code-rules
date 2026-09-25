# Code Rules agent guide

Use this file to understand the product we are building and the decisions it should guide.
[CONTRIBUTING.md](CONTRIBUTING.md) owns repository setup and validation commands.

## Product

### Motivation

Agents generate code quickly, but that code often conflicts with the team's engineering best practices. Finding those mismatches in review is expensive and frustrating: reviewers must explain expectations, request changes, and check the revised implementation.
Repeated corrections consume the time and attention that agents were supposed to save.

Giving agents the right guidance before they write code helps teams avoid that rework. Code that aligns with the team's practices earlier in the process makes software development easier, less expensive, and more enjoyable.

### Mission

Help human organizations generate code with AI that aligns with their best practices.

### Vision

When an individual or team starts a software project, they select the technologies and engineering practices they want to apply.

From then on:
During implementation, all agents follow the organization's best practices, as declared in the project's rules.
During validation, code is checked explicitly for compliance with those rules.

Organizations quickly gain confidence that agents will apply their best practices throughout implementation and review.

When code does fall short, teams capture the lesson in a new rule for that project or share it across projects. Insights and best practices spread across projects and teams, so a lesson learned in one place improves future work elsewhere.

### Ideal customer profiles

## Humans

Our ideal customer is an individual or software team that wants to produce correct, maintainable, and secure code using AI. They are dissatisfied with the code their agents produce or find that enforcing their best practices is painful, ineffective, or both.

The strongest fit is an individual or team with several repositories that share technologies and engineering practices but also have project-specific requirements. At the same time, a single-project user should still benefit from authoring and applying local rules before adopting a shared library.

Their best practices may be scattered across documents, prompts, code reviews, and individual experience. They repeatedly correct agents for the same mistakes and want consistent implementation and review without restating every expectation in every task.

Humans will decide which policies to adopt, and when to create an exception. They benefit from readable files and explicit conventions.

## Agents

Agents need to find relevant rules, interpret their scope, and cite evidence when reviewing work. Agents should get clear, unambiguous guidance on how to write the code or adopt a practice. When agents are confused, they should speak up and suggest an update to the rules.

### Core concepts

#### Rule

A rule expresses one independently adoptable engineering expectation in a Markdown file. It states what to do, when it applies, and what evidence would demonstrate compliance. Rules can govern code, tests, plans, documentation, and other engineering work.

A rule retains its identity and provenance when imported or rendered as a resolved definition. See [Rule](docs/src/content/docs/concepts/rule.md) for the concept and [the rubric and template](docs/src/content/docs/reference/rule-authoring.md) for the authoring standard.

#### Group

A group collects related rules and explains when an agent should read them. There are two types of groups:

1. Technology groups live under `techs/` and cover named languages, frameworks, tools, platforms, or protocols.
2. Practice groups live under `practices/` and cover concerns such as testing, observability, error handling, and architecture.

Groups guide selection; individual rules determine applicability. A testing rule may matter even when a change touches no test files. See [Group](docs/src/content/docs/concepts/groups.md).

#### Library

A library is a versioned collection of groups published in a Git repository. Libraries own their engineering opinions and can be public or private. An organization can publish shared defaults in `<organization>/.code-rules`; a project explicitly chooses which libraries and groups to import.

Projects may import multiple sources, select a commit or tag for each, and add or override rules locally.
Source-qualified IDs distinguish rules from different libraries; they do not resolve contradictory instructions.
See [Library](docs/src/content/docs/concepts/libraries.md) and [Configuration](docs/src/content/docs/reference/configuration.md).

### Principles

1. **Prevent mistakes before they become rework.**
   Give agents relevant guidance before they write code.
   Aim to get the implementation right the first time, when correcting a mistake is cheapest.

2. **Turn lessons into lasting improvements.**
   When code falls short, use the observation to improve a rule, its enforcement, or how agents discover it.
   Capture lessons locally and share them across projects when they apply more broadly.

3. **Reuse shared practices; customize deliberately.**
   Teams should adopt shared best practices without redefining them for every project.
   Projects can add requirements and make explicit exceptions when their needs differ.

4. **Provide consistent expectations across workflows.**
   Make the same resolved rules available for implementation and review.
   Giving an agent a rule does not guarantee that it will follow it.

5. **Keep rule management independent of enforcement.**
   Projects choose their agent prompts, validation tools, and enforcement mechanisms.
   Deliver clear rule files that those workflows can consume.

### Ownership and boundaries

This public repository owns the Code Rules conventions, importer, authoring tools, and documentation.
Independently owned libraries supply the engineering policies.
Keep Fabrica's private rule corpus separate; public examples must be original or authorized for redistribution.

Projects retain their product vision, domain knowledge, and architecture context.
Code Rules manages which versioned rules a codebase adopts and delivers their resolved definitions.
Projects choose how to apply, validate, and enforce those rules through agent prompts or separate tooling. Code Rules does not prescribe or run that workflow; its agent instructions are a suggested integration.
Rule-input validation and generated-file consistency checks are in scope; checking application compliance is not. See [product scope](docs/src/content/docs/start-here/overview.md#what-code-rules-does-not-do).

## Working in this repository

Before implementing or reviewing code, consult [Fabrica's engineering rules](https://github.com/fabricahq/app/tree/main/_rules) and read the individual rules relevant to the change.
For Go implementation, also read [Go conventions](_engineering/go-conventions.md).
Apply the portable guidance; identify app-specific assumptions and explain any adaptation needed for this CLI.
For comments, use [the local comment rule](_engineering/rules/comment-role-result-and-constraints.md), which supersedes the linked app comment policy.
For website JavaScript and TypeScript, use `@fileoverview` headers; an Astro component's frontmatter overview also describes its rendered result because its export is implicit.

The Go CLI is released. The installable authoring skill remains separate work.
Describe proposed behavior honestly, and verify availability against the implementation before claiming support.

When a product decision changes, update the owning concept, guide, or reference and its examples together.
Keep configuration details and command contracts in those documents rather than duplicating them here.
After docs changes, run the validation commands in [CONTRIBUTING.md](CONTRIBUTING.md#validate-changes).
When layout or interaction changes, inspect the rendered pages.

Keep this file focused on enduring product context and decisions useful to almost every agent session.
Revise or remove stale guidance instead of accumulating a history of decisions.

<!-- release-planner:begin v0.4.3 sha256:3d7c63eae9a8c94d -->
## Releases

This repository publishes releases with [Release Planner](https://github.com/fabricahq/release-planner) v0.4.3. When asked to make a release, draft, revise, or correct release notes, or retry a failed release, print the release procedure and follow it:

```sh
release-planner guide
```

First check that `release-planner version` prints `v0.4.3`. If it doesn't, or `release-planner` isn't installed, install that version:

```sh
curl -fsSL https://raw.githubusercontent.com/fabricahq/release-planner/v0.4.3/install.sh | sh -s -- --version v0.4.3
```

Read `.release-planner/policy.md` first for this repository's release policy. You prepare the release pull request; the maintainer approves the release by merging it. Never tag, publish, or merge.
<!-- release-planner:end -->
