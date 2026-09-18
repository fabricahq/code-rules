---
title: "Sync and recovery"
description: "Import libraries, generate resolved rules, and apply changes safely."
---

Sync fetches and validates library snapshots, resolves project rules, and installs the complete vendor and generated output. [Build or install the Go CLI](/guides/install/), then run from your project root:

```sh
code-rules sync
code-rules build
code-rules check
```

Without `--config`, commands use `.code-rules/config.json` relative to the working directory.
The configuration directory contains `local/`, `vendor/`, and `generated/`. The directory and configuration must already exist.

## Ownership

Sync replaces all of `vendor/` and `generated/`, including stale rules, assets, libraries, and index parts.
Keep authored files in `local/` and configuration, which Sync preserves. Files placed manually in managed directories are replaced or removed.
Build replaces only `generated/` and never contacts a repository. Check reads and compares without changing any files.


Human-readable output is the default. Use `--json` for one structured response. Sync and build report sorted `added`, `changed`, and `removed` paths for completed changes. Sync paths start with `vendor/` or `generated/`; build paths are relative to `generated/`.

Check reports `status` and `problems`, including each problem's path and repair command. It verifies generated files and the managed project README without writing either. Refresh an outdated README with `code-rules init`; repair generated output with `code-rules build`. Check exits 1 for differences or invalid inputs, 0 when current; usage errors exit 2.

Structured group-level update summaries remain future work. Review the changed source records and generated provenance for revision changes.

## Stored snapshots

Sync writes `vendor/<alias>/_source.json` with `formatVersion: 1`, repository, requested `ref` or `version`, resolved commit, concrete groups, and `groupSelection`.
Version selections additionally record `resolvedTag` and `resolvedVersion`.

The `files` object maps every retained library-relative path to its lowercase SHA-256 hex digest, computed from original bytes.
It excludes `_source.json` itself. The directory name supplies the source alias; there are no timestamps or machine-specific paths.


Offline build and check reject missing, changed, or unexpected vendor files, invalid records, and source selections that differ from configuration.
Sync deliberately refreshes vendor files, including locally modified snapshots. Put project changes in local rules instead.

Digests detect changes relative to the committed record; they do not authenticate a record that was also modified.

## Safe application

Writers use an exclusive `.code-rules-lock` in the configuration directory.
They validate configuration, local files, and existing managed directories, prepare all results, and stage complete replacement directories.
Immediately before applying, they compare the original input and output bytes again and reject concurrent changes.
All operations reject symlinks and special files in inspected trees; regular files with multiple hard links are also rejected.
The configuration parent is canonicalized, so ordinary system aliases such as macOS `/tmp` work.


A `.code-rules-transaction` journal retains previous output during replacement. If applying fails, the writer restores the previous directories.
If the process is killed during replacement, the next writer recovers the previous output before starting its operation.

A completed transaction is renamed to `.code-rules-cleanup` before deleting backups. Interrupted cleanup cannot make a committed update appear incomplete; the next writer finishes deletion.

If output or backups were edited after interruption, recovery refuses to overwrite them and reports that manual recovery is needed.


A dead lock owner on the same host can be reclaimed. Active owners, locks from another host, and incomplete ownership records block writes.

Unprepared staging without a journal or backups is discarded automatically. For an incomplete lock or damaged journal, verify that no writer is running before manual recovery.
Preserve backups and journals when the error requests manual recovery. Read-only check reports active writes or pending recovery without repairing anything.


These guarantees apply to cooperating Code Rules writers on a local filesystem with normal rename semantics.
Replacing two directories is not one atomic filesystem action; unrelated readers can briefly observe mixed output during replacement.
Check detects an active transaction. Avoid editing the managed directories during sync.
The implementation does not promise power-loss durability, protection against a hostile process racing filesystem paths, or network-filesystem locking.
Cancellation is honored before replacement; once replacement starts, it finishes or rolls back before returning.


Inputs are bounded to 64 MiB per file, 256 MiB per tree, 30,000 entries, and 64 path segments.
The importer also applies its own stricter limits. Output paths must not collide when normalized for case-insensitive filesystems.
