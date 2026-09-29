---
title: "Rule versions"
description: "What a rule version covers, tag and library release formats, change levels, change notes, and retired rules."
---

Each rule in a library has its own [semantic version](https://semver.org/). A rule's version is a Git tag, not a field in the rule file, so frontmatter has no `version` field. A **library release** publishes new versions of one or more rules at once, from a single rule change to a large collection of updates.

This page is the exact specification. [Version your rules](/guides/version-rules/) explains the concepts and walks through the workflow.

## What a version covers

A rule version covers everything an agent needs to read the rule as its author intended:

- The rule's Markdown file.
- The rule's own asset directory, `assets/<rule-name>/`.
- Every file in the library's shared `assets/` directory that the rule or its assets link to, including files those shared files link to in turn.

A project that imports a rule version gets exactly these files as they were in that version. A change to a shared file is a change to every rule that links to it, so each of those rules needs a [change note](#change-notes).

Group metadata, the library's license declaration, and its license and notice files aren't part of any rule version. Projects receive them from the newest library release they import, or from the revision their `ref` names.

## Tag format

Library releases create three kinds of annotated tags. Code Rules ignores other tags.

| Tag | Example | Target | Message |
| --- | --- | --- | --- |
| Rule version | `practices/testing/verify-retry-limits@1.3.0` | The release commit that published the version. | `<change>: <summary>`, where `<change>` is `new`, `major`, `minor`, or `patch`. When several changes were combined, each further summary line follows on its own line. |
| Retired rule | `practices/testing/check-retry-backoff@retired` | The release commit where the rule's file no longer exists. | `retired: <summary>`, with a `Replaced-by` trailer when a rule replaces it; see [Retired rules](#retired-rules). |
| Library release | `release/2` | The release commit. | A list of the library release's changes. |

In a rule version tag, the rule ID is the rule's path without `.md`, and the version is plain `major.minor.patch` numbers, without prerelease or build suffixes. A new rule starts at `1.0.0`.

The release commands create these tags. Don't create, move, or delete them by hand.

Git can't store a tag whose name matches a directory of other tags, so no tag may be named like a group or folder that contains rules, such as `practices/testing`. `code-rules library check` reports such tags.

## Library releases

A **library release** publishes every pending change note as new rule versions. All of its tags point to one **release commit**: the first commit on the default branch that contains the library release's manifest. If the release pull request is merged with a merge commit, that's the merge commit; with a squash or rebase, it's the commit the pull request became. The release commit:

- deletes the library release's change notes, so it never contains changes waiting for a library release, and
- updates the release manifest to list every rule's version.

Every rule in a release commit is exactly one of its published versions.

### Release manifest

The **release manifest**, `code-rules-release.yaml` at the library root, records what each library release published. Only the release commands write it; don't edit it by hand.

```yaml
# Written by code-rules library release. Don't edit.
release: 2
rules:
  practices/code-design/express-operations-as-meaningful-steps: 1.0.0
  practices/code-design/organize-code-by-feature: 1.1.0
  practices/testing/verify-retries: 1.0.0
  practices/testing/verify-retry-limits: 2.0.0
retired:
  practices/testing/check-retry-backoff:
    replacedBy: practices/testing/verify-retries
```

| Field | Meaning |
| --- | --- |
| `release` | The release number. The first library release is `1`, and each library release adds one. Release numbers aren't semantic versions: a library release can hold changes of every size to different rules, and each rule's version describes its own change. |
| `rules` | Every current rule's ID and version. |
| `retired` | Every retired rule's ID, with its `replacedBy` rule when there is one, kept permanently so retired IDs aren't reused. A rule retired without a replacement maps to `{}`. |

The manifest uses one YAML document. Duplicate keys, anchors, aliases, explicit tags, and unknown fields are rejected.

The first library release creates the manifest. Each later library release rewrites it, so the library release's diff shows the exact new version of every changed rule.

### Release tags and GitHub Release pages

Each library release creates one `release/<number>` tag, such as `release/2`. For a repository on GitHub.com, it also creates one **GitHub Release page** on that tag: the page GitHub uses to announce a library release, which people can browse and get notified about. It's an announcement only; Code Rules never reads it. In these docs, a *library release* is a publication of a library's rule versions, as described on this page, and a *GitHub Release page* is only its announcement on GitHub. The page lists the library release's changes, grouped as major, minor, patch, new, and retired, with each rule's old and new version and its summary.

The newest library release is the one with the highest release number. A project normally chooses versions rule by rule, but it can also import exactly what one library release published, by naming its release tag; see [Import one revision](/reference/configuration/#import-one-revision).

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

A change to a shared file follows the same test for each rule that links to it. A clarified diagram is usually a patch; a shared example that now shows a stricter practice may be major for the rules that rely on it.

### Write the summary

Write each change's summary for a project maintainer deciding whether to update. Name what changed in the obligation or guidance, such as "Require a test at the limit for every retry policy." Avoid describing the edit itself, such as "Update the rule."

## Change notes

After the first library release, every change to a rule needs a **change note** until the change is published in a library release. The note records the size of the change and a summary for projects that update. A library release deletes the note and turns it into the rule's next version tag.

Before the first library release, no rule has a version, so rules need no notes. The first library release gives every rule version `1.0.0`.

A change note is `<rule-name>.change.yaml` in the same directory as `<rule-name>.md`. For example, the note for `practices/testing/verify-retry-limits.md` is `practices/testing/verify-retry-limits.change.yaml`. Notes use YAML because every Markdown file in a group is a rule.

```yaml
bump: patch
summary: Clarify the incorrect example.
```

| Field | Meaning |
| --- | --- |
| `bump` | `major`, `minor`, or `patch`. Required for a changed rule. Omit it for a new rule, which is published as `1.0.0`, and for a retired rule. |
| `summary` | Required non-blank text. Each line describes one change. Several lines appear when changes were combined. For a retirement, explain why. |
| `retired` | Optional `true`. Records that the rule was [retired](#retired-rules), so the note exists while its Markdown file does not. |
| `replacedBy` | Optional, only with `retired: true`: the ID of the rule that replaces the retired one. The replacement must exist in the library when the retirement is published. |

Change notes use one YAML document. Duplicate keys, anchors, aliases, explicit tags, and unknown fields are rejected.

A new rule's note omits `bump`:

```yaml
summary: Add the rule.
```

A retired rule's note keeps its name after the Markdown file is deleted. When a better rule replaces it, name the replacement:

```yaml
retired: true
replacedBy: practices/testing/verify-retries
summary: Covered by the broader rule about testing retries.
```

When nothing replaces it, the summary explains why the practice is no longer recommended:

```yaml
retired: true
summary: Withdrawn after feedback that agents shouldn't add explanatory comments.
```

If a rule changes again before a library release, update its existing note rather than adding another. Keep the larger `bump` and add a summary line. `code-rules library change` does this for you:

```yaml
bump: minor
summary: |
  Clarify the incorrect example.
  Add a Python example.
```

`code-rules library check` requires a valid note for each changed rule, including rules that link to a changed shared file, and rejects notes that no longer match a change. See [library check](/reference/cli/#library-check) for the complete list. The [Version your rules](/guides/version-rules/) guide shows the workflow.

## Retired rules

**Retiring** a rule ends its history: the library stops publishing it. Retire a rule when a better rule replaces it, such as a narrow retry-limit rule folded into a broader rule about testing retries, or when the practice is no longer recommended, such as a rule about how agents should comment code after the author concludes they shouldn't add those comments at all. The retirement names its replacement when there is one; its summary explains why.

To retire a rule, delete its Markdown file and asset directory and add a [change note](#change-notes) with `retired: true`. The library release creates an annotated `<rule-id>@retired` tag on the release commit where the file no longer exists, and lists the rule under `retired` in the manifest. The tag message starts with `retired:` and the summary. When a rule replaces it, the message ends with a `Replaced-by` [trailer](https://git-scm.com/docs/git-interpret-trailers):

```text
retired: Covered by the broader rule about testing retries.

Replaced-by: practices/testing/verify-retries
```

The rule's last numbered version stays its final version. A project that pinned the rule before the retirement keeps importing that version. A retired rule's ID can't be reused; give a new rule a new ID.

Projects that update past a retirement see it in the update preview, with its summary and any replacement.

Renaming or moving a rule changes its ID. Record it as retiring the old ID, replaced by the new ID, which starts at `1.0.0`.
