---
title: "Update rules"
description: "Adopt upstream changes deliberately while preserving local decisions."
slug: guides/update
---

Your project adopts new library rules when you run `code-rules project sync`. Sync fetches the revisions allowed by `.code-rules/config.yaml` and regenerates the guidance your agents read. Review the resulting diff before committing so changed or new advice fits the project and its local exceptions.

In this guide, you'll choose a revision, sync all configured sources, review the new rules, and commit the adopted result. You can use the same steps to change selected groups.

Before you begin, [import a library](/guides/select-rules/) and make sure your project has `.code-rules/config.yaml`. Run commands from the project root, the top-level directory of your codebase. In a Git repository, project commands also work from a subdirectory. Keep a clean working tree or identify existing changes so you can review the update's diff.

## Update a library

1. Open `.code-rules/config.yaml` and choose the source to update. Set exactly one revision field under `sources.<name>`:

   - Use `ref` for an exact tag such as `v1.1.0` or a full commit SHA.
   - Use `version` for a [version constraint](/reference/configuration/#semantic-version-constraints) such as `">= 1.1.0, < 2.0.0"`.

   For example, to move a source named `acme-rules` to tag `v1.1.0`, change only its `ref` value:

   ```yaml
   sources:
     acme-rules:
       ref: v1.1.0
   ```

   The YAML snippet shows only the field to change. Keep the source's `repository`, `groups`, and any `exclude` or `replace` entries. If switching from `ref` to `version`, remove `ref`; if switching back, remove `version`. A full commit SHA selects immutable content. An exact tag looks fixed in configuration but can move in the remote repository. A version constraint can select a newer matching tag on a later sync.

2. Fetch the selected rules and verify the generated files:

   ```sh
   code-rules project sync
   code-rules project check
   ```

   Sync processes **every configured source**, not only `acme-rules`. It resolves each tag and version constraint again, then refreshes `.code-rules/vendor/` and `.code-rules/generated/`. You do not need to run build after sync. Check should pass with no stale or inconsistent files.

3. Review the command's changed paths and your Git diff. Compare `.code-rules/config.yaml`, each affected `vendor/<source>/_source.json`, and `.code-rules/generated/provenance.json` to see the requested revision, selected tag, and resolved commit. Inspect changes to the vendored rules, licenses, and notices. Sync can update another source if its configured tag moved or its version constraint now matches a newer tag.

4. Read `.code-rules/generated/RULES.md`, the affected group indexes, and their full resolved rules. Check new and changed obligations against project practice. Review any excluded or replaced upstream rule that changed to see whether the local exception still fits. If rules now require incompatible actions, follow [Resolve conflicting rules](/guides/conflicting-guidance/#generate-a-review-prompt).

5. Commit the reviewed configuration, vendor snapshots, and generated files together. Include local rule changes if the update required them. The committed snapshot is what offline builds, checks, and ordinary agent work use until the next sync.

## Change selected groups

To change which groups you import, edit `sources.<name>.groups` in `.code-rules/config.yaml`, then follow the sync and review steps above. Use an array of group IDs or one supported wildcard selector. For the exact YAML shape, see [Group selection](/reference/configuration/#group-selection).

Before deselecting the last library that supplies a group with local rules, make sure `local/<group-id>/_group.yaml` exists. Otherwise, add local group metadata or move the local rules. After sync, an unselected group index disappears only if no other selected source or local group still supplies it.

## Recover from a failed update

If sync fails, read its reported problem before retrying. A missing or renamed rule targeted by `exclude` or `replace` requires a deliberate configuration change. An invalid library or inaccessible revision requires correcting the source or revision. Sync validates the replacement before installing it, so a failed import leaves the previous working ruleset in place.

If an update stops during installation, retry sync after any active writer exits. Code Rules recovers the previous output before starting again when it can safely do so. If recovery reports edited backups, a damaged journal, or an uncertain lock, preserve the reported files and follow [Sync and recovery](/reference/sync/#recover-from-an-interrupted-update). Do not remove recovery files to force a retry.

A moved tag does not change your committed rules during ordinary work or offline checks. Sync resolves it again. To keep the previous content through future syncs, set `ref` to the full `resolvedCommit` recorded in the prior provenance and sync again. Review that result before committing. For the full update and recovery behavior, see [Sync and recovery](/reference/sync/).
