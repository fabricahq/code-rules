---
title: "Rule and group format"
description: "What Code Rules accepts in rule files, group metadata, and supporting assets."
---

This page defines what Code Rules accepts in a rule file and in group metadata. The same format applies to rules in a library and to a project's local rules. For how to write a rule well, see [Rule rubric and template](/reference/rule-authoring/).

Each rule is a Markdown file, and related rules belong to a **group**: a directory under `techs/`, `practices/`, or `areas/` that holds a `_group.yaml` file. Area groups are local only; see [Group types](/concepts/groups/#group-types). [Library format](/reference/library-format/) covers where groups live in a library.

Group metadata and rule frontmatter use YAML. Duplicate keys, anchors, aliases, and explicit tags are rejected.

## Group metadata

Library authors define each group's selection guidance in its `_group.yaml`, before the group is imported.
The builder reads that source metadata and renders its `whenToRead` cues in the generated root index; it does not infer them from the current rules.
Use the existing `whenToRead` field for this guidance rather than adding a separate `whenToUse` field.

```yaml
name: Testing
description: Verify behavior with meaningful tests.
whenToRead: When adding or changing behavior, fixing a bug, or modifying tests, even when
  no test files are in the diff.
```

All three group fields are required non-blank strings. Surrounding whitespace is trimmed. Unknown fields are rejected.
The directory supplies the group's ID and type.
Both `RULES.md` and the group page include its name, description, and when-to-read guidance. Local metadata takes precedence as a complete record; otherwise each contributing library remains source-labeled.
Follow the shared [whenToRead authoring guidance](/reference/rule-authoring/#write-whentoread-guidance-that-helps-selection) when filling in the group's reading cues.

## Rule metadata

Each rule requires YAML frontmatter containing `title`, `whenToRead`, `impact`, and `impactDescription`, followed by a non-blank Markdown body.
The builder rejects duplicate YAML keys and aliases. Unknown frontmatter fields are rejected. Accepted fields are `title`, `whenToRead`, `impact`, `impactDescription`, `tags`, and `attribution`.
`tags` is optional: use a non-blank string of comma-separated topics or an array of unique, non-blank strings. An empty array is also accepted.
Tags provide search terms, such as `testing` on a Go rule. Code Rules preserves supplied tags but does not use them for selection, grouping, or enforcement. Use `whenToRead` to describe applicability.
Use the existing impact vocabulary: `CRITICAL`, `HIGH`, `MEDIUM-HIGH`, `MEDIUM`, `LOW-MEDIUM`, or `LOW`.
Impact describes the significance of the consequence a rule helps prevent.
Follow the canonical [impact authoring guidance](/reference/rule-authoring/#describe-impact-through-consequences) for level definitions and examples.
Agents select rules using `whenToRead`, follow the full guidance and exceptions, and assess finding severity from concrete evidence.
All applicable rules matter regardless of impact.

`whenToRead` is a required non-empty string describing work that should trigger reading, including before code exists.
The group field of the same name is also one string, describing the group’s scope of applicability.
The full body defines obligations, implementation guidance, exceptions, and validation guidance.
Existing snapshots and local replacements must add rule-level `whenToRead` before building with this format.

See [Write a rule](/guides/write-rules/) for a complete example.

Rules have no `version` field, and you never set a version by hand. The first library release gives every rule version `1.0.0`, and later library releases assign versions from the change notes that authors record with `code-rules library change`; see [Change rules after the first library release](/guides/version-rules/#change-rules-after-the-first-library-release) and [Rule versions](/reference/rule-versions/).

## Rule attribution

A rule may record the source of an adaptation independently of the library's license:

```yaml
attribution:
  - url: https://github.com/sindresorhus/eslint-plugin-unicorn/blob/5d9d745c5365b6fdb824db1122ff982dd824b11a/docs/rules/no-for-each.md
    description: Adapted from Sindre Sorhus's ESLint Unicorn rule; added task guidance.
```

Each optional `attribution` entry requires an absolute HTTP(S) `url` without credentials and a non-blank `description`.
Use a commit-pinned URL and describe the adaptation. These citations identify source material; they do not override the library license.
Attribution in the Markdown body remains preserved. Unknown attribution-object fields are rejected.

Keep attribution with the rule in metadata or prose; a separate attribution file is not required by the format.
Declare accompanying license and notice files in the [library manifest](/reference/library-format/#license-metadata) so Code Rules can retain them and their links.

## Supporting assets

Use two optional locations for supporting files:

```text
assets/                              # Shared across the library
  retry-lifecycle.svg
practices/testing/
  _group.yaml
  verify-retries.md
  assets/
    verify-retries/                   # Owned by verify-retries.md
      explanation.md
      example-response.json
```

A rule owns `assets/<filename-without-.md>/` beside its Markdown file.
For example, `practices/testing/nested/retry.md` owns `practices/testing/nested/assets/retry/`.
Each owned directory must have an adjacent rule. The directory name `assets` is reserved and cannot contain active rules or group metadata.
Markdown within assets is supporting text and does not need rule frontmatter.

Keep the rule's obligations and exceptions in the rule itself. Use assets for explanations, images, sample data, and other supporting material.
Link from the rule with ordinary Markdown, such as `[Explanation](assets/verify-retries/explanation.md)`.
Use `../../assets/retry-lifecycle.svg` from this rule to reference a shared image.

Code Rules preserves each selected rule's complete asset directory, including files that are not individually linked.
From the library-root `assets/` directory, Code Rules imports only the files a selected rule or its Markdown assets link to, including files those shared files link to in turn. Shared files aren't part of a rule's [version](/reference/rule-versions/#what-a-version-covers): a project gets them from the library release that supplies its library-wide files, which `code-rules project update` moves to the newest library release. Keep a rule's obligations in the rule itself, and use shared files to explain and illustrate.
Other shared assets are omitted. Nested folders and binary files are allowed within these directories, subject to the import size and file-type limits.
No asset is executed. Markdown assets must be UTF-8 so their standard Markdown references can be checked.

A rule or its assets may link to its own assets, shared assets, and declared library license or notice files.
Shared assets may link to other shared assets and declared license files, but cannot depend on one rule's private assets.
A link into another rule's assets is invalid: move that material to the shared directory.
Missing local destinations and supporting references elsewhere in the repository fail import.
Filesystem links to another rule document are invalid, even when the target is selected or retained. This also applies to Markdown attachments, including HTML `href` and `src` attributes. Rules must remain independently selectable; move shared supporting explanations into `assets/`. Self-links and anchors within the same document remain valid. External URLs remain links and are not downloaded.

Files elsewhere in the repository can still hold project documentation or tooling, but cannot serve as local supporting dependencies for rules.
Relative links in raw HTML are rejected during generation. Use Markdown links, images, or reference definitions instead.

## Related pages

- [Rule rubric and template](/reference/rule-authoring/) explains how to write a rule well.
- [Write a rule](/guides/write-rules/) walks through writing an individual rule.
- [Library format](/reference/library-format/) covers a library's layout, manifest, and license.
