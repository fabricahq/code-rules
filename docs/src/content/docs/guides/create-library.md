---
title: "Create a rule library"
description: "Initialize a library, add groups and rules, validate its files, and share it through Git."
---

A rule library lets several projects adopt and update the same engineering guidance.
Create one in a new or existing Git repository, then add groups and rules using the shared authoring template.
For a rule used by only one project, you can instead [author it locally](/guides/write-rules/).

:::note[Planned commands]
The `code-rules library` commands are planned for the final implementation phase and are not available yet.
You can author the same files manually using the [format reference](/reference/files/#library-layout) and [rule template](/reference/rule-authoring/#markdown-template).
:::

## 1. Initialize the library

From the repository root, run the proposed command:

```sh
code-rules library init
```

The command creates `rule-library.json` and a short README linking to the canonical authoring guidance.
Choose one license covering the whole library, including its rules and examples, or supply the declaration and file for existing terms.
Rule and group license overrides are unsupported. Use separate libraries for material that requires different declarations.
Code Rules records that choice and retains the actual license text and any notices.
Code Rules does not select a license for you or infer one from existing text.
If you defer the choice, the manifest leaves the license undeclared and the command reminds you to decide the terms before sharing.

Initialization preserves existing files and reports collisions before writing anything.
The command does not create a remote repository, commit files, or publish content.

## 2. Add a group

```sh
code-rules library add group techs/javascript
```

Provide the group's name, description, and `whenToRead` cues.
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
If the group is missing, it shows the `library add group` command needed to create it.
Fill in the required metadata and draft the obligation, conditions, and exceptions.
Add implementation and validation sections when they provide useful guidance.
Tags are optional.

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
    prefer-for-of.md
```

## 4. Validate and review

```sh
code-rules library check
```

The command checks the format, group and rule metadata, paths, and declared license and notice files without changing them.
It reports incomplete required fields with their file paths.
An empty, well-formed group is valid; the output reports group and rule counts so authors can see what they have created.
Missing license declarations produce a visible warning, consistent with the format's support for undeclared terms.

A successful check establishes that the library meets the input format.
Review the guidance and license declarations separately; the check cannot establish their accuracy or legal sufficiency.

## 5. Share through Git

Review and commit the source rules, metadata, and retained terms together.
Publish the repository through your usual Git workflow and publish complete semantic version tags, such as `v1.0.0`, so consumers can use version constraints. Consumers can also pin an exact tag or commit.
Consumers then [configure the library and selected groups](/guides/select-rules/).
Use `code-rules library check` in the library's CI once the command is available.

Keep rule paths stable because paths define rule IDs.
When updating an adapted rule, compare its pinned original source with the newer revision before revising the adaptation.
Consumers update from your maintained library; they do not automatically track the adaptation's original source.

## Add supporting assets when needed

Keep optional rule-specific material in `assets/<rule-name>/` beside the rule file.
For example, `practices/testing/verify-retries.md` can link to `assets/verify-retries/example-response.json`.
Put shared material in the library-root `assets/` directory. Code Rules imports complete owned asset directories and includes shared assets when referenced.
See [Supporting assets](/reference/files/#supporting-assets) before adding images, explanations, or other files.
