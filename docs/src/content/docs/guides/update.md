---
title: "Update rules"
description: "Review each rule change, accept major changes deliberately, and preserve local decisions."
---

Updating rules brings changes from the libraries you use into your project. Library authors may improve advice, fix mistakes, add rules, or make rules stricter. Your project adopts those changes only when you run `code-rules project update`.

In this guide, you'll update your libraries, review each rule change, accept major changes or pin a rule to its current version, and commit the result. You'll also learn how to choose versions rule by rule, import one library release, change your selection of groups, and recover from a failed update.

Start with a project that already [imports rules](/guides/select-rules/). For details about how commands change files, see [Sync and recovery](/reference/sync/).

## Sync and update

Two commands import library rules, and they do different things:

- **`code-rules project sync`** imports the rule versions already recorded in `.code-rules/vendor/`. Everyone who syncs the project gets the same rules. It chooses a rule's version again only for a new source or group, or after you change that rule's version choice.
- **`code-rules project update`** moves rules to newer versions, as far as each source's `versions` allows. It reports every rule change and asks you to accept major changes and retirements before it writes anything.

Neither command runs during ordinary coding, review, or `code-rules project check`, so rules never change underneath your agents.

## Update your libraries

From the project root, run:

```sh
code-rules project update
```

To update only some libraries, name their sources, such as `code-rules project update fabrica`.

