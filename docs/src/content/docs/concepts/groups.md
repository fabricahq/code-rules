---
title: "Group"
description: "How groups organize rules and help agents find relevant guidance."
---

A **rule group** is a collection of related [rules](/concepts/rule/) and tells agents when to read them.

Groups can live in a **library** for projects to import, or locally in a **project** for that project alone. Projects can combine imported groups with their own local rules.

## Group types

Every group must be either a **technology group** under `techs/` or a **practice group** under `practices/`.

### Technologies

**Technology groups** live under `techs/` and cover named technologies, such as "TypeScript", "React", "Playwright", or "AWS".

For example, `techs/playwright` holds rules about writing browser tests with Playwright.

### Practices

**Practice groups** live under `practices/` and cover practices that apply across technologies, such as "Testing", "Observability", or "Error handling".

For example, `practices/testing` holds rules about testing behavior, regardless of the language or test framework.

## Group identities

A group's ID is its path starting with `techs/` or `practices/`. A group named "Automated Tests" might have the ID `practices/testing`.

When a project builds or syncs, local and imported rules with the same group ID appear together in the generated group.

Use `--name` when creating a group to give it a readable title; the title does not change its ID.

## What a group contains

A group is a folder containing:

- **Metadata** in `_group.yaml`: its name, description, and when agents should read it.
- **Rules**: one Markdown file per rule.

For example, from a library repository's root:

```text
/practices/testing/
  _group.yaml
  verify-retry-limits.md
  test-changed-behavior.md
```

For a local group, the same files live in `/.code-rules/local/practices/testing/`, relative to the project root.

You maintain the metadata and rules you author. The CLI supplies group READMEs; you don't need to keep them in sync. Building or syncing creates the guidance agents read.

See [Rule and library format](/reference/rule-library-format/) for the fields and file layout.

## Assets

Rules can optionally link to diagrams, sample data, or longer explanations in `assets/`. You maintain these supporting files; Code Rules includes them in the generated guidance. See [Supporting assets](/reference/rule-library-format/#supporting-assets) for the layout.

## When to read a group

Agents start at the generated `RULES.md`, choose groups using their `whenToRead` descriptions, then read those groups' rules. Each rule's conditions and exceptions determine whether it applies to the task.

For the complete reading process, see the [agent workflow](/for-agents/). To choose groups for your project, follow [Import rules](/guides/select-rules/).
