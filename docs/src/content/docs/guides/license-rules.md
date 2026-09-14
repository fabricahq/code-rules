---
title: "License rules"
description: "Choose clear terms for rule content and preserve them when projects import and aggregate rules."
---

A rule library should explain what consumers may do with its text, examples, and adaptations.
Code Rules preserves those terms through imports; it does not choose a license for the publisher or grant additional rights.
No particular license, including a Fair Source license, is required by the format.

## Decide what the license covers

Distinguish the rule content from application code written using its guidance.
A library's terms should answer these questions:

- May teams read the rules with agents and use the guidance in commercial projects?
- May they copy rule files into private or public repositories?
- May they combine rules into aggregates, modify them, and redistribute the result?
- What attribution, copyright notices, and license terms must accompany those copies?
- Does the library-wide declaration cover all included rules and code examples?
- Are there restrictions on publishing a competing rule library or service?

For a reusable best-practice library, we recommend expressly permitting teams to apply its guidance without requiring their application to adopt the library's license.
Copying protected example code or rule text into an application is a separate use that the chosen terms should address.

In the United States, copyright distinguishes ideas and methods from their original expression.
The [U.S. Copyright Office's explanation](https://www.copyright.gov/help/faq/faq-general.html) provides context for that distinction.
It does not settle the interpretation of a particular license, contract, or copied example.
Have the chosen terms reviewed for the rights and restrictions you intend before publishing them.

## Declare the library's terms

Keep the actual license text in `LICENSE.md` at the library root, or another explicitly named file.
The library metadata identifies the terms for the whole library and any accompanying notice files:

```json
{
  "formatVersion": 1,
  "license": {
    "spdxExpression": "MIT",
    "file": "LICENSE.md",
    "notices": []
  }
}
```

Paths resolve from the library root and identify source files only. Code Rules copies the contents unchanged to `generated/libraries/<source-name>/licenses/LICENSE.md` and `generated/libraries/<source-name>/licenses/notices/001.md`, `002.md`, and so on. Notice numbering follows unique declaration order; output destinations are not configurable.
For example, list `NOTICE.md` in `notices` when the library supplies a notice that must accompany imported content.
No extra notice file is required when the license and rule files already contain the necessary notices.
The [file reference](/reference/files/#library-license-metadata) defines these fields.

Use an SPDX expression for standard terms and a distinct `LicenseRef-…` for custom or modified terms. The [SPDX expression specification](https://spdx.github.io/spdx-spec/v2.3/SPDX-license-expressions/) defines identifiers and compound expressions. The identifier supplements the retained text. Code Rules validates the `license.spdxExpression` syntax and identifiers and preserves your declaration. It does not verify that the declaration matches the retained terms or grants the permissions you need.

A library has one license declaration covering all its rules and groups. Per-rule and per-group overrides are unsupported.
Keep attribution with each rule, in its body or [structured attribution](/reference/files/#rule-attribution). List required notices in the library manifest.
If material requires a different declaration, maintain it in a separate compatible library. Do not relabel third-party material merely to fit a library's license.

A source citation is not a license grant.
Before adapting material from a book, article, or another library, establish the rights needed for the intended use.

## What imports preserve

The planned import workflow copies license and declared notice files from the same resolved commit as the rules.
The offline builder accepts preassembled snapshots and checks that declared files are present.
It returns unchanged license and notice copies at generated paths, but does not fetch source files or write them to disk.
The consuming workspace will retain them alongside the imported rules:

```text
code-rules/
  vendor/
    fabrica/
      LICENSE.md
      rule-library.json
      _source.json
      practices/testing/...
  generated/
    RULES.md
    provenance.json
    libraries/fabrica/
      README.md
      licenses/LICENSE.md
    groups/practices/testing.md
    rules/fabrica/practices/testing/verify-retry-limits.md
```

Preserve copyright notices and per-rule attribution in both the vendored source and the generated rule file.
Keep the library license and declared notices with the snapshot and preserve their links after relocation.
The planned workspace checks will verify snapshot digests to detect changed files. The offline builder checks declared file presence without computing digests.
The planned update report should include license changes alongside rule changes.

Each individual effective rule identifies its source and applicable preserved license.
For example, `generated/rules/fabrica/practices/testing/verify-retry-limits.md` includes this source footer after its guidance.
The repository and commit in this example are illustrative.

```markdown
## Source and attribution

**Rule source:** [Original rule](https://github.com/example/rules/blob/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/practices/testing/verify-retry-limits.md)

Library license and notices:

**Declared license:** MIT

- [LICENSE.md](../../../../libraries/fabrica/licenses/LICENSE.md)
```

Guidance precedes this footer; preserved source metadata follows it. Every imported rule displays its source library's declaration; explicit attribution citations also appear in the footer.
Generated rule files link to these standardized license and notice copies. Provenance records both original source paths and generated paths.
The index links to provenance, which records source libraries, declared license expressions, retained files, and explicit attribution.
When a group includes multiple libraries, retain each rule's licensing information instead of assigning one upstream license to the group.
Copied rule text remains subject to its applicable terms even when the project changes its wording or replaces an imported rule with an adaptation.
Maintain third-party adaptations in a compatible library with the applicable library-wide declaration. Local rules and replacements are for original project guidance; an ID alone cannot establish licensing.

## What a consuming project should understand

The importer leaves the consuming repository's root license unchanged.
Imported content remains separately identified under its applicable terms.
The index should explain that boundary so the root license does not appear to relicense imported rules.
Whether a particular combination or redistribution is permitted depends on the actual licenses and use.

When sharing generated rules, include their library folders and retained license and notice copies.
Preserve the vendored sources when sharing the complete consuming workspace.
Copying a rule file by itself can break its license links and omit required notices.

If licensing is undeclared or ambiguous, report it as unspecified and resolve the permissions before redistribution.
Do not infer a license from the repository's visibility, its source name, or another library's terms.
Private libraries can document internal permissions without adopting a public license.

## What the tool can check

The builder validates declared paths, preserves text and notices, records provenance, and constructs links to the retained files.
It rejects unsafe or unresolved local Markdown references, but does not check whether external URLs are reachable.
It cannot establish ownership, decide legal compatibility, or certify that the chosen terms permit a consumer's intended use.
The offline generator validates declarations and emits this provenance today. Automated downloading, snapshot installation, and update reporting remain part of the proposed importer design.


For source material that does not use this format, follow [Adapt third-party rules](/guides/write-rules/#adapt-third-party-rules). Keep a separate adapted definition with its source citation and retained terms; do not relabel edited content as an unchanged upstream snapshot.
