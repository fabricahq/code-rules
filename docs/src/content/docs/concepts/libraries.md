---
title: "Library"
description: "Share rule groups across projects and choose which rules each project adopts."
---

A **library** is an independently maintained collection of [rule groups](/concepts/groups/) that projects can import. Libraries let you share engineering practices across projects, within your organization, or with third parties.

You can publish your team's rules in a library or import libraries from others. Rules needed by only one project can stay local to that project.

## What a library contains

A library lives in a Git repository. It contains:

- **Library metadata** in `rule-library.yaml`, which declares the format and any license information.
- **Groups** under `techs/` and `practices/`, each with group metadata and Markdown rule files.

For example, a repository named "engineering-rules" might contain:

```text
engineering-rules/
  rule-library.yaml
  practices/testing/
    _group.yaml
    test-changed-behavior.md
```

The repository name is your choice. Library groups live at its root; they don't need a `.code-rules/` directory. Rules can also include [supporting assets](/reference/rule-format/#supporting-assets).

## How projects use a library

Each [project](/concepts/project/) chooses which libraries and groups to import in `.code-rules/config.yaml`. A project can use several libraries and add its own local rules.

Running `code-rules project sync` downloads the selected rules and generates the guidance agents read. Groups with the same ID combine, and the project's exclusions and replacements determine which rules appear.

A project records the version of every rule it imports. Running `code-rules project update` moves rules to newer versions and reports every rule that changed, so each project adopts updates on its own schedule. A project can also pin a rule to a version, with a reason, or import only specific rules instead of whole groups. See [Import rules](/guides/select-rules/) and [Update rules](/guides/update/) for the steps.

## How a library publishes rules

A library doesn't have one version for all of its rules. Each rule has [its own version](/concepts/rule/#how-a-rule-is-versioned), so a project can see exactly which rules changed in a library release and whether work that complied with the previous version could fail the new one.

When an author changes a rule, they add a change note saying how large the change is and what changed. A **library release** turns the pending notes into new rule versions at once, and can hold one rule change or many. Projects import only published versions, unless one deliberately imports an exact commit, so they never see changes that haven't been published.

Library releases usually go through a pull request that the release workflow opens and keeps up to date. Merging it publishes the new versions, along with one GitHub Release page that lists them. A library release has a number for reference, but it isn't a version of the library: projects choose versions rule by rule. See [Version your rules](/guides/version-rules/).

## Sharing a library

Libraries can be public or private. Imports copy their content into the consuming project, so use private libraries only in projects allowed to hold and share that content.

Code Rules preserves declared license text and notices with imported rules. For choosing terms, see [License rules](/guides/license-rules/).

To publish your own, follow [Create your first library](/start-here/create-library/). For file details, see [Library format](/reference/library-format/).
