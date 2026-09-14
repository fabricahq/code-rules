---
title: "Adapt a third-party rule"
description: "Package external guidance in a compatible library with one library-wide license and preserved attribution."
---

Code Rules imports libraries that follow its format.
For guidance from another source, first create a compatible adaptation in a library, even if that library contains only one rule.
An agent can help author it; automatic conversion of arbitrary repositories is outside the import workflow.
Use local rules for guidance you author for your project.

This walkthrough uses an ESLint Unicorn adaptation based on commit `5d9d745c5365b6fdb824db1122ff982dd824b11a`.
The fixture retains the source's MIT license text and an adaptation notice.

## 1. Identify the material and its terms

Select the specific document and record its repository, file path, and exact revision.
Read the source and applicable license files before adapting it.
Follow [License rules](/guides/license-rules/) to record the terms for your intended use.
Code Rules preserves declarations; it does not establish permission for you.

## 2. Create a compatible library

[Create a library](/guides/create-library/) to maintain the adaptation. The original repository does not need to change.
The library uses this layout:

```text
adapted-rules/
  rule-library.json
  LICENSE.md
  NOTICE.md
  techs/javascript/
    _group.json
    prefer-for-of.md
```

Declare one license covering the whole library in `rule-library.json`:

```json
{
  "formatVersion": 1,
  "license": {
    "spdxExpression": "MIT",
    "file": "LICENSE.md",
    "notices": ["NOTICE.md"]
  }
}
```

Keep the complete license text and required notices at those paths.
Rules and groups cannot override this declaration. Material requiring another declaration belongs in a separate library.

## 3. Draft the definition and record attribution

Use the [canonical template](/reference/rule-authoring/#markdown-template) and required frontmatter.
In this example, `whenToRead` identifies work on array iteration using `forEach`.
The body recommends `for-of` when it preserves the intended behavior.
Implementation and validation guidance explain how to assess that condition.

Compare the draft with the original obligation and exceptions.
Explain deliberate changes: this adaptation adds task guidance and a sparse-array exception.
A detector's analysis limitations do not automatically become exceptions to the written rule.

Record the original source alongside the rule's required metadata:

```yaml
attribution:
  - url: https://github.com/sindresorhus/eslint-plugin-unicorn/blob/5d9d745c5365b6fdb824db1122ff982dd824b11a/docs/rules/no-for-each.md
    description: Adapted from ESLint Unicorn; added task guidance and a sparse-array exception.
```

Attribution identifies the source; the library manifest owns the license declaration.
Do not put edited material in `vendor/` and label it an unchanged upstream snapshot.

## 4. Review, import, and generate

Review the definition against the authoring rubric and the original source.
The planned `code-rules library check` validates the library format.
Publish the compatible library to a Git repository and select it through the consuming project's `sources` configuration.
The proposed sync workflow imports its selected groups; the builder then generates effective rules and license links.
See [Configuration](/reference/configuration/) for selecting a library and groups.

The offline example is available in the Code Rules development checkout:

```sh
bun install --frozen-lockfile
GENERATED_FILES="$PWD/.runbook-output/adaptation-guide" bun tests/manual/applicability-walkthrough.ts library-licenses
```

Open `.runbook-output/adaptation-guide/05-licensed-library/README.md` and follow its file links.
Confirm that the generated rule records `licenseBasis: library`, the MIT declaration, and attribution to the original document.
Follow its license and notice links into `vendor/licensed/`.
The example supplies a snapshot offline; the library repository is illustrative and no upstream project is downloaded or executed.

## 5. Maintain the adaptation

Commit the adapted rule, group metadata, manifest, and retained terms together in the library.
When upstream changes, compare the old and new source and deliberately revise the adaptation.
Changing a citation alone does not establish that the rule incorporates updated guidance.
The library maintainer reviews that change once; consuming projects then adopt its new version through the normal update workflow.
