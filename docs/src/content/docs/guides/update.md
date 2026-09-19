---
title: "Update rules"
description: "Adopt upstream changes deliberately while preserving local decisions."
---

To adopt newer rules, select an exact revision or version constraint and run `code-rules project sync`.
Sync downloads the selected rules from your source libraries and rebuilds the indexes and resolved rule files that agents read.
Sync works with an [installed CLI](/start-here/install/). See [Sync and recovery](/reference/sync/) for filesystem behavior.

## Update a library

1. Choose an exact tag or commit, or a HashiCorp version constraint such as `>= 1.2.0, < 2.0.0`.
2. Set `sources.<name>.ref` for an exact revision, or `sources.<name>.version` for a constraint. Specify exactly one.
3. From the project root, run:

   ```sh
   code-rules project sync
   ```

4. Review and commit the configuration, refreshed vendor snapshots, and regenerated files together.

For an exact `ref`, sync follows that ref. For a `version` constraint, it selects the highest matching semantic version tag.
A commit SHA stays fixed, and a tag resolves to its current target.
To move from `v1.0.0` to `v1.1.0`, change the ref before syncing.

## What sync refreshes

A single sync performs the download and regeneration together:

1. Resolve each source's exact ref or version constraint to a commit.
2. Download its selected rule groups into `.code-rules/vendor/<source-name>/` and record the resolved commit.
3. Apply source-specific exclusions and replacements, then include the project's local rules.
4. Regenerate the group indexes and individual resolved rule files under `.code-rules/generated/`.
5. Regenerate `RULES.md`, library READMEs, declared license copies, and provenance under `.code-rules/generated/`, then install the complete validated result.

For example, `.code-rules/generated/groups/practices/testing.md` lists the active testing rules from all selected sources and the project. Small groups include full definitions; larger groups link to them.
After sync, that index reflects the downloaded versions and the project's local choices.
You do not need to run `build` separately after sync.

## Review the update

Other configured refs remain unchanged.
Sync resolves every configured tag and version constraint again, so also review changes to other sources' resolved commits if their tags have moved.
The command reports added, changed, and removed file paths. Inspect the Git diff of configuration, vendor source records, and generated provenance to compare requested revisions, selected tags, and resolved commits.
Inspect the vendor diff for upstream changes hidden by exclusions or replacements, and changes to retained licenses and notices.
When a replacement target changes, compare its old and new text before deciding whether the local exception still makes sense.

When updated rules introduce competing obligations, use the [conflict-review prompt](/guides/conflicting-guidance/#generate-a-review-prompt) to inspect the combined guidance.

A removed or renamed target causes a configuration error.
Update the affected exclusion or replacement deliberately.

## Change selected groups

Edit the relevant `sources.<name>.groups` and sync again when the imported selection changes.
The vendor snapshot must match that selection before an offline build can use it.
Regeneration removes a group index only when no source or discovered local group still supplies it.
Before deselecting the last library supplying a local rule's group, ensure `local/<group-id>/_group.json` exists.
If it already exists, keep the local files unchanged. Otherwise author group metadata, or move or remove the local rules.

## Recover from a failed update

Sync validates and renders before installing output.
A failed import preserves the previous working ruleset.
Interrupted installations must be detected and recovered before another operation can claim success.

Moved tags never update rules automatically during coding, review, or offline checks.
To adopt a moved tag deliberately, run sync and review the new resolved commit.
To retain its previous content through future syncs, set `ref` to the previously recorded full commit SHA.
