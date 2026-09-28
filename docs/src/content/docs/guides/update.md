---
title: "Update rules"
description: "Review each rule change, accept major changes deliberately, and preserve local decisions."
---

Updating rules brings changes from the libraries you use into your project. Library authors may improve advice, fix mistakes, add rules, or make rules stricter. Your project adopts those changes only when you run `code-rules project update`.

Reviewing an update lets you check that the changed guidance still fits your project, including any rules you've excluded or replaced. In this guide, you'll update your libraries, review each rule change, accept major changes or keep an older version of a rule, and commit the result. You'll also learn how to change your selection of groups and recover from a failed update.

Start with a project that already [imports rules](/guides/select-rules/). For details about how commands change files, see [Sync and recovery](/reference/sync/).

## Sync and update

Two commands import library rules, and they do different things:

- **`code-rules project sync`** imports the commits already recorded in `.code-rules/vendor/`. Everyone who syncs the project gets the same rules. Sync resolves a revision only for a new source, or after you change a source's repository or `ref`.
- **`code-rules project update`** moves sources to newer revisions, reports every rule change, and asks you to accept major changes before it writes anything.

Neither command runs during ordinary coding, review, or `project check`, so rules never change underneath your agents.

## Update your libraries

From the project root, run:

```sh
code-rules project update
```

To update only some libraries, name their sources, such as `code-rules project update fabrica`.

For a library that [versions rules](/concepts/rule/#how-a-rule-is-versioned), update moves to its newest release and reports each rule that changed:

```text
fabrica  4f1c2a9 -> 9e07b3d
  major    practices/testing/verify-retry-limits           1.3.0 -> 2.0.0
           Require a test at the limit for every retry policy.
  minor    practices/code-design/organize-code-by-feature  1.0.0 -> 1.1.0
           Add a Go example.
  new      practices/testing/verify-retries                1.0.0
           Add the rule.
  retired  practices/testing/check-retry-backoff           1.2.0, superseded by practices/testing/verify-retries
           Covered by the broader rule about testing retries.
  retired  practices/code-design/comment-intent            2.1.0, withdrawn
           Withdrawn after feedback that agents shouldn't add explanatory comments.

Nothing was updated: 1 major change and 2 retirements affect rules this project uses.
Review them, then run: code-rules project update --accept-major
```

Each line shows the change, the rule, and its old and new versions, followed by the summaries of every version in between. A retired rule shows its last version, the reason it was retired, and any replacement.

| Change | What it means for your project |
| --- | --- |
| `patch` | Clearer wording or examples. The obligation is unchanged. |
| `minor` | New guidance that work following the previous version still satisfies. |
| `major` | A stricter or different obligation. Code that followed the previous version could fail it. |
| `new` | A rule added to a group you import. |
| `retired` | The library stopped publishing the rule, so your agents will stop reading it. `superseded` means a named rule replaces it. `withdrawn` means the author no longer recommends the practice. |

When no rule your project uses has a major change or retirement, update applies the new release right away. Otherwise it changes nothing and exits with status `1`.

## Accept major changes

For each major change and retirement, read the new rule, or the reason for the retirement, and decide what your project should do. Compare the old and new text in `.code-rules/vendor/<source-name>/` after updating, or in the library's GitHub Releases. Then choose one of these for each rule:

- **Adopt it.** Plan any work your code needs to follow the new obligation. For a superseded rule, read its replacement, and check that you import the replacement's group.
- **Keep the older version.** [Fork the rule](#keep-an-older-version-of-a-rule) before updating. This also keeps a retired rule you still want to follow.
- **Stop using it.** Add an [exclusion](/guides/select-rules/#exclude-a-rule) with your reason.

When you've decided, apply the update:

```sh
code-rules project update --accept-major
```

Major changes to rules you exclude or replace don't need consent, because your agents don't read them. Update still lists them so you can check that your exception still makes sense. If a retired rule is still named in an exclusion or replacement, update stops and tells you which entry to delete.

## Keep an older version of a rule

To stay on a rule's older major version while updating the rest of the library, fork it into your project:

```sh
code-rules project add rule practices/testing/verify-retry-limits \
  --from fabrica@1.3.0 \
  --reason 'Our batch jobs keep the 1.x retry policy until the queue migration.'
```

This copies version `1.3.0` into `.code-rules/local/`, adds attribution that links to the original, and replaces the imported rule with your copy. Your fork no longer receives updates. Revisit it when you're ready to adopt the newer version, then remove the replacement and the local file.

Then update the library:

```sh
code-rules project update
```

## Libraries versioned as a whole

Some libraries publish one tag, such as `v1.2.0`, for all of their rules. For those sources, what update can do depends on the configured `ref`:

| `ref` | What update does |
| --- | --- |
| Version range, such as `>= 1.2.0, < 2.0.0` | Moves to the highest matching version tag. |
| Exact tag, such as `v1.2.0` | Moves only if the publisher moved the tag. |
| Full commit | Nothing. |

To move to a version outside the range, or to another tag, edit `ref` and run `code-rules project sync`. These libraries don't report individual rule changes, so review the diff of `.code-rules/vendor/<source-name>/` to see what changed.

If a library you use starts versioning rules, remove the source's `ref` and run `code-rules project sync` to follow its releases. Review the diff as you would any update.

## Review and commit the update

Update reports added, changed, and removed file paths. Before committing:

- Inspect the diff of `.code-rules/vendor/` for upstream changes, including rules hidden by your exclusions and replacements, and changes to retained licenses and notices.
- When a replacement's target changed, compare its old and new text before deciding whether your local rule still makes sense.
- When updated rules introduce competing obligations, use the [conflict-review prompt](/guides/conflicting-guidance/#generate-a-review-prompt) to inspect the combined guidance.

Commit the configuration, vendor snapshots, and generated files together.

## Change selected groups

Edit the relevant `sources.<name>.groups` and run `code-rules project sync`.
Sync imports the new groups from the release already recorded for that source, so the rest of your rules don't change.
The vendor snapshot must match that selection before an offline build can use it.
Regeneration removes a group index only when no source or discovered local group still supplies it.
Before deselecting the last library supplying a local rule's group, ensure `local/<group-id>/_group.yaml` exists.
If it already exists, keep the local files unchanged. Otherwise author group metadata, or move or remove the local rules.

## Recover from a failed update

Update and sync validate and render before installing output.
A failed import, or an update that needs your consent, preserves the previous working ruleset.
Interrupted installations must be detected and recovered before another operation can claim success.

Moved tags never update rules automatically during coding, review, or offline checks.
`project update` resolves tags again and reports a moved tag's new commit for review.
To keep a tag's previous content regardless, set `ref` to the previously recorded full commit SHA.
