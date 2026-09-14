---
title: "Project status"
description: "What this documentation describes and what remains to be built."
---

**Code Rules is in early implementation.**
The documentation includes implemented offline generation and proposed workflows for the first release.
The CLI and installable authoring skill have not shipped.

## What works in this checkout

The offline `buildRules` API accepts configuration, library snapshots, local rules, and a tool version.
It returns generated text files without fetching libraries or writing to disk:

- Resolve multiple libraries, exclusions, complete local replacements, local additions, and local-only groups.
- Select explicit groups or `"*"`, `"practices/*"`, and `"techs/*"` from snapshots complete for that scope.
- Generate `RULES.md`, adaptive group pages, individual effective rules, and deterministic provenance.
- Include complete rules in small groups and paginate larger indexes without truncating rule bodies.
- Generate a README per imported library and copy declared license and notice text to standard paths under `generated/libraries/`.
- Validate rule metadata, group metadata, snapshot identity, SPDX declarations, and source-scoped exceptions.

The shared authoring rubric and template are also available.
Try the [development example](/guides/use-rules/#try-the-working-builder) or read the [offline API contract](/reference/files/#offline-builder-api).
Automated tests and an interactive runbook exercise both rule-delivery formats, licenses, and group selectors.

[Imports PR #2](https://github.com/fabricahq/code-rules/pull/2) covers network fetching and Git repository URL handling.
They are not part of this checkout's offline builder.
Workspace installation, digest verification, generated-file consistency checks, CLI commands, and the installable authoring skill remain planned.

## What is settled

- Separate public tooling from independently owned rule libraries.
- Organize groups under `techs/` and `practices/`.
- Accept commit or tag refs, record resolved commits, and scope project exceptions to each source.
- Commit effective rules under `code-rules/generated/`.
- Give writing and reviewing agents the same rule-loading guidance.
- Use one authoring rubric for the template, authoring skill, and rule reviews.
- Import compatible libraries; adapt other source material explicitly before importing it.
- Give library authors explicit commands to initialize, add groups, add rules, and validate their files.

## What is proposed

CLI commands and the installable authoring skill workflow remain proposed.
The builder implements the documented configuration and rule metadata, but these unreleased formats may still change.
The `_source.json` file format and workspace installation contract remain proposed.
The proposed [`code-rules update`](/reference/cli/#update-the-tool) command upgrades the CLI to its latest stable release.
The proposed [`code-rules conflicts --prompt`](/reference/cli/#conflict-review-prompt) command prepares an agent review of conflicting guidance.
Examples illustrate how those interfaces fit together.
They are not installation instructions for a published package.

## First release

The first release will import multiple named libraries directly into a project. Each source has its own commit-or-tag ref, group selection, and exceptions.
The initial proof uses original example rules, local exceptions, and generated files that a person can inspect.
Reliable imports and a private-project pilot follow that proof.

The final implementation phase standardizes library authoring with `library init`, `library add group`, `library add rule`, and `library check`.
It includes the authoring skill's adaptation workflow and a walkthrough from an empty library to importable rules with retained terms.
See [Create a rule library](/guides/create-library/) and [Adapt a third-party rule](/guides/adapt-rules/) for the proposed experience.

Organization libraries that inherit from and republish other libraries, and assisted group selection, are later work.

## Outside the product scope

Code Rules manages the rules associated with a codebase. Application compliance checks, enforcement, review orchestration, and merge gates belong to project-specific agent instructions or separate tooling. These are integration choices, not features waiting for implementation.
The supplied agent workflow is a suggestion; projects choose how to apply and validate their rules.

Follow development in the [public repository](https://github.com/fabricahq/code-rules).
