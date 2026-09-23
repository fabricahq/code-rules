---
title: "Sync and recovery"
description: "When to run sync, build, or check, which files they change, and how to recover from problems."
---

The `code-rules project sync` command updates your project's imported library files and regenerates the guidance your agents read. Run it after adding a library, changing its revision or selected groups, or to fetch updates allowed by your configured version range.

This page explains how to run sync, what it changes, and what to do when files are missing, outdated, or left by an interrupted update. It also explains when `code-rules project build` or `code-rules project check` is enough. For how Code Rules selects and combines library rules, see [How imports work](/reference/imports/).

## Choose the right command

- **`code-rules project sync`**: Fetch selected library revisions or restore imported files. Sync validates the library files and regenerates agent guidance.
- **`code-rules project build`**: Apply local rule changes or exceptions using the library files you already have. Build regenerates guidance without contacting a repository.
- **`code-rules project check`**: Find out whether generated guidance and the managed Code Rules guide are up to date. Check validates stored inputs and output, reports problems, and leaves files unchanged.

You do not need to run build after a successful sync; sync already generates the guidance. Run check when you want to verify consistency without making changes.

## Run from your project root

After [installing Code Rules](/start-here/install/) and [setting up your project](/start-here/set-up-project/), run:

```sh
code-rules project sync
```

In a Git repository, project commands find the nearest repository root and use its `.code-rules/config.json`. Outside Git, run commands from the project root. Custom configuration locations are not supported. The paths below are relative to `.code-rules/`.

## Which files change

| File or directory | What it contains | What the commands do |
| --- | --- | --- |
| `config.json` | Your selected libraries, groups, and exceptions. | Sync, build, and check read it without changing it. |
| `README.md` | The managed Code Rules guide. | Init, build, and sync refresh an older, unedited guide. Check verifies it without changing it. |
| `local/` | Rules and replacements you author for this project. | Sync, build, and check preserve these files. |
| `vendor/` | Original files copied from selected library revisions. | Sync replaces this directory. Build and check validate it without changing it. |
| `generated/` | Rules and reading indexes for your agents. | Sync and build replace this directory. Check compares it with the expected output. |

Replacement includes removing files that no longer belong in the output, such as removed rules, old index pages, and unused library folders. Files you add or edit inside `vendor/` or `generated/` can be replaced or removed. Keep your changes in configuration and `local/`.

For the complete directory layout, see [Project files](/reference/files/).

## Read the command's result

Commands print human-readable output by default. Add `--json` when another tool needs one structured response:

```sh
code-rules project check --json
```

Sync and build report counts and sorted lists of added, changed, and removed paths. JSON output includes those lists in `added`, `changed`, and `removed`.

- Sync paths start with `vendor/` or `generated/`.
- Build paths are relative to `generated/`.

There is no separate structured summary of added or removed groups. To see which library revisions changed, review the source records and generated [provenance records](/reference/provenance/).

Check reports `status` and `problems`, including each problem's path and suggested repair command. It verifies both generated guidance and the managed Code Rules guide without writing either.

| Check exit code | Meaning |
| --- | --- |
| `0` | The files are current. |
| `1` | Files differ from the expected result, or an input is invalid. |
| `2` | The command was used incorrectly, such as with an invalid option. |

## Repair missing or outdated files

Use the problem reported by `code-rules project check` to choose a repair:

| Problem | What to do |
| --- | --- |
| Generated guidance is missing or outdated, but imported files are valid. | Run `code-rules project build`. |
| Imported files are missing, modified, or no longer match your configured sources or groups. | Run `code-rules project sync`. Preserve any edits you intended to keep as local rules first. |
| The managed Code Rules guide is missing or outdated. | Run `code-rules project build` or `code-rules project sync` to refresh it and regenerate guidance. Run `code-rules project init` to refresh only setup files. |
| Configuration or rule metadata is invalid. | Correct the reported input, then retry the appropriate command. |

Run `code-rules project check` again after repairing the problem.

### How stored imports are checked

For each library, Code Rules records its imported revision and file checksums in `vendor/<source-name>/_source.json`. A **checksum** detects whether a file's contents differ from the recorded copy.

Build and check work offline. They reject missing, changed, or unexpected imported files, invalid source records, and library selections that no longer match your configuration. Sync fetches the selected library files again, including replacing locally modified copies.

Checksums detect changes relative to the stored record. They cannot establish that files are authentic if someone also changed that record. For record fields and rule origins, see [Provenance](/reference/provenance/).

## Recover from an interrupted update

Sync and build keep the previous output while installing replacement files. If installation fails, the command restores the previous directories. If the process stops during replacement, the next sync or build recovers the previous output before starting its own work.

If the update completed but cleanup was interrupted, the next sync or build finishes deleting the backups. The completed update remains complete.

| Situation | What to do |
| --- | --- |
| Another Code Rules command is still writing files. | Let it finish before starting another update. |
| A previous writing process stopped on this machine. | Retry sync or build. Code Rules can reclaim its lock and recover the interrupted operation. |
| Check reports a pending recovery. | Run sync or build after any active writer exits. Check reports the problem but does not repair it. |
| Recovery reports edited output or backups, an incomplete lock, or a damaged journal. | Stop automatic retries, verify that no writer is running, and inspect the reported files. Preserve backups and the recovery journal for manual recovery. |

A lock records which process owns the update. Code Rules only reclaims a lock when it can establish that the process has stopped on the same machine. Active processes, locks from another machine, and incomplete ownership records block writes.

Recovery refuses to overwrite output or backups edited after an interruption. Follow the reported error instead of deleting recovery files to force the command through.

## How updates protect your files

Sync and build prepare and validate the complete replacement before installing it. Immediately before replacement, they check that the original input and output files have not changed. If another process changed them, the update stops.

Code Rules uses these temporary directories beside your configuration:

| Directory | Purpose |
| --- | --- |
| `.code-rules-lock` | Allows only one cooperating Code Rules command to write at a time. |
| `.code-rules-transaction` | Holds the recovery journal and previous output during replacement. The journal records the update's progress. |
| `.code-rules-cleanup` | Marks a completed update whose backups can be deleted. |

After completing an update, Code Rules renames the transaction directory to the cleanup directory before deleting backups. It also discards abandoned staging files when no journal or backups exist.

Avoid editing managed directories during an update. Replacing `vendor/` and `generated/` takes separate filesystem operations, so another program can briefly see a mixture of old and new files. Check detects an active update instead of accepting mixed output as consistent.

Cancellation can stop work before replacement starts. Once replacement begins, the command finishes or rolls back before returning.

### Filesystem requirements and limits

These protections apply to cooperating Code Rules commands on a local filesystem with normal file-renaming behavior. They do not guarantee recovery from power loss, protection against hostile concurrent file changes, or locking on network filesystems.

All operations reject symbolic links and special files within the directories they inspect. They also reject regular files with multiple hard links. Ordinary aliases in the project's parent path, such as macOS `/tmp`, are supported.

| Input resource | Limit |
| --- | --- |
| Each file | 64 MiB. |
| Each directory tree | 256 MiB and 30,000 entries. |
| Each path | 64 segments. |

Imports have additional, stricter [limits](/reference/imports/#import-limits-and-version-errors). Output paths must remain distinct after normalization for case-insensitive filesystems.
