---
title: "Project status"
description: "What this documentation describes and what remains to be built."
---

**Code Rules is in early implementation.**
These docs describe the intended first release so teams can review the experience before implementation.
The documentation site, authoring rubric, and template are available.
The offline Builds module can combine in-memory library snapshots and local rules into generated files.
It emits a root group index, group pages, and individual effective definitions. Small groups include full rules; larger groups use bounded applicability indexes. Rule-level `whenToRead` metadata is required.
It has focused automated tests and a runnable example for development.
The builder supports `groups: "*"`, `"practices/*"`, and `"techs/*"` with snapshots complete for the selected scope and records the expanded group IDs in provenance.
Repository addresses use explicit HTTPS or SSH Git syntax. The builder supports GitHub.com and GitLab.com source links and retained-file links for other hosts.
Library fetching, workspace installation and consistency checks, CLI commands, and the installable authoring skill remain unimplemented.

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

The CLI commands, authoring skill workflow, configuration fields, metadata files, and format versions are proposed interfaces.
The proposed [`code-rules update`](/reference/cli/#update-the-tool) command upgrades the CLI to its latest stable release.
The proposed [`code-rules conflicts --prompt`](/reference/cli/#conflict-review-prompt) command prepares an agent review of conflicting guidance.
Examples illustrate how those interfaces fit together.
They are not installation instructions for a published package.

## First release

The first release will support multiple named libraries imported directly by a project, each with its own commit-or-tag ref, group selection, and exceptions.
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
