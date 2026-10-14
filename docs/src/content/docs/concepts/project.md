---
title: "Project"
description: "The codebase that uses rules and skills, and the files Code Rules adds to it."
---

A **project** is the codebase whose rules and skills Code Rules manages, such as a web application, a service, or a CLI tool. The **project root** is that codebase's top-level directory.

A project can use rules and skills written just for it, ones imported from [libraries](/concepts/libraries/) and other repositories, or both. Each project chooses which ones to use and when to adopt updates.

## What a project includes

Your codebase keeps its existing files. Code Rules adds a **Code Rules directory**, `.code-rules/`, at the project root, and, once you install skills, a copy of each skill in the folders your agents read:

```text
my-project/
  src/
  README.md
  .code-rules/
    config.yaml
    local/
    vendor/
    generated/
  .claude/skills/
  .agents/skills/
```

Inside `.code-rules/`:

- **Configuration** in `config.yaml` selects libraries, groups, skills, and versions, plus any rules or skills to exclude or replace.
- **Local rules and skills** in `local/` hold groups, rules, and skills you author for this project.
- **Imported content** in `vendor/` is a copy of the library and repository content your project imports.
- **Generated guidance** in `generated/` combines your selected rules into `RULES.md`, group pages, and rule files that agents can read, and lists installed skills in `SKILLS.md`.

Outside `.code-rules/`, Code Rules writes **installed skills** into `.claude/skills/` and `.agents/skills/`, the targets your configuration names. It manages only the skill folders it installed there; other tools' skills in the same folders stay untouched.

You maintain the configuration and local rules and skills. Code Rules manages the imported copies, generated files, and installed skills. For examples of each file, see [Project files](/reference/files/).

## Set up a project

Run `code-rules project init` from your codebase's root to create its `.code-rules/` directory. In a Git repository, subsequent project commands also work from subdirectories. Outside Git, run them from the project root.

Follow [Set up your first project](/start-here/set-up-project/) to add rules, generate guidance, and tell your agent to read it.
