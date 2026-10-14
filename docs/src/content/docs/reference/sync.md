---
title: "Sync and recovery"
description: "When to run sync, build, or check, which files they change, who owns installed skill folders, and how to recover from problems."
---

The `code-rules project sync` command imports your project's library files, regenerates the guidance your agents read, and installs your project's skills. Run it after adding a source, changing its selected groups, rules, or skills, pins, `ref`, or `follow`, changing `skillTargets` or `extends`, or to restore imported files.

Sync imports the rule and skill versions recorded for each source, the commit recorded for each plain repository, and the library release recorded for a preset, so it never adopts newer versions on its own. To move to newer versions, run `code-rules project update`, which previews every change and applies it only after you confirm, offering to pin each major change or retirement instead. See [Update rules](/guides/update/).

This page explains how to run sync, what it changes, and what to do when files are missing, outdated, or left by an interrupted update. It also explains when `code-rules project build` or `code-rules project check` is enough. For how Code Rules selects and combines library rules, see [How imports work](/reference/imports/).

## Choose the right command

- **`code-rules project sync`**: Import the recorded rule and skill versions, or choose versions for new and changed sources. Sync validates the library files, regenerates agent guidance, and installs skills into the target folders.
- **`code-rules project update`**: Preview newer versions, new rules and skills, retirements, new commits on followed branches, and preset changes, and apply them after you confirm. Pinned rules and skills, and sources that use `ref`, don't move. Update then does everything sync does.
- **`code-rules project build`**: Apply local rule and skill changes or exceptions using the files you already have. Build regenerates guidance and installed skills without contacting a repository.
- **`code-rules project check`**: Find out whether generated guidance, installed skills, and the managed Code Rules guide are up to date. Check validates stored inputs and output, reports problems, and leaves files unchanged.

You do not need to run build after a successful sync or update; both already generate the guidance. Run check when you want to verify consistency without making changes.

## Run from your project root

After [installing Code Rules](/start-here/install/) and [setting up your project](/start-here/set-up-project/), run:

```sh
code-rules project sync
```

In a Git repository, project commands find the nearest repository root and use its `.code-rules/config.yaml`. Outside Git, run commands from the project root. Custom configuration locations are not supported. The paths below are relative to `.code-rules/`, except the target folders, `.claude/skills/` and `.agents/skills/`, which are relative to the project root.

## Which files change

