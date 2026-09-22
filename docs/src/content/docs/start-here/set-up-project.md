---
title: "Set up your first project"
description: "Create a local rule, give it to your agent, then import a rule from the public library."
---

A **project** is the codebase whose rules Code Rules manages. In this walkthrough, you'll give one project a local rule, connect those rules to your coding agent, and then import a rule from the Fabrica public library.

Start in the **project root**, the top-level directory of the codebase you want to work on. You need [Code Rules installed](/start-here/install/), and Git for the library import at the end. To publish guidance for several projects instead, follow [Create your first library](/start-here/create-library/).

## 1. Set up the project

```sh
code-rules project init
```

This creates the **Code Rules directory**, `.code-rules/`, with your **project configuration** in `.code-rules/config.json` and a place for project-only rules in `local/`. Imported library snapshots and generated guidance will also live here. It does not change your project's `README.md` or agent instructions.

Run all project commands below from the project root. They use `.code-rules/` in your current working directory; they do not search parent directories or discover the Git root. See [Project files](/reference/files/) for the layout.

## 2. Add your first local rule

A **local rule** belongs to this project. Let's add one about testing changed behavior.

First create its group. A group collects related rules and tells agents when to read them:

```sh
code-rules project add group practices/testing \
  --name Testing \
  --description 'Tests for the behavior this project provides.' \
  --when-to-read 'When adding or changing behavior, fixing bugs, or reviewing tests.'
```

Create `.code-rules/local/practices/testing/test-changed-behavior.md` in your editor and paste this complete rule:

```md
---
title: Test changed behavior
whenToRead: When adding or changing externally visible behavior.
impact: HIGH
impactDescription: Prevents behavior changes from silently breaking existing use cases.
tags: testing
---

## Test changed behavior

When a change alters behavior that a caller or user relies on, add or update a test for that outcome.

For example, if a discount changes an order's total, assert the resulting total rather than which private helper was called.

Run the relevant tests before considering the change complete. Changes that do not alter behavior do not need a new test solely to accompany the edit.
```

The fields at the top describe the rule; the Markdown below tells the agent what to do. You can write files directly, as here, or use [the rule authoring commands](/reference/cli/#project-add-rule) to create a draft.

Now generate the guidance your agent will read:

```sh
code-rules project build
code-rules project check
```

Open `.code-rules/generated/RULES.md`. It points to the Testing group and your rule. Edit the source under `local/` when you want to change the rule, then run `code-rules project build` again.

## 3. Give the rules to your agent

Add this section to your existing `AGENTS.md`, or the equivalent project instruction file your coding agent reads. Create that file if you don't have one; preserve any existing instructions.

```markdown
## Engineering rules

Before planning, implementing, or reviewing a change, read `.code-rules/generated/RULES.md`.
Open the groups relevant to the task, read their rules in full, and follow the applicable guidance and exceptions.
If a group links to full rule definitions or additional index pages, read those too.
```

Try a small change you already need in this project. For example, ask your agent:

> Read this project's engineering rules, then implement the change. Explain which rules apply and how you'll verify the result.

Your project now has a working local rule. Code Rules supplies the files; your agent follows the instructions you've given it. For a more detailed implementation and review workflow, see [For agents](/for-agents/).

## 4. Import a rule from the public library

Now let's reuse a rule someone else has written. A **library** is an independently maintained collection of rule groups that projects can import.

The [Fabrica public library](https://github.com/fabricahq/.code-rules-public) includes a code-design group. In `v0.1.0`, that group contains one rule: **Express operations as meaningful steps**. It helps agents keep functions understandable without extracting unnecessary helpers. Read it before deciding to adopt it.

From the same project root, run:

```sh
code-rules project add library fabrica \
  --repository https://github.com/fabricahq/.code-rules-public.git \
  --ref v0.1.0 \
  --groups practices/code-design
code-rules project sync
code-rules project check
```

`fabrica` is this project's name for the source. `--ref v0.1.0` selects the library version used in this walkthrough. `project add library` records that choice; `project sync` downloads the selected rules and regenerates the agent guidance.

Open `.code-rules/generated/RULES.md` again. You'll now see your local Testing group and the imported Code design group. Your local rule remains yours to edit. The imported rule retains its source and license information.

Review and commit `.code-rules/` and your agent instruction changes together. Your teammates and agents can then use the same guidance.

When you're ready, [import and customize more rules](/guides/select-rules/) or [update the rules you use](/guides/update/). To share rules of your own, continue with [Create your first library](/start-here/create-library/).

In a Git repository, initialize from its root. Later project commands can run from any subdirectory. Outside Git, run commands from the project root.
