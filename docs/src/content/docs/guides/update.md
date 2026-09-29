---
title: "Update rules"
description: "Review each rule change, accept major changes deliberately, and preserve local decisions."
---

Updating rules brings changes from the libraries you use into your project. Library authors may improve advice, fix mistakes, add rules, or make rules stricter. Your project adopts those changes only when you run `code-rules project update`.

In this guide, you'll update your libraries, review each rule change, accept major changes or keep an older version of a rule, and commit the result. You'll also learn how to change your selection of groups and recover from a failed update.

Start with a project that already [imports rules](/guides/select-rules/). For details about how commands change files, see [Sync and recovery](/reference/sync/).

## Sync and update

Two commands import library rules, and they do different things:

- **`code-rules project sync`** imports the commits already recorded in `.code-rules/vendor/`. Everyone who syncs the project gets the same rules. It resolves a revision only for a new source, or after you change a source's repository or `ref`.
- **`code-rules project update`** moves sources to the newest release, reports every rule change, and asks you to accept major changes and retirements before it writes anything.

Neither command runs during ordinary coding, review, or `code-rules project check`, so rules never change underneath your agents.

## Update your libraries

From the project root, run:

```sh
code-rules project update
```

To update only some libraries, name their sources, such as `code-rules project update fabrica`.

`code-rules project update` moves each library to its newest release and reports each [rule version](/concepts/rule/#how-a-rule-is-versioned) that changed:

```text
fabrica  4f1c2a9 -> 9e07b3d
  major    practices/testing/verify-retry-limits           1.3.0 -> 2.0.0
           Require a test at the limit for every retry policy.
  minor    practices/code-design/organize-code-by-feature  1.0.0 -> 1.1.0
           Add a Go example.
  new      practices/testing/verify-retries                1.0.0
           Add the rule.
  retired  practices/testing/check-retry-backoff           1.2.0 superseded
           Replaced by practices/testing/verify-retries.
           Covered by the broader rule about testing retries.
  retired  practices/code-design/comment-intent            2.1.0 withdrawn
           Withdrawn after feedback that agents shouldn't add
           explanatory comments.

Nothing was updated: 1 major change and 2 retirements affect
rules this project uses. Review them, then run:
  code-rules project update --accept-major
```

Each line shows the change, the rule, and its old and new versions, followed by the summaries of every version in between. A retired rule shows its last version, the reason it was retired, and any replacement.

| Change | What it means for your project |
| --- | --- |
| `patch` | Work that complied with the previous version still complies. The rule adds no new guidance. |
| `minor` | Work that complied with the previous version still complies. The rule adds new guidance. |
| `major` | Work that complied with the previous version could fail this one. |
| `new` | A rule added to a group you import. |
| `retired` | The library stopped publishing the rule, so your agents will stop reading it. `superseded` means a named rule replaces it. `withdrawn` means the author no longer recommends the practice. |

When no rule your project uses has a major change or retirement, `code-rules project update` applies the new release right away. Otherwise it changes nothing and exits with status `1`.

## Accept major changes

For each major change and retirement, read the new rule, or the reason for the retirement, and decide what your project should do. Compare the old and new text in `.code-rules/vendor/<source-name>/` after updating, or in the library's GitHub Releases. Then choose one of these for each rule:

- **Adopt it.** Plan any work your code needs to follow the new obligation. For a superseded rule, read its replacement, and check that you import the replacement's group.
- **Keep the older version.** [Fork the rule](#keep-an-older-version-of-a-rule) before updating. This also keeps a retired rule you still want to follow.
- **Stop using it.** Add an [exclusion](/guides/select-rules/#exclude-a-rule) with your reason.

When you've decided to adopt every major change and retirement, accept them all:

```sh
code-rules project update --accept-major
```

To accept only some, name each rule you accept. Use the rule's full ID, including its source name, after `=`:

```sh
code-rules project update \
  --accept-major=fabrica:practices/testing/verify-retry-limits \
  --accept-major=fabrica:practices/code-design/comment-intent
```

The update applies only when every major change and retirement of a rule you use is accepted. Fork or exclude the rest first, then run the command again.

Major changes to rules you exclude or replace don't need consent, because your agents don't read them. `code-rules project update` still lists them so you can check that your exception still makes sense. If a retired rule is still named in an exclusion or replacement, the command stops and tells you which entry to delete.

## Keep an older version of a rule

To stay on a rule's older major version while updating the rest of the library, fork it into your project:

```sh
code-rules project add rule practices/testing/verify-retry-limits \
  --from fabrica@1.3.0 \
  --reason 'Batch jobs keep the 1.x retry policy until they migrate.'
```

This copies version `1.3.0` into `.code-rules/local/`, adds attribution that links to the original, and replaces the imported rule with your copy. Your fork no longer receives updates. Revisit it when you're ready to adopt the newer version, then remove the replacement and the local file.

Then update the library:

```sh
code-rules project update
```

## Pin a source

To hold a whole library at one release, set its `ref` to a rule version tag, such as `practices/testing/verify-retry-limits@1.3.0`, or to a commit, then run `code-rules project sync`. `code-rules project update` skips pinned sources. To follow releases again, remove `ref` and run `code-rules project update`. See [Select a revision](/reference/configuration/#select-a-revision).

## Review and commit the update

`code-rules project update` reports added, changed, and removed file paths. Before committing:

- Inspect the diff of `.code-rules/vendor/` for upstream changes, including rules hidden by your exclusions and replacements, and changes to retained licenses and notices.
- When a replacement's target changed, compare its old and new text before deciding whether your local rule still makes sense.
- When updated rules introduce competing obligations, use the [conflict-review prompt](/guides/conflicting-guidance/#generate-a-review-prompt) to inspect the combined guidance.

Commit the configuration, vendor snapshots, and generated files together.

## Change selected groups

Edit the relevant `sources.<name>.groups` and run `code-rules project sync`.
`code-rules project sync` imports the new groups from the release already recorded for that source, so the rest of your rules don't change.
The vendor snapshot must match that selection before an offline build can use it.
Regeneration removes a group index only when no source or discovered local group still supplies it.
Before deselecting the last library supplying a local rule's group, ensure `local/<group-id>/_group.yaml` exists.
If it already exists, keep the local files unchanged. Otherwise author group metadata, or move or remove the local rules.

## Recover from a failed update

`code-rules project update` and `code-rules project sync` validate and render before installing output.
A failed import, or an update that needs your consent, preserves the previous working ruleset.
Interrupted installations must be detected and recovered before another operation can claim success.

Rules never change automatically during coding, review, or offline checks, even if someone moves a tag your `ref` names.
Code Rules keeps the commit it recorded until you change `ref` or run `code-rules project update`.
