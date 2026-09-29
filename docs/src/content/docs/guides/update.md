---
title: "Update rules"
description: "Preview each rule change, keep rules where they are when you need to, and adopt the rest deliberately."
---

Updating rules brings changes from the libraries you use into your project. Library authors may improve advice, fix mistakes, add rules, make rules stricter, or retire them. Your project adopts those changes only when you run `code-rules project update` and confirm them.

In this guide, you'll preview and apply updates, keep a rule at its current version when you're not ready for a change, update a single library or rule, and commit the result. You'll also learn how to import one library release, change which rules you import, and recover from a failed update.

Start with a project that already [imports rules](/guides/select-rules/). For details about how commands change files, see [Sync and recovery](/reference/sync/).

## Sync and update

Two commands import library rules, and they do different things:

- **`code-rules project sync`** imports the rule versions already recorded in `.code-rules/vendor/`. Everyone who syncs the project gets the same rules. It chooses a version only for a newly added source or selected rule, a changed repository, or when you add or change a pin or `ref`.
- **`code-rules project update`** looks for newer rule versions, new rules, and retirements, shows you a preview, and applies the changes once you confirm.

Neither command runs during ordinary coding, review, or `code-rules project check`, so rules never change underneath your agents.

## Update your libraries

From the project root, run:

```sh
code-rules project update
```

