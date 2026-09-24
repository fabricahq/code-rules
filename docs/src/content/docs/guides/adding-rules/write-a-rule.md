---
title: "Write a rule"
description: "Add a local rule to an existing project and review its guidance."
slug: guides/write-rules
---

A **rule** is a Markdown file that tells an agent when and how to apply an engineering practice. This guide shows how to create one rule and add it to an existing project.

## Prerequisites

Before you begin:

- [Install Code Rules](/start-here/install/).
- [Initialize your project](/start-here/set-up-project/#1-set-up-the-project) to create `.code-rules/config.yaml`, if you haven’t already.
- Run project commands from anywhere in your Git repository. Outside Git, run them from the project root. All file paths below are relative to the project root.

<span id="add-a-rule-to-an-existing-project"></span>

## Write the rule

Our first task is to write a new rule. Let's say that your app limits uploads to 10 MB, but agents keep adding the vague message "Upload failed" when someone selects a larger file. After correcting this more than once, you want future agents to explain the limit and what the user can do. We'll capture that lesson in a local rule.

First, we'll need to choose a [group](/concepts/groups) where the rule will live. What we want to capture about error messages is more philosophical in nature and not specific to any one technology, so this rule belongs in a **practice** group (versus a **technology** group). If `practices/error-handling` does not exist locally, create it:

```sh
code-rules project add group practices/error-handling \
  --name 'Error handling' \
  --description 'Help users understand and recover from errors.' \
  --when-to-read 'When writing or reviewing errors shown to users.' \
  --non-interactive
```

Then create a draft rule in that group:

```sh
code-rules project add rule practices/error-handling/make-errors-actionable \
  --title 'Make errors actionable' \
  --when-to-read 'When writing or reviewing validation errors shown to users.' \
  --impact MEDIUM \
  --impact-description 'Helps users recover from invalid input without guessing what to change.' \
  --non-interactive
```

Open `.code-rules/local/practices/error-handling/make-errors-actionable.md` in your editor. The generated draft contains a `<!-- code-rules:draft -->` marker to show that it is unfinished. While that marker remains, `code-rules project build`, `code-rules project sync`, and `code-rules project check` report an error identifying the draft. Existing generated guidance stays unchanged.

Replace the file’s entire contents with the example below. It includes the rule’s instruction, an example error message, and a way to check the behavior. Replacing the entire draft also removes the marker, so you can build the completed rule:

<span id="example-rule"></span>

````md
---
title: Make errors actionable
whenToRead: When writing or reviewing validation errors shown to users.
impact: MEDIUM
impactDescription: Helps users recover from invalid input without guessing what to change.
tags: errors
---

## Make errors actionable

When a validation error blocks a user's action, say what failed and what they can change. Include the relevant limit or requirement when it helps them recover. Do not show private details or suggest retrying unchanged input that will fail again.

For example, if uploads must be 10 MB or smaller, replace "Upload failed" with "This file is larger than 10 MB. Choose a file no larger than 10 MB." Apply the same guidance to other validation errors, such as a missing required field.

Check the message against the actual validation condition. Confirm that an oversized file shows the size limit, a file within the limit does not show that error, and the suggested next step would let the user continue.
````

:::tip
In this case, we're populating a new rule by copying an example. In everyday use, you would tell your agent to write a rule that aligns with the [rubric](#rubric-and-template) below. This is especially powerful when you want to formalize guidance you gave an agent one time into something it will remember all the time.
:::

Run `code-rules project build` so that your new rule is included in `.code-rules/generated/` and the indexes agents use to find the rule are updated. Then run `code-rules project check` to confirm those generated files match your rules and configuration:

```sh
code-rules project build
code-rules project check
```

Open `.code-rules/generated/RULES.md`, follow the Error handling group link, and confirm the resolved rule tells agents to name the failed constraint and a useful next step. Keep future edits in `local/`, then build and check again. You can change the rule’s title and wording without renaming its file. If you move or rename the file, update any configuration entries or links that refer to its old path.

<span id="rubric-template-and-skill"></span>

## Rubric and template

A useful rule tells an agent when it applies, what to do, and how to check its work. Code Rules provides a recommended rubric and a matching template to help you cover those points without starting from scratch:

- The [authoring rubric](/reference/rule-authoring/#authoring-rubric) explains what makes a rule useful, with questions to consider when writing or reviewing one.
- The [Markdown template](/reference/rule-authoring/#markdown-template) turns that advice into a starting structure, with sections for guidance, rationale, examples, and validation. `code-rules project add rule` uses this template to create the draft you edited above.

### Adapt the template to your rule

A simple rule may need only a few paragraphs; keep the sections that help explain it. The rubric is _recommended guidance_, not a hard requirement. Code Rules requires valid YAML frontmatter with `title`, `whenToRead`, `impact`, and `impactDescription`, followed by a non-empty Markdown body. The [file format reference](/reference/rule-library-format/#rule-metadata) lists accepted fields and values. Your team can choose additional writing conventions, such as including a rationale and examples in every rule.

If your rule adapts someone else’s guidance, follow [Import rules from another source](/guides/select-rules/#from-another-source) for the steps to retain its attribution and license terms.
