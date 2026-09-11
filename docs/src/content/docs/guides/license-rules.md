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
- Are embedded code examples covered by the same terms, or separately licensed?
- Are there restrictions on publishing a competing rule library or service?

For a reusable best-practice library, we recommend expressly permitting teams to apply its guidance without requiring their application to adopt the library's license.
Copying protected example code or rule text into an application is a separate use that the chosen terms should address.

In the United States, copyright distinguishes ideas and methods from their original expression.
The [U.S. Copyright Office's explanation](https://www.copyright.gov/help/faq/faq-general.html) provides context for that distinction.
It does not settle the interpretation of a particular license, contract, or copied example.
Have the chosen terms reviewed for the rights and restrictions you intend before publishing them.

## Declare the library's terms

Keep the actual license text in `LICENSE.md` at the library root, or another explicitly named file.
The proposed library metadata identifies the default terms and any accompanying notice files:

```json
{
  "formatVersion": 1,
  "license": {
    "file": "LICENSE.md",
    "notices": []
  }
}
```

Paths resolve from the library root.
For example, list `NOTICE.md` in `notices` when the library supplies a notice that must accompany imported content.
No extra notice file is required when the license and rule files already contain the necessary notices.
The [file reference](/reference/files/#library-license-metadata) defines these fields.

Keep attribution with each rule, in its metadata or Markdown body.
If particular rules or examples have different terms, state them explicitly and reference the applicable license files.
The library's default declaration must not conceal those differences or override rights held by another author.

A source citation is not a license grant.
Before adapting material from a book, article, or another library, establish the rights needed for the intended use.

## What imports preserve

The importer copies the license and declared notice files from the same resolved commit as the rules:

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
    practices/testing.md
    rules/fabrica/practices/testing/verify-retry-limits.md
```

Preserve copyright notices and per-rule attribution in both the vendored source and the generated rule file.
Keep referenced rule-specific license files with the snapshot and repair their relative links after relocation.
Include the preserved files in snapshot digests so offline checks can detect missing or changed files.
License changes belong in the update report alongside rule changes.

Each individual effective rule identifies its source and applicable preserved license.
For example, `generated/rules/fabrica/practices/testing/verify-retry-limits.md` begins with the following header.
The repository and commit in this example are illustrative.

```markdown
# Verify retry limits

Rule ID: `fabrica:practices/testing/verify-retry-limits`

[Active definition](https://github.com/example/rules/blob/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/practices/testing/verify-retry-limits.md)

Library default license and notices:
- [LICENSE.md](../../../../../vendor/fabrica/LICENSE.md)
```

The rule's metadata and body follow this header, retaining its original copyright notices and attribution.
Generated rule files link to their applicable preserved terms.
The index links to provenance, which records source libraries and their declared license paths.
When a group includes multiple libraries, retain each rule's licensing information instead of assigning one upstream license to the group.
Copied rule text remains subject to its applicable terms even when the project changes its wording or replaces an imported rule with an adaptation.
Record the applicable terms for the local definition; an ID alone cannot establish its licensing.

## What a consuming project should understand

The importer leaves the consuming repository's root license unchanged.
Imported content remains separately identified under its applicable terms.
The index should explain that boundary so the root license does not appear to relicense imported rules.
Whether a particular combination or redistribution is permitted depends on the actual licenses and use.

Keep the vendored license and notice files when sharing generated rule files.
Copying a rule file by itself can break its license links and omit required notices.

If licensing is undeclared or ambiguous, report it as unspecified and resolve the permissions before redistribution.
Do not infer a license from the repository's visibility, its source name, or another library's terms.
Private libraries can document internal permissions without adopting a public license.

## What the tool can check

Code Rules can validate declared paths, preserve texts and notices, track their provenance, and check that generated links resolve.
It cannot establish ownership, decide legal compatibility, or certify that the chosen terms permit a consumer's intended use.
The preservation behavior described here is part of the proposed importer design.
