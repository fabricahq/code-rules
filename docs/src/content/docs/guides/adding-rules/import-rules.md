---
title: "Import rules"
description: "Add rules from a library to your project and review the guidance your agents will read."
slug: guides/select-rules
---

A **library** shares rules across projects. Let's say your team has already written rules for handling errors and testing code. You want your project to use those rules without copying and maintaining them by hand. You'll choose both groups, import their rules, and review the result.

## Prerequisites

Before you begin:

- [Install Code Rules](/start-here/install/) and [initialize your project](/start-here/set-up-project/#1-set-up-the-project).
- Choose a Code Rules library your project can access. If you don't have one, explore [Fabrica's public rule library](https://github.com/fabricahq/public-rules/) or [create your own library](/start-here/create-library/). Find the library's Git URL, a published tag or version range, and the group paths it provides.
- Run project commands from anywhere in your Git repository. Outside Git, run them from the project root. All file paths below are relative to the project root.

## Import from a Code Rules library

1. Read the library's group descriptions and **When to read** cues. Choose groups that fit the work your project does. For this example, we'll use `practices/error-handling` and `practices/testing`. Replace every illustrative value below with your library's real repository, released version range, and group paths. The example URL does not name a published library.

   ```sh
   code-rules project add library acme-rules \
     --repository https://github.com/YOUR-ORG/YOUR-RULES.git \
     --ref '>= 1.0.0, < 2.0.0' \
     --groups practices/error-handling \
     --groups practices/testing \
     --non-interactive
   ```

   `acme-rules` is an example source name for the fictional company Acme. Choose a name that identifies your library. Repeat `--groups` to select more groups. You can also select an exact tag or full commit SHA with `--ref`; see [revision selection](/reference/configuration/#commit-or-tag-references) and [group selectors](/reference/configuration/#import-every-group).

2. Open `.code-rules/config.yaml` and review the new `sources.acme-rules` entry. The add command records the repository, both selected groups, and revision choice; it does not fetch rules. For the range above, it writes a `version` constraint. An exact tag or commit goes in `ref`. A tag can move, while a full commit SHA stays fixed.

3. Fetch the library and prepare the rules your agents will read:

   ```sh
   code-rules project sync
   code-rules project check
   ```

   With the example range, sync selects the highest matching semantic version tag and saves the selected library files under `.code-rules/vendor/acme-rules/`. It also builds `.code-rules/generated/RULES.md` and the resolved rules. Sync refreshes **all** configured sources, so review any changes beyond `acme-rules`. Check confirms that the stored inputs and generated files agree.

4. Open `.code-rules/generated/RULES.md`. Follow the "Error handling" and "Testing" group links and read their rules. Both groups now share one starting point for your agents. Check `.code-rules/generated/provenance.json` for the selected tag and resolved commit. Review the Git diff, including configuration, vendor files, and generated guidance, then commit those files together. Your agents can use the [resolved-rule workflow](/for-agents/) to select relevant rules for each task.

## Adapt the import to your project

To add local guidance or exclude or replace a shared rule, follow [Customize imported rules](/guides/customize/). A source's position in configuration never makes its rules override another source's rules. Keep any project exception explicit and review the generated result.

## From another source

Code Rules imports compatible libraries. The CLI does not turn a style guide, linter rule, or other document into a rule automatically. To use guidance from another source:

1. Read the material at a specific revision and confirm you may adapt and share it. Retain the license text and required notices. A citation alone does not grant permission. See [License a library](/guides/license-rules/).
2. [Create a library](/start-here/create-library/) that you own. Write the adapted rule there, keep the source revision in its [attribution](/reference/rule-library-format/#rule-attribution), and explain your changes. Run `code-rules library check`, review the rule and retained terms, then publish a version.
3. Return to [Import from a Code Rules library](#import-from-a-code-rules-library) using your published URL, version, and group. Publishing a new version later does not change the project's adopted rules. Change the project's revision choice if needed, then [sync and review the update](/guides/update/).
