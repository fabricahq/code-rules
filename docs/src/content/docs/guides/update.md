---
title: "Update rules"
description: "Adopt upstream changes deliberately while preserving local decisions."
---

To adopt newer rules, select the version you want and run `code-rules sync`.
Sync downloads the selected rules from your source libraries and rebuilds the indexes and effective rule files that agents read.
The command is part of the proposed CLI and has not shipped yet.

## Update a library

1. Choose the latest tag or full Git commit SHA you want to adopt from the library.
2. Set `sources.<name>.ref` in `code-rules/config.json` to that version.
3. From the project root, run:

   ```sh
   code-rules sync
   ```

4. Review and commit the configuration, refreshed vendor snapshots, and regenerated files together.

Sync follows the configured refs; it does not automatically choose the newest release.
A commit SHA stays fixed, and a tag resolves to its current target.
To move from `v1.0.0` to `v1.1.0`, change the ref before syncing.

## What sync refreshes

A single sync performs the download and regeneration together:

1. Resolve each source's ref to an exact commit.
2. Download its selected rule groups into `code-rules/vendor/<source-name>/` and record the resolved commit.
3. Apply source-specific exclusions and replacements, then include the project's local rules.
4. Regenerate the group indexes and individual effective rule files under `code-rules/generated/`.
5. Regenerate `code-rules/generated/RULES.md` and provenance records, then install the complete validated result.

For example, `code-rules/generated/practices/testing.md` indexes the active testing rules from all selected sources and the project, linking to their full definitions.
After sync, that index reflects the downloaded versions and the project's local choices.
You do not need to run `build` separately after sync.

## Review the update

Other configured refs remain unchanged.
Sync resolves every configured tag again, so also review changes to other sources' resolved commits if their tags have moved.
The report identifies requested refs, old and new resolved commits, and added, removed, and changed rules by source-qualified ID.
It also shows upstream changes hidden by exclusions or replacements, plus changes to preserved licenses and notices.
When a replacement target changes, compare its old and new text before deciding whether the local exception still makes sense.

When updated rules introduce competing obligations, use the [conflict-review prompt](/guides/conflicting-guidance/#generate-a-review-prompt) to inspect the combined guidance.

A removed or renamed target causes a configuration error.
Update the affected exclusion or replacement deliberately.

## Change selected groups

Edit the relevant `sources.<name>.groups` and sync again when the imported selection changes.
The vendor snapshot must match that selection before an offline build can use it.
Regeneration removes a group index only when no source or declared local-only group still supplies it.
Move or remove local rules before deselecting their last imported group, or declare that group in `localGroups` with local metadata.

## Recover from a failed update

The intended importer validates and renders before installing output.
A failed import preserves the previous working ruleset.
Interrupted installations must be detected and recovered before another operation can claim success.

Moved tags never update rules automatically during coding, review, or offline checks.
To adopt a moved tag deliberately, run sync and review the new resolved commit.
To retain its previous content through future syncs, set `ref` to the previously recorded full commit SHA.
