---
title: "Library"
description: "Share rule groups and skills across projects and choose which ones each project adopts."
---

A **library** is an independently maintained collection of [rule groups](/concepts/groups/) and [skills](/concepts/skill/) that projects can import. Libraries let you share engineering practices across projects, within your organization, or with third parties.

You can publish your team's rules and skills in a library or import libraries from others. Rules and skills needed by only one project can stay local to that project.

## What a library contains

A library lives in a Git repository. It contains:

- **Library metadata** in `rule-library.yaml`, which declares the format and any license information.
- **Groups** under `techs/` and `practices/`, each with group metadata and Markdown rule files.
- **Skills** under `skills/`, one folder per skill, each with a `SKILL.md`. Optional.
- **Presets** under `presets/`, starting configurations that projects can extend. Optional; see [Share defaults with presets](/guides/presets/).

For example, a repository named "engineering-rules" might contain:

```text
engineering-rules/
  rule-library.yaml
  practices/testing/
    _group.yaml
    test-changed-behavior.md
  skills/build-feature/
    SKILL.md
```

The repository name is your choice. Groups and skills live at its root; they don't need a `.code-rules/` directory. Rules can also include [supporting assets](/reference/rule-format/#supporting-assets).

## How projects use a library

Each [project](/concepts/project/) chooses which libraries, groups, and skills to import in `.code-rules/config.yaml`. A project can use several libraries and add its own local rules and skills.

Running `code-rules project sync` downloads the selected rules and skills, generates the guidance agents read, and installs the skills. Groups with the same ID combine, and the project's exclusions and replacements determine which rules appear.

A project records the version of every rule and skill it imports. Running `code-rules project update` moves them to newer versions and reports every one that changed, so each project adopts updates on its own schedule. A project can also pin a rule to a version, with a reason, or import only specific rules instead of whole groups. Skills aren't pinned: all of a library's skills come from one library release, and move together when the project updates. See [Import rules](/guides/select-rules/), [Install skills](/guides/install-skills/), and [Update rules](/guides/update/) for the steps.

## How a library publishes rules

A library doesn't have one version for all of its rules and skills. Each rule and skill has its own version ([rules](/concepts/rule/#how-a-rule-is-versioned), [skills](/concepts/skill/#how-a-skill-is-versioned)), so a project can see exactly which rules changed in a library release and whether work that complied with the previous version could fail the new one.

When an author changes a rule or skill, they add a change note saying how large the change is and what changed. A **library release** turns the pending notes into new versions at once, and can hold one change or many. Projects import only published versions, unless one deliberately imports an exact commit, so they never see changes that haven't been published.

`code-rules library release` publishes a library release as one Git tag that records every rule's and skill's version, along with one GitHub Release page that lists the changes. A library release has a number for reference, but it isn't a version of the library: projects choose versions rule by rule, and take a library's skills together from one library release. See [Version your rules](/guides/version-rules/).

## Sharing a library

Libraries can be public or private. Imports copy their content into the consuming project, so use private libraries only in projects allowed to hold and share that content.

Code Rules preserves declared license text and notices with imported rules and skills. For choosing terms, see [License rules](/guides/license-rules/).

To publish your own, follow [Create your first library](/start-here/create-library/). For file details, see [Library format](/reference/library-format/).
