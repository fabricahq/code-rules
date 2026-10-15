---
title: "Set up your first project"
description: "Write a rule for your coding agent, then reuse rules written by others."
---

We're going to enable your coding agent to get consistent instructions for how to write and review code in your codebase. In Syllabus, that codebase is your **project**.

We will declare guidance in a **rule,** which is a Markdown file that explains one practice you want your agent to follow, such as testing changes to your code.

In this walkthrough, you'll:

1. Write a rule just for your project. We call this a **local rule**.
2. Build the files your agent will read from that rule.
3. Add a rule written by someone else. You'll get it from a **library**, a collection of rules maintained separately for projects to reuse. We'll use Fabrica's public library as an example.
4. Tell your coding agent where to find and read both rules, then try a task.
5. Commit the guidance so everyone working on the project can use it.

Start in the **project root**, the top-level directory of the codebase you want to work on. You need [Syllabus installed](/start-here/install/), and Git to import the library. To publish rules for other projects to use instead, follow [Create your first library](/start-here/create-library/).

## 1. Set up the project

```sh
syllabus project init
```

This creates the **Syllabus directory**, `.syllabus/`, with your **project configuration** in `.syllabus/config.yaml` and a place for project-only rules in `rules/`. It also adds a section to your project's `AGENTS.md`, creating the file if needed, which you'll look at in step 5. Initializing Syllabus doesn't change your project's `README.md` or your own text in `AGENTS.md`.

Later, when you import a library, Syllabus saves a copy of its files, at the rule versions your configuration selects, in `vendor/`. Syllabus prepares your local and selected imported rules in `generated/`, with a `RULES.md` index that helps agents find the rules to read.

Run the commands below from the project root. After initialization, you can also run project commands from its Git subdirectories. See [Project files](/reference/files/) for the layout.

## 2. Add your first local rule

A **local rule** belongs to this project. Let's add one about testing changed behavior.

First create its group. A group collects related rules and tells agents when to read them:

```sh
syllabus project add group practices/testing \
  --name Testing \
  --description 'Tests for the behavior this project provides.' \
  --when-to-read 'When adding or changing behavior, fixing bugs, or reviewing tests.'
```

Use the CLI to create a draft rule in that group:

```sh
syllabus project add rule practices/testing/test-changed-behavior \
  --title 'Test changed behavior' \
  --when-to-read 'When adding or changing externally visible behavior.' \
  --impact HIGH \
  --impact-description 'Prevents behavior changes from silently breaking existing use cases.'
```

Open `.syllabus/rules/practices/testing/test-changed-behavior.md` in your editor and replace its entire contents with this complete rule:

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
syllabus project build
```

`syllabus project build` reads your rules and `.syllabus/config.yaml`, validates them, and writes the results into `.syllabus/generated/`. It:

- **Applies your choices:** includes the selected rules, leaves out rules you've excluded, and substitutes your local replacements for imported rules where configured.
- **Organizes the rules for reading:** creates `RULES.md` and group indexes with links and instructions that help agents find and read relevant rules. See [a sample `RULES.md` and the files it links to](/reference/files/#example-follow-a-rule-from-the-index).
- **Splits large indexes into pages:** keeps navigation manageable without shortening the full rule text.

If two rules give conflicting advice, you decide which to exclude or replace in configuration. `syllabus project build` applies those decisions; it doesn't detect or resolve contradictory advice on its own.

For this first build, open `.syllabus/generated/RULES.md` and follow its links to the Testing group and your rule. You've turned your source rule into part of the organized guidance your agent will use.

Keep editing rules in `rules/`, then build again to update the agent's copy. Don't edit `generated/` directly.

You can verify that the generated files match your source files:

```sh
syllabus project check
```

This checks the rule files, not whether your application follows the rules.

## 4. Import a rule from the public library

Now let's reuse a rule someone else has written. A **library** is an independently maintained collection of rule groups that projects can import.

The [Fabrica public library](https://github.com/fabricahq/public-rules) includes a code-design group. Its rules help agents structure functions, modules, and folders, such as **Express operations as meaningful steps**, which keeps functions understandable without extracting unnecessary helpers. Read them before deciding to adopt them.

From the same project root, run:

```sh
syllabus project add source fabrica \
  --repository https://github.com/fabricahq/public-rules.git \
  --rules 'practices/code-design/*'
```

This adds the library's repository and selected group to `.syllabus/config.yaml` under the source name `fabrica`. A source can also supply [skills](/concepts/skill/), which sync installs where your agents find them; see [Install skills](/guides/install-skills/).

Each rule in Fabrica's library has its own version. The project follows each rule's newest version whenever you run `syllabus project update`.

Now run `syllabus project sync` to download the selected rules and build the agent guidance. You don't need to run `syllabus project build` separately. `syllabus project sync` records the version of each rule it imported, so your project keeps the same rules until you choose to [update them](/guides/update/).

```sh
syllabus project sync
syllabus project check
```

Open `.syllabus/generated/RULES.md` again. You'll now see your local Testing group and the imported "Code design" group. Your local rule remains yours to edit. Each imported rule shows its version and retains its source and license information.

## 5. Give the rules to your agent

Your local and imported rules are now available in the repository. Your coding agent reads them because `syllabus project init` added this section to `AGENTS.md`, between `<!-- syllabus:begin -->` and `<!-- syllabus:end -->` markers:

```markdown
## Skills and Rules

Before planning, implementing, reviewing, testing, or debugging a change:

1. Read `.syllabus/generated/RULES.md` and follow its instructions to
   select relevant groups and read their rules in full, including linked
   files and additional index pages.
2. If `.syllabus/generated/SKILLS.md` lists a skill that fits the task,
   use it, and read the rules it cites before the steps that apply them.
3. Follow the applicable rules and their exceptions while doing the work.
4. Before finishing, check your work against those rules and run the
   relevant validation. Briefly report what you verified and any gaps.

If required rule files are unavailable or give conflicting instructions,
report the issue rather than silently skipping them or choosing a policy.

To add, change, or update rules or skills, follow `.syllabus/README.md`.
```

Step 2 matters once the project installs [skills](/concepts/skill/). Syllabus keeps the section current whenever you run `init`, `build`, `sync`, or `update`, and never changes your text outside the markers. Don't edit the section itself; write your own instructions above or below it. To manage `AGENTS.md` yourself instead, turn the section off with [`manageAgentsFile: false`](/reference/configuration/#agentsmd-section).

If your agent reads a different instruction file, have it import or link `AGENTS.md`. For Claude Code, add the line `@AGENTS.md` to `CLAUDE.md`.

Now ask your agent for a change as you normally would. For example, in a checkout project:

> Fix the bug where applying an expired coupon still reduces the order total.

You don't need to mention Syllabus in each request. The project instructions tell your agent to read and apply the rules automatically.

For your first task, check that the agent reads the relevant rules and validates its changes. In this example, the testing rule calls for a test covering the expired coupon. These instructions guide the agent; they don't guarantee compliance. For a more detailed implementation and review workflow, see [For agents](/for-agents/).

## 6. Commit the guidance for your team

Once you've confirmed your agent reads and applies the rules to your liking, review and commit the entire `.syllabus/` directory and `AGENTS.md` together. This includes your configuration, local rules, imported library files, and generated guidance.

Anyone who checks out the repository will then have the same rules and instructions for their agent to read on future tasks.

## Next steps

When you're ready, [import and customize more rules](/guides/select-rules/) or [update the rules you use](/guides/update/).

To share rules of your own, continue with [Create your first library](/start-here/create-library/).