By default, `code-rules project update` moves every rule to its newest version and reports each [rule version](/concepts/rule/#how-a-rule-is-versioned) that changed:

```text
fabrica
  major    practices/testing/verify-retry-limits           1.3.0 -> 2.0.0
           Require a test at the limit for every retry policy.
  minor    practices/code-design/organize-code-by-feature  1.0.0 -> 1.1.0
           Add a Go example.
  new      practices/testing/verify-retries                1.0.0
           Add the rule.
  retired  practices/testing/check-retry-backoff           1.2.0 superseded
           Replaced by practices/testing/verify-retries.
           Covered by the broader rule about testing retries.
  held     practices/testing/verify-backoff                1.3.0
           Newest version: 2.0.0.

Nothing was updated: 1 major change and 1 retirement affect
rules this project uses. Review them, then accept them all:
  code-rules project update --accept-major
Or keep a rule at its current version by adding its entry
under sources.fabrica.versions.rules:
  practices/testing/verify-retry-limits: "1.3.0"
  practices/testing/check-retry-backoff: "1.2.0"
```

Each line shows the change, the rule, and its old and new versions, followed by the summaries of every version in between. A retired rule shows its last version, the reason it was retired, and any replacement. A held rule shows its newest version without moving to it, because its version choice keeps it where it is.

| Change | What it means for your project |
| --- | --- |
| `patch` | Work that complied with the previous version still complies. The rule adds no new guidance. |
| `minor` | Work that complied with the previous version still complies. The rule adds new guidance. |
| `major` | Work that complied with the previous version could fail this one. |
| `new` | A rule added to a group you import. |
| `retired` | The library stopped publishing the rule, so your agents will stop reading it. `superseded` means a named rule replaces it. `withdrawn` means the author no longer recommends the practice. |
| `held` | A newer version exists, but the rule's [version choice](#choose-versions-rule-by-rule) keeps it where it is. |

When no rule your project uses has a major change or retirement, `code-rules project update` applies the changes right away. Otherwise it changes nothing and exits with status `1`.

## Accept major changes

For each major change and retirement, read the new rule, or the reason for the retirement, and decide what your project should do. Compare the old and new text in `.code-rules/vendor/<source-name>/` after updating, or in the library's GitHub Release pages. Then choose one of these for each rule:

- **Adopt it.** Plan any work your code needs to follow the new obligation. For a superseded rule, read its replacement, and check that you import the replacement's group.
- **Keep the current version.** [Pin the rule](#choose-versions-rule-by-rule) to the version you have, by pasting the entry the command printed into configuration. This also keeps a retired rule you still want to follow.
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

The update applies only when every major change and retirement of a rule you use is accepted. Pin or exclude the rest first, then run the command again.

Major changes to rules you exclude or replace don't need consent, because your agents don't read them. `code-rules project update` still lists them so you can check that your exception still makes sense. If a rule you exclude or replace is retired, your exception no longer points at anything. The command stops and tells you which entry to delete.

## Choose versions rule by rule

By default, every rule follows its newest version. To treat some rules differently, add `versions` to the source in `.code-rules/config.yaml`. For example, to keep `verify-backoff` at version 1.3.0 while the rest of the library moves forward:

```yaml
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    groups:
      - practices/testing
    versions:
      rules:
        practices/testing/verify-backoff: "1.3.0"
    exclude: {}
    replace: {}
```

Each rule can be set to:

| Setting | Effect |
| --- | --- |
| `latest` | Follow the newest version. This is the default. |
| An exact version, such as `"1.3.0"` | Pin the rule to exactly that version. Update still reports newer versions. |
| A constraint, such as `"~> 1.3"` | Take any version from 1.3 up to, but not including, 2.0: fixes and additions, never a major change. |

To keep a rule at the version you have, pin it to that version. The rule's current version is in `.code-rules/vendor/<source-name>/_source.json` and in its generated rule file, and `code-rules project update` prints the pinning entry when it stops for a major change. An agent can look the version up and add the entry for you.

To reverse the default, keeping every rule where it is and letting only chosen rules move, set `default: hold`:

```yaml
versions:
  default: hold
  rules:
    practices/testing/verify-retry-limits: latest
```

After editing `versions`, run `code-rules project sync`. For each rule whose choice you changed, sync applies the new choice; for example, setting a rule to `"1.3.0"` moves it to 1.3.0, and changing it to `latest` moves it to its newest version. Editing `versions` is itself your consent, so sync doesn't ask for `--accept-major`. See [Choose versions](/reference/configuration/#choose-versions) for every option.

A pinned rule keeps its identity: it still appears as the library's rule, with its version, in your generated guidance and provenance. To change what a rule says instead, [fork it](/reference/cli/#fork-a-library-rule) into your project's local rules.

## Import one library release

To import exactly what one library release published, choose the library release by number:

```yaml
versions:
  release: 5
```

`code-rules project update` doesn't move this source. To import another library release, change the number and run `code-rules project sync`. To go back to choosing versions rule by rule, remove `release`.

## Review and commit the update

`code-rules project update` reports added, changed, and removed file paths. Before committing:

- Inspect the diff of `.code-rules/vendor/` for upstream changes, including rules hidden by your exclusions and replacements, and changes to retained licenses and notices.
- When a replacement's target changed, compare its old and new text before deciding whether your local rule still makes sense.
- When updated rules introduce competing obligations, use the [conflict-review prompt](/guides/conflicting-guidance/#generate-a-review-prompt) to inspect the combined guidance.

Commit the configuration, vendor snapshots, and generated files together.

## Change selected groups or rules

Edit the relevant `sources.<name>.groups` or `sources.<name>.rules` and run `code-rules project sync`.
`code-rules project sync` imports newly selected rules at the newest versions their version choices allow, and leaves the rest of your rules unchanged.
The vendor snapshot must match that selection before an offline build can use it.
Regeneration removes a group index only when no source or discovered local group still supplies it.
Before deselecting the last library supplying a local rule's group, ensure `local/<group-id>/_group.yaml` exists.
If it already exists, keep the local files unchanged. Otherwise author group metadata, or move or remove the local rules.

## Recover from a failed update

`code-rules project update` and `code-rules project sync` validate and render before installing output.
A failed import, or an update that needs your consent, preserves the previous working ruleset.
Interrupted installations must be detected and recovered before another operation can claim success.

Rules never change automatically during coding, review, or offline checks.
Code Rules keeps the versions it recorded until you change `versions` or run `code-rules project update`.
