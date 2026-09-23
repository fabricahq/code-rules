---
title: "Set up your first project"
description: "Write a rule for your coding agent, then reuse rules written by others."
---

We're going to enable your coding agent to get consistent instructions for how to write and review code in your codebase. In Code Rules, that codebase is your **project**.

We will declare guidance in a **rule,** which is a Markdown file that explains one practice you want your agent to follow, such as testing changes to your code.

In this walkthrough, you'll:

1. Write a rule just for your project. We call this a **local rule**.
2. Build the files your agent will read from that rule.
3. Add a rule written by someone else. You'll get it from a **library**, a collection of rules maintained separately for projects to reuse. We'll use Fabrica's public library as an example.
4. Tell your coding agent where to find and read both rules, then try a task.
5. Commit the guidance so everyone working on the project can use it.

Start in the **project root**, the top-level directory of the codebase you want to work on. You need [Code Rules installed](/start-here/install/), and Git to import the library. To publish rules for other projects to use instead, follow [Create your first library](/start-here/create-library/).

In a Git repository, initialize from its root. Later project commands can run from any subdirectory. Outside Git, run commands from the project root.

## 1. Set up the project

```sh
code-rules project init
```

This creates the **Code Rules directory**, `.code-rules/`, with your **project configuration** in `.code-rules/config.json` and a place for project-only rules in `local/`. Initializing Code Rules does not change your project's `README.md` or agent instructions.

Later, when you import a library, Code Rules saves a copy of its files at the version you selected in `vendor/`. Code Rules prepares your local and selected imported rules in `generated/`, with a `RULES.md` index that helps agents find the rules to read.

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

Use the CLI to create a draft rule in that group:

```sh
code-rules project add rule practices/testing/test-changed-behavior \
  --title 'Test changed behavior' \
  --when-to-read 'When adding or changing externally visible behavior.' \
  --impact HIGH \
  --impact-description 'Prevents behavior changes from silently breaking existing use cases.'
```

Open `.code-rules/local/practices/testing/test-changed-behavior.md` in your editor and replace its entire contents with this complete rule:

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

The fields at the top describe the rule; the Markdown below tells the agent what to do. Replacing the draft also removes its unfinished-draft marker, so the rule is ready to build.

## 3. Build the guidance your agent will read

Adding a rule saves the instructions you've written. **Building prepares your project's rules for agents to use.**

With one rule, you could point your agent directly at the Markdown file. With dozens of rules from your project and several libraries, agents need help finding the relevant ones. You also need a way to choose which rules apply to your project.

Run:

```sh
code-rules project build
```

Build reads your rules and `.code-rules/config.json`, validates them, and writes the results into `.code-rules/generated/`. It:

- **Applies your choices:** includes the selected rules, leaves out rules you've excluded, and substitutes your local replacements for imported rules where configured.
- **Organizes the rules for reading:** creates `RULES.md` and group indexes with links and instructions that help agents find and read relevant rules. See [a sample `RULES.md` and the files it links to](/reference/files/#example-follow-a-rule-from-the-index).
- **Splits large indexes into pages:** keeps navigation manageable without shortening the full rule text.

If two rules give conflicting advice, you decide which to exclude or replace in configuration. Build applies those decisions; it doesn't detect or resolve contradictory advice on its own.

For this first build, open `.code-rules/generated/RULES.md` and follow its links to the Testing group and your rule. You've turned your source rule into part of the organized guidance your agent will use.

Keep editing rules in `local/`, then build again to update the agent's copy. Don't edit `generated/` directly.

You can verify that the generated files match your source files:

```sh
code-rules project check
```

This checks the rule files, not whether your application follows the rules.

## 4. Import a rule from the public library

Now let's reuse a rule someone else has written. A **library** is an independently maintained collection of rule groups that projects can import.

The [Fabrica public library](https://github.com/fabricahq/.code-rules-public) includes a code-design group. In `v0.1.0`, that group contains one rule: **Express operations as meaningful steps**. It helps agents keep functions understandable without extracting unnecessary helpers. Read it before deciding to adopt it.

From the same project root, run:

```sh
code-rules project add library fabrica \
  --repository https://github.com/fabricahq/.code-rules-public.git \
  --ref '>= 0.1.0, < 0.2.0' \
  --groups practices/code-design
```

This adds the library's repository, version constraint, and selected group to `.code-rules/config.json` under the source name `fabrica`.

The version constraint `>= 0.1.0, < 0.2.0` allows updates within the `0.1.x` series. Each time you run `code-rules project sync`, Code Rules automatically selects the newest release that matches.

Now run `code-rules project sync` to download the selected rules and build the agent guidance. You don't need to run `code-rules project build` separately.

```sh
code-rules project sync
code-rules project check
```

Open `.code-rules/generated/RULES.md` again. You'll now see your local Testing group and the imported "Code design" group. Your local rule remains yours to edit. The imported rule retains its source and license information.

## 5. Give the rules to your agent

Your local and imported rules are now available in the repository, but you haven't yet told your coding agent to read them. Let's add that instruction.

Add this section to your existing `AGENTS.md`, or the equivalent project instruction file your coding agent reads. Create that file if you don't have one; preserve any existing instructions.

```markdown
## Engineering rules

Before planning, implementing, reviewing, testing, or debugging a change:

1. Read `.code-rules/generated/RULES.md` and follow its instructions to
   select relevant groups and read their rules in full, including linked
   files and additional index pages.
2. Follow the applicable rules and their exceptions while doing the work.
3. Before finishing, check your work against those rules and run the
   relevant validation. Briefly report what you verified and any gaps.

If required rule files are unavailable or give conflicting instructions,
report the issue rather than silently skipping them or choosing a policy.
```

Now ask your agent for a change as you normally would. For example, in a checkout project:

> Fix the bug where applying an expired coupon still reduces the order total.

You don't need to mention Code Rules in each request. The project instructions tell your agent to read and apply the rules automatically.

For your first task, check that the agent reads the relevant rules and validates its changes. In this example, the testing rule calls for a test covering the expired coupon. These instructions guide the agent; they don't guarantee compliance. For a more detailed implementation and review workflow, see [For agents](/for-agents/).

## 6. Commit the guidance for your team

Once you've confirmed your agent reads and applies the rules to your liking, review and commit the entire `.code-rules/` directory and your agent instruction changes (`AGENTS.md`) together. This includes your configuration, local rules, imported library files, and generated guidance.

Anyone who checks out the repository will then have the same rules and instructions for their agent to read on future tasks.

## Next steps

When you're ready, [import and customize more rules](/guides/select-rules/) or [update the rules you use](/guides/update/).

To share rules of your own, continue with [Create your first library](/start-here/create-library/).
