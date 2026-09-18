---
title: "Project status"
description: "Implemented Go CLI capabilities and work remaining before a public release."
---

**The Go CLI is implemented. Public release publication remains separate work.** Build it from source or inspect an unpublished binary candidate using the [installation guide](/guides/install/).

## What works in this checkout

- Project initialization, managed agent instructions, local groups and rules, and source configuration.
- Independent library initialization, groups, rules, declared terms, and read-only library validation.
- Interactive terminal prompts and explicit flags; human output by default and structured `--json` results without prompts.
- Git imports from exact commits, exact tags, or the highest tag matching a HashiCorp version constraint.
- Multiple sources, explicit or wildcard group selections, exclusions, complete local replacements, and local additions.
- Local group metadata precedence while retaining all contributed metadata in provenance.
- Original-byte vendor snapshots, offline digest verification, and coordinated file updates with recovery.
- Generated root/group indexes, complete rule files, declared license and notice copies, and deterministic provenance.
- Full rules inline when the group fits; summaries and line-bounded pagination for larger groups.
- Offline build and read-only check, including verification of the managed project README.
- Candidate binary packaging, extracted-executable tests, and upgrade/rollback checks for macOS and Linux.
- Release automation that tests assets, attaches them to a draft, and publishes after a maintainer merges a release-notes PR.

Start with [project setup](/guides/set-up-project/), [library authoring](/guides/create-library/), or the [CLI reference](/reference/cli/). The shared [rule rubric and template](/reference/rule-authoring/) are available to humans and agents.

## What remains

- First public release publication and any signing/notarization activation.
- An installable Code Rules authoring skill.
- [`code-rules update`](/reference/cli/#update-the-tool) and [`code-rules conflicts --prompt`](/reference/cli/#conflict-review-prompt), which are proposed and not implemented.
- Semantic group-level update reports, assisted group selection, and libraries that inherit from and republish other libraries.

The configuration and file formats are implemented but unreleased and may still change. Package builds and passing tests do not establish publication approval.

## Outside the product scope

Code Rules manages which engineering rules a project adopts and delivers readable guidance. Application compliance checks, enforcement, review orchestration, and merge gates belong to project instructions or separate tooling. A successful `check` verifies file consistency, not application compliance.

Follow development in the [public repository](https://github.com/fabricahq/code-rules).
