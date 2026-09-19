---
title: "Project"
description: "A codebase configured to use Code Rules. It owns selected libraries, local rules, exceptions, and generated guidance."
---

A **project** is a codebase configured to use Code Rules. It owns its selected libraries, local rules, and exceptions, and the generated guidance used by its agents.

Configuring a project is not the same as installing the CLI. Installing Code Rules makes the `code-rules` command available on your computer. `code-rules init` sets up a particular project.

## What a project owns

The project, not a library, decides what its agents read.

- A project can start with only local rules and no imported library.
- A project can adopt multiple libraries and pin each library's version independently.
- A project owns exclusions and replacements.

A payments service is a project. The organization's shared Go rules form a library.

## Where a project stores files

The default directory is `.code-rules/`. That folder holds the project configuration, local rules, and generated guidance.
A custom configuration path places its related directories beside that configuration file.

A project does not require a Git repository. `code-rules init` works in an ordinary directory. Sync needs Git later if the project imports a library.

See [Configuration](/reference/configuration/) for the recorded fields.

## How this fits the other concepts

- A [rule](/concepts/rule/) is one engineering expectation.
- A [group](/concepts/groups/) collects related rules and says when to read them.
- A [library](/concepts/libraries/) publishes rules for reuse.
- A project is the place that selects and uses rules.

See [Set up your first project](/start-here/set-up-project/) to create one.
