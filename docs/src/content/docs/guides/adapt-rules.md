---
title: "Adapt a third-party rule"
description: "Turn external guidance into a maintained Code Rules definition with attribution and retained terms."
---

Code Rules imports libraries that follow its format.
For guidance from another source, first author a compatible adaptation locally or in a shared library.
An agent can help draft it; automatic conversion of arbitrary repositories is outside the import workflow.

This walkthrough follows the repository's existing ESLint Unicorn adaptation fixture.
It uses the `no-for-each` documentation at commit `5d9d745c5365b6fdb824db1122ff982dd824b11a`.
The fixture retains the source's MIT declaration, license text, and an adaptation notice.

## 1. Identify the material and its terms

Select the specific rule or document and record its repository, file path, and exact revision.
Read the source and applicable license files before adapting it.
Follow [License rules](/guides/license-rules/) to record the terms for your intended use.
Code Rules preserves those declarations; it does not establish permission for you.

## 2. Choose who maintains the adaptation

For a rule used by one codebase, author it under that project's `code-rules/local/` directory.
For reuse across projects, [create a compatible library](/guides/create-library/) and maintain the adaptation there.
The original repository does not need to change.

Our local example has this layout:

```text
code-rules/
  config.json
  local/
    techs/javascript/
      _group.json
      prefer-for-of.md
    licenses/unicorn/
      LICENSE.md
      NOTICE.md
```

If JavaScript is a local-only group, select it through `localGroups` in the project configuration:

```json
{
  "schemaVersion": 1,
  "sources": {},
  "localGroups": ["techs/javascript"]
}
```

If the project already imports that group, add the local rule to it without redefining its group metadata.
See [Configuration](/reference/configuration/) for combining imported and local groups.

## 3. Draft the definition

Use the [canonical template](/reference/rule-authoring/#markdown-template) and required frontmatter.
In this example, `whenToRead` identifies work on array iteration using `forEach`.
The body recommends `for-of` when it preserves the intended behavior.
Implementation and validation guidance explain how to assess that condition.

Compare the draft with the original obligation and exceptions.
Explain deliberate changes: this adaptation adds task guidance and a sparse-array exception.
A detector's inability to analyze some code does not automatically make that code exempt from the written rule.

You can give an agent this instruction:

> Read the pinned source, retained terms, and Code Rules authoring rubric. Draft one compatible rule in the chosen group. Preserve its meaning and relevant exceptions, identify deliberate changes, and record attribution and license files. Present unresolved policy or licensing questions for review.

## 4. Record the source and retained terms

Include the following fields alongside the rule's required metadata:

```yaml
licenses:
  - expression: MIT
    files: [licenses/unicorn/LICENSE.md]
    attributionFiles: [licenses/unicorn/NOTICE.md]
attribution:
  - url: https://github.com/sindresorhus/eslint-plugin-unicorn/blob/5d9d745c5365b6fdb824db1122ff982dd824b11a/docs/rules/no-for-each.md
    description: Adapted from ESLint Unicorn; added task guidance and a sparse-array exception.
```

Paths start at `local/` for a local definition, or at the library root for a published definition.
Preserve the actual referenced license and notice files with the rule.
The identifier supplements those files.
Do not place an edited adaptation in `vendor/` and label it an unchanged upstream snapshot.

## 5. Review and generate

Review the completed definition against the authoring rubric and its source.
For a shared library, the planned `code-rules library check` validates its input format.
In a consuming project, the planned `code-rules build` generates effective rules from local and vendored inputs.

The offline generator already demonstrates this flow in the Code Rules development checkout:

```sh
bun install --frozen-lockfile
GENERATED_FILES="$PWD/.runbook-output/adaptation-guide" bun tests/manual/applicability-walkthrough.ts licenses
```

Open `.runbook-output/adaptation-guide/05-licensed-rule/`, then inspect `generated/provenance.json` and the generated JavaScript rule.
Confirm that the effective origin is local, the declared license is `MIT`, and the attribution identifies the pinned original document.
Follow the generated license and notice links to the retained files.
The fixture uses committed material and does not download or execute the upstream project.

## 6. Maintain the adaptation

Commit the adaptation, retained terms, and project configuration together.
When upstream changes, compare the old and new source and deliberately revise the adaptation.
Changing a citation alone does not establish that the rule incorporates the updated guidance.

For adaptations in a shared library, the maintainer reviews upstream changes once.
Consuming projects then import a new version of that compatible library through the normal update workflow.
