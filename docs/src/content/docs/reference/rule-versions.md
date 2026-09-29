---
title: "Rule versions"
description: "Tag format, release commits, change levels, change notes, and retired rules for library rules."
---

Each rule in a library has its own [semantic version](https://semver.org/). A rule's version is a Git tag, not a field in the rule file, so frontmatter has no `version` field.

A version covers the rule's Markdown file and its owned asset directory. Group metadata, shared assets, and license files are not versioned; projects receive changes to them with the next release.

This page is the exact specification. [Version your rules](/guides/version-rules/) explains the concepts and walks through the workflow.

## Tag format

| Part | Format |
| --- | --- |
| Name | `<rule-id>@<major>.<minor>.<patch>`, such as `practices/testing/verify-retry-limits@1.3.0`. The rule ID is the rule's path without `.md`. A retired rule also has one `<rule-id>@retired` tag; see [Retired rules](#retired-rules). |
| Version | Plain `major.minor.patch` numbers. Prerelease and build suffixes are not used. A new rule starts at `1.0.0`. |
| Kind | Annotated tag. |
| Target | The release commit that published the version. |
| Message | `<change>: <summary>`, where `<change>` is `new`, `major`, `minor`, or `patch`, or for a retirement, `superseded` or `withdrawn`. When several changes were combined, each further summary line follows on its own line. |

The release commands create these tags. Don't create, move, or delete them by hand.

Code Rules ignores tags that don't start with `techs/` or `practices/`. Git cannot store a tag whose name matches a directory of other tags, so no tag may be named like a group or folder that contains rules, such as `practices/testing`. `code-rules library check` reports such tags.

## Release commits

A **release commit** is the commit where a release tags its new rule versions. Each release tags all of its versions on a single commit.

Releasing removes the change notes, so a release commit never contains changes waiting to be released. Every rule in it is exactly one of its published versions. Projects import only release commits, so they never receive unreleased changes.

`code-rules library release` also keeps a branch named `code-rules/released` pointed at the newest release commit. Projects that follow the library's releases use this branch to find the newest one. Don't push to this branch yourself, and don't add branch rules that stop the release workflow from updating it.

## Choose a version change

Rule versions describe the rule's **obligation**: what work must do to comply with it.

| Change | Definition | Example |
| --- | --- | --- |
| `major` | Work that complied with the previous version could fail this one. | Lower the required retry limit, or require a test the rule previously only recommended. |
| `minor` | Work that complied with the previous version still complies, and this version adds new guidance. | Add a Python example of the same test. |
| `patch` | Work that complied with the previous version still complies, and this version adds no new guidance. | Fix a misleading sentence or a typo in an example. |

To decide, ask two questions in order. First, could work that complied with the previous version fail this one? If yes, the change is major. If no, does this version add new guidance, such as a new example or advice for a new situation? If yes, the change is minor; if no, it's a patch.

These edits show how the definitions apply:

| Edit | Change |
| --- | --- |
| Narrow an exception, lower a limit, or turn a recommendation into a requirement. | `major` |
| Replace the obligation with a different one, even when it seems equally strict. | `major` |
| Add an example in another language, or advice for a situation the rule already allowed. | `minor` |
| Reword a sentence, fix a typo, or correct an example that contradicted the rule's own text. | `patch` |

Widening where a rule applies can be major, even though it only adds text: code in the newly covered situation may not comply. Adding a new case is minor only when all work that complied with the previous version still complies. When unsure, choose the larger change.

### Write the summary

Write each change's summary for a project maintainer deciding whether to update. Name what changed in the obligation or guidance, such as "Require a test at the limit for every retry policy." Avoid describing the edit itself, such as "Update the rule."

## Change notes

After a library's first release, every change to a rule needs a **change note** until the change is released. The note records the size of the change and a summary for projects that update. Releasing deletes the note and turns it into the rule's next version tag.

Before the first release, no rule has a version, so rules need no notes. The first `code-rules library release` publishes every rule at `1.0.0`.

A change note is `<rule-name>.change.yaml` in the same directory as `<rule-name>.md`. For example, the note for `practices/testing/verify-retry-limits.md` is `practices/testing/verify-retry-limits.change.yaml`. Notes use YAML because every Markdown file in a group is a rule.

```yaml
bump: patch
summary: Clarify the incorrect example.
```

| Field | Meaning |
| --- | --- |
| `bump` | `major`, `minor`, or `patch`. Required for a changed rule. Omit it for a new rule, which is released as `1.0.0`, and for a retired rule. |
| `summary` | Required non-blank text. Each line describes one change. Several lines appear when changes were combined. For a retirement, explain why. |
| `retired` | Optional `superseded` or `withdrawn`. Records that the rule was [retired](#retired-rules), so the note exists while its Markdown file does not. |
| `replacedBy` | The ID of the rule that replaces a `superseded` rule. Required for `superseded` and rejected otherwise. The replacement must exist in the library when the retirement is released. |

Change notes use one YAML document. Duplicate keys, anchors, aliases, explicit tags, and unknown fields are rejected.

A new rule's note omits `bump`:

```yaml
summary: Add the rule.
```

A retired rule's note keeps its name after the Markdown file is deleted. A superseded rule names its replacement:

```yaml
retired: superseded
replacedBy: practices/testing/verify-retries
summary: Covered by the broader rule about testing retries.
```

A withdrawn rule has no replacement, so its summary explains why the practice is no longer recommended:

```yaml
retired: withdrawn
summary: Withdrawn after feedback that agents shouldn't add explanatory comments.
```

If a rule changes again before a release, update its existing note rather than adding another. Keep the larger `bump` and add a summary line. `code-rules library change` does this for you:

```yaml
bump: minor
summary: |
  Clarify the incorrect example.
  Add a Python example.
```

`code-rules library check` requires a valid note for each changed rule and rejects notes that no longer match a change. See [library check](/reference/cli/#library-check) for the complete list. The [Version your rules](/guides/version-rules/) guide shows the workflow.

## Retired rules

**Retiring** a rule ends its history: the library stops publishing it. A rule is retired for one of two reasons:

| Reason | Meaning | Example |
| --- | --- | --- |
| `superseded` | A better rule replaces it. The retirement names the replacement, a rule in the same library. | A narrow retry-limit rule is folded into a broader rule about testing retries. |
| `withdrawn` | The practice is no longer recommended, because it turned out to be wrong or unhelpful. It has no replacement. | A rule about how agents should comment code, after the author concludes agents shouldn't add those comments at all. |

To retire a rule, delete its Markdown file and asset directory and add a [change note](#change-notes) with the reason. The release creates an annotated `<rule-id>@retired` tag on the release commit where the file no longer exists. The tag message starts with the reason and summary. A superseded rule's message ends with a `Replaced-by` [trailer](https://git-scm.com/docs/git-interpret-trailers):

```text
superseded: Covered by the broader rule about testing retries.

Replaced-by: practices/testing/verify-retries
```

The rule's last numbered version stays its final version. A retired rule's ID can't be reused; give a new rule a new ID.

Projects that update past a retirement see its reason, summary, and any replacement. A retirement of a rule the project uses requires the same consent as a major change.

Renaming or moving a rule changes its ID. Record it as retiring the old ID as `superseded`, replaced by the new ID, which starts at `1.0.0`.
