---
title: "Project"
description: "The codebase that uses rules, and the files Code Rules adds to it."
---

A **project** is the codebase whose rules Code Rules manages, such as a web application, a service, or a CLI tool. The **project root** is that codebase's top-level directory.

A project can use rules written just for it, rules imported from one or more [libraries](/concepts/libraries/), or both. Each project chooses which rules to use and when to adopt library updates.

## What a project includes

Your codebase keeps its existing files. Code Rules adds a **Code Rules directory**, `.code-rules/`, at the project root:

```text
my-project/
  src/
  README.md
  .code-rules/
    config.yaml
    local/
    vendor/
    generated/
```

Inside `.code-rules/`:

- **Configuration** in `config.yaml` selects libraries, versions, and groups, plus any rules to exclude or replace.
- **Local rules** in `local/` hold groups and rules you author for this project.
- **Imported rules** in `vendor/` are copies of the library content your project imports.
- **Generated guidance** in `generated/` combines your selected rules into `RULES.md`, group pages, and rule files that agents can read.

You maintain the configuration and local rules. Code Rules manages the imported copies and generated files. For examples of each file, see [Project files](/reference/files/).

## Set up a project

Run `code-rules project init` from your codebase's root to create its `.code-rules/` directory. In a Git repository, subsequent project commands also work from subdirectories. Outside Git, run them from the project root.

Follow [Set up your first project](/start-here/set-up-project/) to add rules, generate guidance, and tell your agent to read it.
