---
title: "Adapt a third-party rule"
description: "Package external guidance in a compatible library with one library-wide license and preserved attribution."
---

**Adapting a rule** means turning guidance from another source, such as a linter or style guide, into a Code Rules rule. You can reuse useful advice even when its author hasn't published a compatible library. The adapted rule needs to preserve the original meaning, attribution, and applicable terms.

In this guide, you'll adapt a JavaScript rule from ESLint Unicorn. You'll check the source and its terms, create a library for the adaptation, write the rule, and inspect the result in a consuming project.

Code Rules imports rules from compatible libraries; it does not automatically convert arbitrary documents. If the source already provides a compatible library, follow [Import rules](/guides/select-rules/) instead.

## 1. Identify the material and its terms

This example uses ESLint Unicorn at commit `5d9d745c5365b6fdb824db1122ff982dd824b11a`. The adapted library must retain the source's MIT license text and an adaptation notice.

Select the specific document and record its repository, file path, and exact revision.
Read the source and applicable license files before adapting it.
Follow [License rules](/guides/license-rules/) to record the terms for your intended use.
Code Rules preserves declarations; it does not establish permission for you.

## 2. Create a compatible library

[Create a library](/start-here/create-library/) to maintain the adaptation. The original repository does not need to change.
The library uses this layout:

```text
adapted-rules/
  rule-library.yaml
  LICENSE.md
  NOTICE.md
  techs/javascript/
    _group.yaml
    prefer-for-of.md
```

Declare one license covering the whole library in `rule-library.yaml`:

```yaml
formatVersion: 1
license:
  spdxExpression: MIT
  file: LICENSE.md
  notices:
    - NOTICE.md
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

Compare the definition with the original source. Use the [authoring rubric](/reference/rule-authoring/) to look for useful writing improvements.
`code-rules library check` validates the library format.
Publish the compatible library to a Git repository and select it through the consuming project's `sources` configuration.
The Imports API fetches the selected groups; the builder then generates resolved rules and license links. Sync applies those files to the consuming project.
See [Configuration](/reference/configuration/) for selecting a library and groups.

After generating the files, confirm that the rule displays the MIT declaration and attribution to the original document.
Its entry in `generated/provenance.json` should record `licenseBasis: "library"`.
Open `generated/libraries/<source-name>/README.md` for the library summary.
Follow the generated rule’s license and notice links into `generated/libraries/<source-name>/licenses/`.
The copies preserve the original text; provenance maps their source and generated paths.

## 5. Maintain the adaptation

Commit the adapted rule, group metadata, manifest, and retained terms together in the library.
When upstream changes, compare the old and new source and deliberately revise the adaptation.
Changing a citation alone does not establish that the rule incorporates updated guidance.
The library maintainer reviews that change once; consuming projects then adopt its new version through the normal update workflow.
