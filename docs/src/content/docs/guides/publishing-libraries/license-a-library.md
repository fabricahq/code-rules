---
title: "License a library"
description: "Declare the terms for sharing a rule library and check that imports retain them."
slug: guides/license-rules
---

A license states the terms for using, copying, modifying, and sharing rule content. Clear terms help projects know what they may do with your guidance and which notices they must keep. A rule gets its license from the library that contains it.

This guide shows how to record a library-wide declaration and check an imported copy. Code Rules keeps the declared text and notices and links generated rules back to them. Code Rules does not choose terms or grant permission to use someone else's work.

## Prerequisites

You need a Code Rules library with a `rule-library.yaml` file. Run the commands below from the library's root. To create a library first, follow [Create a library](/start-here/create-library/).

<span id="decide-what-the-license-covers"></span>

## 1. Decide which terms apply

Identify who wrote each rule, example, and adaptation in the library. Read the terms of any material you did not write before you copy or adapt it. A source link gives credit; it does not grant permission.

Choose terms that cover the uses you intend to allow. Consider reading rules with agents, applying their guidance, copying rule text, modifying it, and sharing copies or aggregates. Decide which copyright notices and attribution must travel with those copies. To let teams use the guidance without licensing their application code under the library's terms, make that boundary clear in your terms.

One declaration covers every group and rule. If included material needs different terms, put it in a separate compatible library. Public repository visibility alone does not supply a license.

<span id="declare-the-librarys-terms"></span>

## 2. Record the license and notices

Put the complete license text in `LICENSE.md` at the library root. If notices must accompany the rules, put them in `NOTICE.md`. Keep any required source attribution in the affected rule's body or [attribution metadata](/reference/rule-library-format/#rule-attribution).

Then declare those files in `rule-library.yaml`. For example, **if you choose MIT** and have an accompanying notice, use:

```yaml
formatVersion: 1
license:
  spdxExpression: MIT
  file: LICENSE.md
  notices:
    - NOTICE.md
```

When no separate notice applies, use `notices: []`. Do not relabel third-party text to match a chosen license. For other SPDX expressions, custom terms, and file-path rules, see [Library license metadata](/reference/rule-library-format/#library-license-metadata).

## 3. Check the library

From the library root, run:

```sh
code-rules library check
```

The check reports group and rule counts and verifies that declared files exist and the metadata is valid. Read `LICENSE.md`, any notices, and attributed rules before publishing. The check cannot establish ownership, permission, or whether the declaration matches the retained text.

<span id="what-imports-preserve"></span>
<span id="what-a-consuming-project-should-understand"></span>

## 4. Inspect an imported copy

After a project [imports the library](/start-here/create-library/#6-try-the-library-in-a-project), run these commands from that project's root:

```sh
code-rules project sync
code-rules project check
```

Open `.code-rules/generated/libraries/<source-name>/licenses/LICENSE.md` and any files under `licenses/notices/`. Open a generated rule and follow its source, license, notice, and attribution links. The project's root license remains unchanged; the imported rules keep their own declared terms.

If you share generated rules, include their library folders and retained license and notice files. A rule file copied alone may lose those links or omit a required notice. For the exact generated paths and provenance fields, see [Library license metadata](/reference/rule-library-format/#library-license-metadata) and [Provenance](/reference/provenance/).

<span id="what-the-tool-can-check"></span>

Code Rules checks the declared files and preserves them with imports. It cannot decide whether a license permits your intended use. To turn guidance from another source into a rule, follow [Import rules from another source](/guides/select-rules/#from-another-source).
