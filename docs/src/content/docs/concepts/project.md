---
title: "Project"
description: "A codebase configured to use Code Rules. It owns selected libraries, local rules, exceptions, and generated guidance."
---

A **project** is the codebase whose rules Code Rules manages. Its **project root** is the top-level directory of that codebase. The project selects libraries, authors project-only rules, and chooses exceptions for its agents.

Configuring a project is not the same as installing the CLI. Installing Code Rules makes the `code-rules` command available on your computer. `code-rules project init` sets up a particular project.

## What a project owns

The project, not a library, decides what its agents read.

- A project can start with only local rules and no imported library.
- A project can adopt multiple libraries and pin each library's version independently.
- A project owns exclusions and replacements.

A payments service is a project. The organization's shared Go rules form a library.

## Where a project stores files

Code Rules stores this project's configuration and rules in the `.code-rules/` directory at its root. Rules can be project-only, imported from libraries, or both.

This is the **Code Rules directory**. It contains project configuration, project-only rules in `local/`, imported library snapshots in `vendor/`, and generated guidance in `generated/`. The **project configuration** is `.code-rules/config.json`.

Run project commands from the project root. They use `.code-rules/` in your current working directory; Code Rules does not search parent directories or discover a Git root.

A project does not require a Git repository. `code-rules project init` works in an ordinary directory. Sync needs Git later if the project imports a library.

See [Configuration](/reference/configuration/) for the recorded fields.

## How this fits the other concepts

- A [rule](/concepts/rule/) is one engineering expectation.
- A [group](/concepts/groups/) collects related rules and says when to read them.
- A [library](/concepts/libraries/) publishes rules for reuse.
- A project is the codebase that selects and uses rules.

See [Set up your first project](/start-here/set-up-project/) to set up rules for your codebase.
