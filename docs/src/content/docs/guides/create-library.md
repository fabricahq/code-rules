---
title: "Create a rule library"
description: "Initialize a library, add groups and rules, validate its files, and share it through Git."
---

A rule library lets several projects adopt and update the same engineering guidance.
Create one in a new or existing Git repository, then add groups and rules using the shared authoring template.
For a rule used by only one project, you can instead [author it locally](/guides/write-rules/).

The library commands work through the development CLI or the [locally installed release candidate](/guides/install/).
Initialize from the Git repository root. Other library commands can run from any subdirectory. Use `--directory path` to author or check a library elsewhere; initialization still requires the selected target to be a repository root. Outside Git, use the library root or an explicit `--directory`.

## 1. Initialize the library

From the repository root, run:

```sh
code-rules library init
```

The command creates `rule-library.json` and a short README linking to the canonical authoring guidance.
Supply one license covering the whole library, including its rules and examples, or supply the declaration and file for existing terms.
Rule and group license overrides are unsupported. Use separate libraries for material that requires different declarations.
Code Rules records that choice and retains the actual license text and any notices.
To initialize with explicit terms, supply existing UTF-8 files:

```sh
code-rules library init --spdx MIT --license-file /path/to/LICENSE.md --notice-file /path/to/NOTICE.md
```

Omit `--notice-file` when no notice is needed. Initialization copies the supplied bytes to root-level `LICENSE.md` and `NOTICE.md` and declares those paths in `rule-library.json`.
It does not download or write a license from the SPDX identifier.
Manually authored manifests can declare other contained paths. Vendoring and generation standardize retained license locations.
Code Rules does not select a license for you or infer one from existing text.
If you defer the choice, the manifest leaves the license undeclared and the command reminds you to decide the terms before sharing.

Initialization preserves existing files and reports collisions before writing anything.
The command does not create a remote repository, commit files, or publish content.

## 2. Add a group

```sh
code-rules library add group techs/javascript
```

Provide the group's name, description, and one `whenToRead` string. The command also creates a group README explaining authoring and validation.
The command creates `techs/javascript/_group.json`.
You can create a group before adding any rules.

Describe the whole group's scope, including rules you expect it to contain later.
For example, a JavaScript group's cue might be “Planning, writing, changing, or reviewing JavaScript code.”
Follow the [group authoring guidance](/reference/rule-authoring/#write-whentoread-guidance-that-helps-selection).
For guidance that crosses technologies, create a group such as `practices/testing` instead.

## 3. Add a rule

```sh
code-rules library add rule techs/javascript/prefer-for-of
```

The command creates `prefer-for-of.md` in the existing group using the canonical template.
Create the group before adding a rule. If it is missing, the Go CLI returns an error with the `code-rules library add group GROUP_PATH` command; it does not prompt to create the group.
Fill in the required metadata and draft the obligation, conditions, and exceptions.
Add implementation and validation sections when they provide useful guidance.
Tags are optional.

The draft contains a `<!-- code-rules:draft -->` marker. Complete the guidance, remove unused template prompts, and remove that marker before running `library check`.
Use `--body-file path` to supply a completed Markdown body.
The template is a starting point, not a rule to adopt unchanged.
An agent can help draft and review the text using the [authoring rubric](/reference/rule-authoring/#authoring-rubric).
For material from another source, follow [Adapt a third-party rule](/guides/adapt-rules/).

After completing these steps with a declared license, the library contains:

```text
README.md
rule-library.json
LICENSE.md
techs/
  javascript/
    _group.json
    README.md
    prefer-for-of.md
```

## 4. Validate and review

```sh
code-rules library check
```

The command checks the format, group and rule metadata, paths, asset ownership, local Markdown links, and declared license and notice files without changing them.
It reads `techs/`, `practices/`, `assets/`, the manifest, and declared terms. Unrelated repository content, such as `.git/` and `node_modules/`, is outside the check.
It reports incomplete required fields with their file paths.
An empty, well-formed group is valid; the output reports group and rule counts so authors can see what they have created.
Missing license declarations produce a visible warning, consistent with the format's support for undeclared terms.

A successful check establishes that the library meets the input format.
Review the guidance and license declarations separately; the check cannot establish their accuracy or legal sufficiency.

## 5. Share through Git

Review and commit the source rules, metadata, and retained terms together.
Publish the repository through your usual Git workflow and publish complete semantic version tags, such as `v1.0.0`, so consumers can use version constraints. Consumers can also pin an exact tag or commit.
Consumers then [configure the library and selected groups](/guides/select-rules/).
Use `code-rules library check` in the library's CI.

Keep rule paths stable because paths define rule IDs.
When updating an adapted rule, compare its pinned original source with the newer revision before revising the adaptation.
Consumers update from your maintained library; they do not automatically track the adaptation's original source.

## Add supporting assets when needed

Keep optional rule-specific material in `assets/<rule-name>/` beside the rule file.
For example, `practices/testing/verify-retries.md` can link to `assets/verify-retries/example-response.json`.
Put shared material in the library-root `assets/` directory. Code Rules imports complete owned asset directories and includes shared assets when referenced.
See [Supporting assets](/reference/files/#supporting-assets) before adding images, explanations, or other files.