| File or directory | What it contains | What the commands do |
| --- | --- | --- |
| `config.yaml` | Your sources, preset, install targets, selected groups, rules, and skills, pins, and exceptions. | Sync, build, and check read it without changing it. Update reads it, and writes a pin when you keep a rule or skill at its current version, or an exclusion when you decline a new rule or skill. |
| `README.md` | The managed Code Rules guide. | Init, build, and sync refresh an older, unedited guide. Check verifies it without changing it. |
| `local/` | Rules, replacements, and skills you author for this project. | Sync, build, and check preserve these files. Sync and update add `local/<group-id>/_group.yaml` only when your local rules in a group would otherwise lose the group's metadata; see [Keep a local rule's group](#keep-a-local-rules-group). Update replaces a forked rule or skill only when you pass `--update-fork` or answer `replace`. |
| `vendor/` | Original files copied from the selected rule and skill versions, and the preset file. | Sync and update replace this directory. Build and check validate it without changing it. |
| `generated/` | Rules, reading indexes, `SKILLS.md`, and `installed-skills.json`. | Sync, update, and build replace this directory. Check compares it with the expected output. |
| `.claude/skills/<name>/` and `.agents/skills/<name>/` | Installed copies of skills, one folder per skill. | Sync, update, and build write, replace, and remove the skill folders Code Rules owns, and leave every other folder alone. Check compares them with the expected output. See [Who owns the target folders](#who-owns-the-target-folders). |

Replacement includes removing files that no longer belong in the output, such as removed rules, old index pages, unused library folders, and the installed folders of skills the project no longer installs. Files you add or edit inside `vendor/`, `generated/`, or an installed skill folder can be replaced or removed. Keep your changes in configuration and `local/`.

For the complete directory layout, see [Project files](/reference/files/).

## Read the command's result

Commands print human-readable output by default. Add `--json` when another tool needs one structured response:

```sh
code-rules project check --json
```

Sync, update, and build report counts and sorted lists of added, changed, and removed paths. JSON output includes those lists in `added`, `changed`, and `removed`. When the command first [recovered an interrupted one](#recover-from-an-interrupted-update), it says so before the counts, which don't include the files the recovery restored or finished, and JSON output sets `recovered` to `true`, also for an update's preview, which writes nothing of its own; a declined update says it in its message.

- Sync and update paths start with `vendor/`, `generated/`, `.claude/skills/`, or `.agents/skills/`. Paths in a target folder are relative to the project root; the rest are relative to the Code Rules directory. Update also lists `config.yaml` when it writes a pin or an exclusion, and the files of a fork it replaces, and both list a `local/<group-id>/_group.yaml` they add.
- Build paths are relative to `generated/`, except paths in a target folder, which start with `.claude/skills/` or `.agents/skills/` and are relative to the project root.

When the target folders are linked, as [Linked target folders](#linked-target-folders) describes, each installed file is listed once, under the folder the link points to.

`code-rules project update` also reports each rule's and skill's change, versions, and summary. There is no separate structured summary of added or removed groups. To see which library revisions changed, review the source records and generated [provenance records](/reference/provenance/).

Check reports `status` and `problems`, including each problem's path and suggested repair command. It verifies generated guidance, installed skills, and the managed Code Rules guide without writing any of them. A problem in a target folder has a path relative to the project root, starting with `.claude/skills/` or `.agents/skills/`.

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
| An installed skill folder is missing, edited, or unexpected, or `generated/installed-skills.json` is stale, and imported files are valid. | Run `code-rules project build`. If the folders are missing because `.gitignore` excludes them, also add an exception and commit them; see [Who owns the target folders](#who-owns-the-target-folders). |
| Configuration, rule metadata, or skill metadata is invalid. | Correct the reported input, then retry the appropriate command. |

Run `code-rules project check` again after repairing the problem.

### How stored imports are checked

For each library, Code Rules records each imported rule's version and file checksums in `vendor/<source-name>/_source.json`. A **checksum** detects whether a file's contents differ from the recorded copy.

Build and check work offline. They reject missing, changed, or unexpected imported files, invalid source records, version records that don't cover exactly the imported rules and skills, and selections that no longer match your configuration, including the configuration a preset recorded in `vendor/_preset/` contributes. Sync fetches the recorded library files again, including replacing locally modified copies.

For skills, check also fails when:

- An installed copy differs from what sync would write, such as after a hand edit to a file in `.claude/skills/build-feature/`.
- A folder that `generated/installed-skills.json` lists is missing, such as because `.gitignore` excludes it and it was never committed.
- `generated/installed-skills.json` or `generated/SKILLS.md` doesn't match the skills the configuration installs.
- Two skills the project installs have the same name, or a target folder has a folder Code Rules would install a skill into but doesn't own.

Check never inspects target folders Code Rules doesn't own.

Checksums detect changes relative to the stored record. They cannot establish that files are authentic if someone also changed that record. For record fields and rule origins, see [Provenance](/reference/provenance/).

## Who owns the target folders

The target folders, `.claude/skills/` and `.agents/skills/`, are shared: people and other tools install skills there too. Code Rules owns only the skill folders it installed, as `generated/installed-skills.json` records them, and treats every other folder as someone else's:

- Sync, update, and build write each installed skill to `<target>/<name>/`, replacing the whole folder, and remove the folders of skills the project no longer installs, including after you remove a target from `skillTargets`. An owned folder you edited by hand is replaced or removed like any other output.
- They never change, remove, or read the contents of a folder Code Rules didn't install.
- They create a target folder, such as `.claude/skills/`, when they first install a skill into it, and remove it when they leave it empty.

Two errors protect folders Code Rules doesn't own. Each fails before writing anything:

| Error | Meaning and next step |
| --- | --- |
| `skill-folder-not-managed` | A target already has a folder with the name of a skill the project installs, and Code Rules didn't install it, such as a hand-copied skill or another tool's. To let Code Rules manage the skill, move or delete that folder and run the command again. To keep the folder, [exclude](/reference/configuration/#exclude-or-replace-a-skill) the skill instead. |
| `skill-name-conflict` | Two skills the project installs have the same name, such as `fabrica:skills/review` and `pstack:skills/review`, or an imported skill and a local skill. Installed folders are named by skill name, so both can't be installed. Exclude one of them. Code Rules never renames a skill to make room. |

When `generated/installed-skills.json` is missing or was changed outside Code Rules, Code Rules can't tell which folders it owns, so it treats the folders it would write as not managed. Restore the record from Git, such as with `git checkout -- .code-rules/generated/installed-skills.json`, then run the command again.

Commit the target folders. A `.gitignore` that excludes them, such as with `/.claude/`, leaves fresh checkouts without the installed skills, and `code-rules project check` reports the missing folders. Change the line to `/.claude/*` and add the exception `!/.claude/skills/`; Git can't re-include a folder inside an excluded folder. See [Commit the installed skills](/guides/install-skills/#commit-the-installed-skills).

### Linked target folders

Some projects make one target folder a symbolic link to the other, such as `.agents/skills` linked to `.claude/skills`, so that every agent reads the same files. Code Rules detects the link and writes each skill once, into the folder the link points to. The link itself, and a link from a parent folder such as `.agents` to `.claude`, must stay inside the project; a link that points outside the project root is refused. Every other symbolic link in a target folder is refused, as elsewhere in the project.

## Keep a local rule's group

A local rule needs its group's metadata, from `local/<group-id>/_group.yaml` or an imported library. Sometimes only an imported rule supplied it, such as when you [fork](/reference/cli/#fork-a-library-rule) an individually selected rule and the library later retires it, or you deselect or remove the library. When a sync or update would leave your local rules in a group with no metadata, it writes the group's `_group.yaml` to `local/<group-id>/_group.yaml` in the same step that replaces `vendor/` and `generated/`, lists the file as added, and prints a warning naming it. The metadata comes from the first source whose record in `vendor/` lists the group's metadata, configured sources first: from the library release that now supplies that source's library-wide files, when it still has the group, and otherwise from the last imported copy in `vendor/`, which is also what a source you removed from the configuration supplies. The file is then yours to edit like any local group metadata. A vendored copy must match the checksum its source record lists: when it was changed or deleted since sync imported it, the command fails without changing anything. Restore it by running `code-rules project sync` with the previous configuration, then make your change and sync again, or write `local/<group-id>/_group.yaml` yourself.

## Recover from an interrupted update

Sync, update, and build keep the previous output while installing replacement files. If installation fails, the command restores the previous directories. The installed skill folders in the target folders are part of the same step: one lock and one recovery journal cover them together with `vendor/` and `generated/`, so recovery restores or finishes them all together. If the process stops during replacement, the next sync, update, or build recovers the previous output before starting its own work. Update installs the pins and exclusions it writes to `config.yaml`, and each fork it [replaces with the newest version](/reference/cli/#replace-a-fork-with-the-newest-version), in the same step, so recovery restores or finishes `config.yaml`, the replaced local rules and their asset directories, `vendor/`, and `generated/` together. Likewise, recovery keeps or removes a `local/<group-id>/_group.yaml` that sync or update added together with `vendor/` and `generated/`. A failed or interrupted update may leave empty directories under `local/`, such as a group's first `assets/` directory that [replacing a fork](/reference/cli/#replace-a-fork-with-the-newest-version) created. They're harmless: every command ignores them, and you can delete them.

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

Sync, update, and build prepare and validate the complete replacement, including every installed skill folder, before installing it. Immediately before replacement, they check that the original input and output files, including the owned skill folders, have not changed. If another process changed them, the update stops.

Code Rules uses these temporary directories beside your configuration, for the target folders too:

| Directory | Purpose |
| --- | --- |
| `.code-rules-lock` | Allows only one cooperating Code Rules command to write at a time. |
| `.code-rules-transaction` | Holds the recovery journal and previous output during replacement. The journal records the update's progress. |
| `.code-rules-cleanup` | Marks a completed update whose backups can be deleted. |

After completing an update, Code Rules renames the transaction directory to the cleanup directory before deleting backups. It also discards abandoned staging files when no journal or backups exist.

A journal that covers target folders uses a newer format than earlier versions of Code Rules read. An earlier version that finds one stops and asks you to upgrade Code Rules, rather than reporting a damaged journal.

Avoid editing managed directories during an update. Replacing `vendor/`, `generated/`, the installed skill folders, and, for update, `config.yaml` takes separate filesystem operations, so another program can briefly see a mixture of old and new files. Check detects an active update instead of accepting mixed output as consistent.

Cancellation can stop work before replacement starts. Once replacement begins, the command finishes or rolls back before returning.

### Filesystem requirements and limits

These protections apply to cooperating Code Rules commands on a local filesystem with normal file-renaming behavior. They do not guarantee recovery from power loss, protection against hostile concurrent file changes, or locking on network filesystems.

All operations reject symbolic links and special files within the directories they inspect, except a link between the target folders, as [Linked target folders](#linked-target-folders) describes. They also reject regular files with multiple hard links. Ordinary aliases in the project's parent path, such as macOS `/tmp`, are supported.

| Input resource | Limit |
| --- | --- |
| Each file | 64 MiB. |
| Each directory tree | 256 MiB and 30,000 entries. |
| Each path | 64 segments. |

Imports have additional, stricter [limits](/reference/imports/#import-limits-and-version-errors). Output paths must remain distinct after normalization for case-insensitive filesystems.
