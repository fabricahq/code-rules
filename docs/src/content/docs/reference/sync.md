---
title: "Sync and recovery"
description: "When to sync, build, or check, and how to recover from stale or interrupted output."
---

After you add a library, change its revision or selected groups, or want updates allowed by a version range, run `code-rules project sync`. Sync fetches the selected library files and generates the guidance your agents read.

## Choose the right command

| Command | Use it when | Network access |
| --- | --- | --- |
| `code-rules project sync` | You changed library selection or need to restore imported files. It also builds guidance. | Required for each configured library. |
| `code-rules project build` | You changed local rules or exceptions and the stored library files are valid. | Not required. |
| `code-rules project check` | You want to verify stored inputs, generated guidance, and the managed Code Rules guide without changing files. | Not required. |

You do not need to build after a successful sync. For how Code Rules selects rules, see [How imports work](/reference/imports/).

## Run from your project root

After [setting up your project](/start-here/set-up-project/), run:

```sh
code-rules project sync
code-rules project check
```

In a Git repository, project commands also work from a subdirectory and use the nearest repository root. Outside Git, run them from the project root.

## Which files change

Sync replaces `.code-rules/vendor/` with the selected library files and replaces `.code-rules/generated/` with active rules and indexes. Build regenerates `generated/` from stored inputs. Both commands can refresh an older, unedited `.code-rules/README.md`. Check writes nothing.

All three commands read `.code-rules/config.yaml` and your `local/` rules without changing them. Keep project-specific edits there. Edits inside `vendor/` or `generated/` can be replaced or removed. For the full layout, see [Project files](/reference/files/).

## Read the command's result

Sync and build report added, changed, and removed files. Review their Git diff before committing. Use [provenance](/reference/provenance/) to identify changes in library revisions or rule origins.

Check reports whether files are current. If they are stale or missing, it names the problem and suggests a repair command. Invalid inputs return an error instead. If another tool needs a structured result, use `--json`:

```sh
code-rules project check --json
```

A successful check exits `0`. Stale output or invalid input exits `1`; incorrect command use exits `2`.

## Repair missing or outdated files

| What check reports | What to do |
| --- | --- |
| Generated guidance is stale or missing, but stored imports are valid. | Run `code-rules project build`, then check again. |
| Imported files are missing, changed, or do not match selected sources or groups. | Copy any work you need to keep outside `vendor/`, then run `code-rules project sync` and check again. Make project rules in `local/` or change the source library. |
| The managed Code Rules guide is missing or outdated. | Run build or sync to refresh it with guidance. Run `code-rules project init` if you only need to refresh setup files. |
| Configuration or rule metadata is invalid. | Fix the reported input, then rerun the appropriate command and check. |

### How stored imports are checked

Build and check work offline. They compare your selected sources and groups with the stored imports and detect missing, changed, or unexpected files. Sync fetches the selected files again. An offline check cannot tell whether a remote tag moved or whether a remote repository changed. For the recorded revision, see [Provenance](/reference/provenance/).

## Recover from an interrupted update

Sync and build prepare new output before replacing the old files. If an update fails, Code Rules restores the previous output when it can do so safely. A later sync or build can also recover from a stopped process.

| Situation | What to do |
| --- | --- |
| Another Code Rules command is writing files. | Let it finish before retrying. |
| The previous process stopped, or check reports pending recovery. | After the writer exits, rerun sync or build. Check identifies the problem but does not repair it. |
| Recovery reports edited output or backups, a damaged recovery record, or uncertain ownership of the write. | Stop retries. Confirm no writer is running, inspect the reported files, and preserve the backups and recovery files for manual recovery. |

Do not delete recovery files or force an overwrite to make a retry pass. Code Rules refuses to overwrite files whose state it cannot establish. Keep authored changes in configuration and `local/`; review the result after recovery and run check again.

## How updates protect your files

Sync validates all selected libraries before changing managed output. Build validates its stored inputs before writing. If another process changes inputs or output during either command, the update stops. Check does not accept an update that is still in progress as consistent.

These protections assume cooperating Code Rules commands and a local filesystem with normal rename behavior. They do not guarantee recovery from power loss, hostile concurrent edits, or network filesystem behavior. Code Rules also rejects symbolic links, special files, and files with multiple hard links in the directories it manages. For import-specific size limits, see [Import limits and version errors](/reference/imports/#import-limits-and-version-errors).

For the code paths and tests behind sync and recovery, [inspect the implementation](/for-agents/#inspect-implementation-and-tests).
