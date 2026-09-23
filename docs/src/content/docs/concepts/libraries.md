---
title: "Library"
description: "Share rule groups across projects and choose which rules each project adopts."
---

A **library** is an independently maintained collection of [rule groups](/concepts/groups/) that projects can import. Libraries let you share engineering practices across projects, within your organization, or with third parties.

You can publish your team's rules in a library or import libraries from others. Rules needed by only one project can stay local to that project.

## What a library contains

A library lives in a Git repository. It contains:

- **Library metadata** in `rule-library.json`, which declares the format and any license information.
- **Groups** under `techs/` and `practices/`, each with group metadata and Markdown rule files.

For example, a repository named "engineering-rules" might contain:

```text
engineering-rules/
  rule-library.json
  practices/testing/
    _group.json
    test-changed-behavior.md
```

The repository name is your choice. Library groups live at its root; they don't need a `.code-rules/` directory. Rules can also include [supporting assets](/reference/rule-library-format/#supporting-assets).

## How projects use a library

Each [project](/concepts/project/) chooses which libraries, versions, and groups to import in `.code-rules/config.json`. A project can use several libraries and add its own local rules.

Running `code-rules project sync` downloads the selected rules and generates the guidance agents read. Groups with the same ID combine, and the project's exclusions and replacements determine which rules appear.

Library authors publish updates as new versions. Each project chooses when to adopt them by syncing with an exact version or a version constraint. See [Import rules](/guides/select-rules/) and [Update rules](/guides/update/) for the steps.

## Sharing a library

Libraries can be public or private. Imports copy their content into the consuming project, so use private libraries only in projects allowed to hold and share that content.

Code Rules preserves declared license text and notices with imported rules. For choosing terms, see [License rules](/guides/license-rules/).

To publish your own, follow [Create your first library](/start-here/create-library/). For file details, see [Rule and library format](/reference/rule-library-format/).
