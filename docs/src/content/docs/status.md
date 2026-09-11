---
title: "Project status"
description: "What this documentation describes and what remains to be built."
---

**Code Rules is in early implementation.**
These docs describe the intended first release so teams can review the experience before implementation.
The documentation site, authoring rubric, and template are available.
The offline Builds module can combine in-memory library snapshots and local rules into generated files.
It emits applicability indexes and individual effective definitions, with bounded index parts and required rule-level `whenToRead` metadata.
It has focused automated tests and a runnable example for development.
Library fetching, workspace installation and consistency checks, CLI commands, and the installable authoring skill remain unimplemented.

## What is settled

- Separate public tooling from independently owned rule libraries.
- Organize groups under `techs/` and `practices/`.
- Accept commit or tag refs, record resolved commits, and scope project exceptions to each source.
- Commit effective rules under `code-rules/generated/`.
- Give writing and reviewing agents the same rule-loading guidance.
- Use one authoring rubric for the template, authoring skill, and rule reviews.

## What is proposed

The CLI commands, authoring skill workflow, configuration fields, metadata files, and format versions are proposed interfaces.
The proposed [`code-rules update`](/reference/cli/#update-the-tool) command upgrades the CLI to its latest stable release.
The proposed [`code-rules conflicts --prompt`](/reference/cli/#conflict-review-prompt) command prepares an agent review of conflicting guidance.
Examples illustrate how those interfaces fit together.
They are not installation instructions for a published package.

## First release

The first release will support multiple named libraries imported directly by a project, each with its own commit-or-tag ref, group selection, and exceptions.
The initial proof uses original example rules, local exceptions, and generated files that a person can inspect.
Reliable imports and a private-project pilot follow that proof.

Organization libraries that inherit from and republish other libraries, and assisted group selection, are later work.
Review orchestration and merge gates belong to the surrounding factory.

Follow development in the [public repository](https://github.com/fabricahq/code-rules).
