---
title: "Library"
description: "Understand who owns shared rules, project exceptions, and the tooling that connects them."
---

A **library** is a Git repository containing rule groups.
The Code Rules tool imports its content at a specific commit.

## Keep tooling and opinions separate

[Code Rules](https://github.com/fabricahq/code-rules) owns the conventions, importer, and documentation.
A library owns its engineering opinions and their provenance.
A consuming [project](/concepts/project/) owns the versions and exceptions it adopts.

The importer works with compatible libraries independently of who publishes them.
Sync uses the caller's Git credentials to access private libraries.

## Create or adapt a library

The [library authoring commands](/start-here/create-library/) initialize a library, add groups and rules explicitly, and validate the input format.
Use them in a new or existing repository; a dedicated repository name is not required.

Code Rules imports libraries that follow its format.
For guidance from other sources, first [author a compatible adaptation](/guides/adapt-rules/) in a compatible library, even when it contains only one rule.
Agents can assist with authoring; automatic conversion of arbitrary repositories is outside the import workflow.
The adaptation's author maintains its meaning, attribution, retained terms, and updates from the original source.

## Name an organization library

We recommend `<organization>/.code-rules`, such as `acme/.code-rules`.
The name is a convention, not automatic discovery.
A project's configuration explicitly names each source repository, exact ref or version constraint, and selected groups.

The consuming project's default directory is `.code-rules/`.
A custom configuration path places its related directories beside that configuration file.
Its name does not depend on the remote repository name.

## Share only what applies

An organization may maintain rules for many projects.
A rule naming one application's internal packages or contracts does not automatically apply to the rest of the organization.
Keep that obligation local to the applicable project, or give its group an explicit scope.

## Private content stays private only if consumers do

Vendoring copies source rules and generated text into the consuming repository's Git history.
Use a library only in projects authorized to receive and distribute its content.
Preserve required attribution and license notices during imports.
See [License rules](/guides/license-rules/) for what library terms should cover and how consumers retain them.

## Import multiple canonical sources

A project can import multiple libraries directly.
For example, a project can import `fabricahq/.code-rules-example` as `fabrica` and `acme/.code-rules` as `acme`.
Each source owns its rules, and the project selects each source's exact ref or version constraint independently.
Sync records each resolved commit so offline work uses the exact imported snapshot.
The repository names illustrate the configuration; verify available rules before selecting groups.

Rules from sources that share a group combine into one group page, which includes full rules for small groups or links to each resolved rule from applicability summaries.
Source-qualified IDs keep their origins distinct, and source order grants no override priority.
Use explicit project exclusions or replacements to resolve competing obligations.

## Inspect an adopted library

Each imported source gets a generated `libraries/<source-name>/README.md`.
It identifies the repository, requested revision, resolved commit, and declared license, and links to the authoritative provenance.
Declared license and notice copies live in that folder’s `licenses/` directory; libraries without declarations omit that directory.
The full resolved rules live separately in `generated/rules/`.

## Nested library inheritance

Code Rules does not import a library that itself inherits and republishes other libraries.
A project imports each canonical library directly.
Any later publishing mechanism needs to preserve rule identities and provenance across levels.

See [Project](/concepts/project/) for the codebase that selects and uses a library.