The command previews every change in each library, with each [rule version](/concepts/rule/#how-a-rule-is-versioned)'s summary, then asks you to confirm:

```text
team
  major    practices/testing/verify-retry-limits     1.3.0 -> 2.0.0
           Require a test at the limit for every retry policy.
  major    techs/react/prefer-server-components      1.4.0 -> 2.0.0
           Require server components for all data fetching.
  minor    techs/react/test-hooks-in-isolation       2.1.0 -> 2.2.0
           Add an example for custom hooks.
  new      practices/testing/verify-retries          1.0.0
           Add the rule.
  retired  practices/testing/check-retry-backoff     1.2.0
           Replaced by practices/testing/verify-retries.
           Covered by the broader rule about testing retries.
  pinned   practices/testing/verify-backoff          1.3.0
           Newest version: 2.0.0.
           Reason: Waiting on the author's response to #45.
```

| Change | What it means for your project |
| --- | --- |
| `patch` | Work that complied with the previous version still complies. The rule adds no new guidance. |
| `minor` | Work that complied with the previous version still complies. The rule adds new guidance. |
| `major` | Work that complied with the previous version could fail this one. |
| `new` | A rule the library added to a group you import. |
| `retired` | The library stopped publishing the rule, so your agents will stop reading it. The preview names its replacement when there is one. |
| `pinned` | A newer version exists, but your [pin](#keep-a-rule-at-its-current-version) keeps the rule where it is. |

Read the major changes, new rules, and retirements closely: each can change what your code must do. Then confirm, and the update applies exactly the changes the preview showed.

In a script or CI job, where there's no terminal to confirm in, the command only shows the preview. To apply it, pass `--yes`:

```sh
code-rules project update --yes
```

## Keep a rule at its current version

When an update brings a change you're not ready for, keep that rule where it is with a **pin**, and let everything else update. A pin records the version and why you're keeping it:

```yaml
sources:
  team:
    repository: https://github.com/acme/.code-rules.git
    groups:
      - techs/react
      - practices/testing
    pins:
      practices/testing/verify-retry-limits:
        version: "1.3.0"
        reason: Waiting on the author's response to acme/.code-rules#45.
    exclude: {}
    replace: {}
```

There are three ways to add one:

- **During the update.** In a terminal, `code-rules project update` asks about each major change and retirement: adopt it, or keep the current version. Choose to keep it, give a reason, and the command writes the pin.
- **With `--keep`.** Pin rules as part of the update, without prompts:

  ```sh
  code-rules project update \
    --keep team:practices/testing/verify-retry-limits \
    --reason "Waiting on the author's response to acme/.code-rules#45."
  ```

- **By hand.** Add the pin to `.code-rules/config.yaml`, using the version recorded in `.code-rules/vendor/<source-name>/_source.json`, and run `code-rules project sync`. An agent asked to keep a rule at its current version can do this for you.

A pinned rule keeps its identity: it still appears as the library's rule, with its version, in your generated guidance and provenance. Every update preview lists it, with its newest version and your reason, so the decision isn't forgotten.

When you're ready to adopt the newer version, delete the pin and update that rule:

```sh
code-rules project update team:practices/testing/verify-retry-limits
```

A pin can also move a rule to a specific version: set `version`, and `code-rules project sync` moves the rule to exactly that version, up or down. To change what a rule says instead, [fork it](/reference/cli/#fork-a-library-rule) into your project's local rules. See [Pin a rule](/reference/configuration/#pin-a-rule) for the details.

## Update one library or one rule

To limit an update, name what to update:

```sh
code-rules project update team
code-rules project update team:techs/react/prefer-server-components
```

The first updates only the `team` library. The second moves only that rule; nothing else changes, including new rules. The rest waits for your next full update. Nothing is recorded in configuration, so use a pin when you want the decision to last.

## Handle retirements

A retired rule appears in the preview with its last version, the summary explaining why, and its replacement if it has one. When you confirm, the rule is dropped. For a rule with a replacement, read the replacement, and check that you import its group.

To keep following a rule the library retires, choose to keep it when the update preview offers, or pin it to its last version before you confirm. Once the retirement is applied, the rule is gone and can't be pinned. If you exclude, replace, or individually select a rule that the library retires, that entry no longer does anything; sync and update warn about it so you can delete it.

## Import one library release

To import exactly what one library release published, set the source's `ref` to its tag:

```yaml
sources:
  vendor-rules:
    repository: https://github.com/example/engineering-rules.git
    groups: "*"
    ref: release/5
    exclude: {}
    replace: {}
```

`code-rules project update` doesn't move this source. To import another library release, change the tag and run `code-rules project sync`. To go back to following rule versions, remove `ref`.

`ref` also accepts any other tag or a full commit SHA, which library authors use to test unreleased changes. Rules with unreleased changes have no version, so Code Rules warns about them every time you sync or update. See [Import one revision](/reference/configuration/#import-one-revision).

## Review and commit the update

`code-rules project update` reports added, changed, and removed file paths. Before committing:

- Inspect the diff of `.code-rules/vendor/` for upstream changes, including rules hidden by your exclusions and replacements, and changes to retained licenses and notices.
- When a replacement's target changed, compare its old and new text before deciding whether your local rule still makes sense.
- When updated rules introduce competing obligations, use the [conflict-review prompt](/guides/conflicting-guidance/#generate-a-review-prompt) to inspect the combined guidance.

Commit the configuration, vendor snapshots, and generated files together.

## Change selected groups or rules

Edit the relevant `sources.<name>.groups` or `sources.<name>.rules` and run `code-rules project sync`.
`code-rules project sync` imports newly selected rules at their newest versions, and leaves the rest of your rules unchanged.
The vendor snapshot must match that selection before an offline build can use it.
Regeneration removes a group index only when no source or discovered local group still supplies it.
Before deselecting the last library supplying a local rule's group, ensure `local/<group-id>/_group.yaml` exists.
If it already exists, keep the local files unchanged. Otherwise author group metadata, or move or remove the local rules.

## Recover from a failed update

`code-rules project update` and `code-rules project sync` validate and render before installing output.
A failed import, or an update you didn't confirm, preserves the previous working ruleset.
Interrupted installations must be detected and recovered before another operation can claim success.

Rules never change automatically during coding, review, or offline checks.
Code Rules keeps the versions it recorded until you change your selection, pins, or `ref`, or confirm an update.
