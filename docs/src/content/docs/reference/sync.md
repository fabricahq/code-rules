---
title: "Sync and recovery"
description: "When to run sync, build, or check, which files they change, and how to recover from problems."
---

The `code-rules project sync` command imports your project's library files, regenerates the guidance your agents read, and installs your skills. Run it after adding a source, changing its selected rules or skills, pins, or `ref`, or to restore imported files.

:::note[Preview]
Skills come in the next Code Rules release. A skill links only to the cited rules the project already includes: sync doesn't yet take a cited rule the project's rule `include` leaves out.
:::

Sync imports the rule versions recorded for each source, so it never adopts newer versions on its own. To move to newer versions, run `code-rules project update`, which previews every change and applies it only after you confirm, offering to pin each major change or retirement instead. See [Update rules](/guides/update/).

This page explains how to run sync, what it changes, and what to do when files are missing, outdated, or left by an interrupted update. It also explains when `code-rules project build` or `code-rules project check` is enough. For how Code Rules selects and combines library rules, see [How imports work](/reference/imports/).

## Choose the right command

- **`code-rules project sync`**: Import the recorded rule versions, or choose versions for new and changed sources. Sync validates the library files and regenerates agent guidance.
- **`code-rules project update`**: Preview newer versions, new rules, and retirements, and apply them after you confirm. Pinned rules and sources that use `ref` don't move. Update then does everything sync does.
- **`code-rules project build`**: Apply local rule changes, exceptions, or authored skills using the library files you already have. Build regenerates guidance and reinstalls skills without contacting a repository.
- **`code-rules project check`**: Find out whether generated guidance, installed skills, and the managed Code Rules guide are up to date. Check validates stored inputs and output, reports problems, and leaves files unchanged.

You do not need to run build after a successful sync or update; both already generate the guidance. Run check when you want to verify consistency without making changes.

## Run from your project root

After [installing Code Rules](/start-here/install/) and [setting up your project](/start-here/set-up-project/), run:

```sh
code-rules project sync
```

In a Git repository, project commands find the nearest repository root and use its `.code-rules/config.yaml`. Outside Git, run commands from the project root. Custom configuration locations are not supported. The paths below are relative to `.code-rules/`.

## Which files change

| File or directory | What it contains | What the commands do |
| --- | --- | --- |
| `config.yaml` | Your sources, selected rules and skills, pins, exceptions, and skill targets. | Sync, build, and check read it without changing it. Update reads it, and writes a pin when you keep a rule at its current version, or an exclusion when you decline a new rule. |
| `README.md` | The managed Code Rules guide. | Init, build, and sync refresh an older, unedited guide. Check verifies it without changing it. |
| `rules/` | Groups, rules, and replacements you author for this project. | Sync, build, and check preserve these files. Sync and update add `rules/<group-id>/_group.yaml` only when your authored rules in a group would otherwise lose the group's metadata; see [Keep an authored rule's group](#keep-an-authored-rules-group). |
| `assets/` | Supporting files that several of your rules share. | Sync, build, and check read them without changing them. |
| `skills/` | Skills you author for this project, one folder per skill. | Sync, build, and check read them without changing them. |
| `vendor/` | Original files copied from the selected rule versions. | Sync and update replace this directory. Build and check validate it without changing it. |
| `generated/` | Rules, reading indexes, and the skills index for your agents. | Sync, update, and build replace this directory. Check compares it with the expected output. |
| `../.claude/skills/<name>/`, `../.agents/skills/<name>/` | Installed copies of your skills, in each [skill target](/reference/configuration/#skill-targets). | Sync, update, and build replace the folders Code Rules installed and remove those of skills no longer included. They never touch other folders there. Check compares the folders with what build would write. |

Replacement includes removing files that no longer belong in the output, such as removed rules, old index pages, and unused library folders. Files you add or edit inside `vendor/` or `generated/` can be replaced or removed. Keep your changes in configuration, `rules/`, and `assets/`. Every project command fails while a `local/` directory exists, where earlier versions kept authored rules; its message shows where to move each file.

For the complete directory layout, see [Project files](/reference/files/).

## Read the command's result

Commands print human-readable output by default. Add `--json` when another tool needs one structured response:

```sh
code-rules project check --json
```

Sync, update, and build report counts and sorted lists of added, changed, and removed paths. JSON output includes those lists in `added`, `changed`, and `removed`. When the command first [recovered an interrupted one](#recover-from-an-interrupted-update), it says so before the counts, which don't include the files the recovery restored or finished, and JSON output sets `recovered` to `true`, also for an update's preview, which writes nothing of its own; a declined update says it in its message.

- Sync and update paths start with `vendor/` or `generated/`. Update also lists `config.yaml` when it writes a pin or an exclusion, and both list a `rules/<group-id>/_group.yaml` they add.
- Build paths are relative to `generated/`.
- Installed skill files are listed by their path relative to the same directory, such as `../.claude/skills/build-feature/SKILL.md` for sync.

`code-rules project update` also reports each rule's change, versions, and summary. There is no separate structured summary of added or removed groups. To see which library revisions changed, review the source records and generated [provenance records](/reference/provenance/).

Check reports `status` and `problems`, including each problem's path and suggested repair command. It verifies generated guidance, installed skills, and the managed Code Rules guide without writing any of them.

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
| An installed skill is missing, edited, or no longer installed. | Run `code-rules project build`. It replaces edits to installed copies; see [Check installed skills](#check-installed-skills). |
| Configuration or rule metadata is invalid. | Correct the reported input, then retry the appropriate command. |

Run `code-rules project check` again after repairing the problem.

### How stored imports are checked

For each library, Code Rules records each imported rule's version and file checksums in `vendor/<source-name>/_source.json`. A **checksum** detects whether a file's contents differ from the recorded copy.

Build and check work offline. They reject missing, changed, or unexpected imported files, an executable bit that differs from the record, invalid source records, rule version records that don't cover exactly the imported rules, and library selections that no longer match your configuration. Sync fetches the recorded library files again, including replacing locally modified copies.

Checksums detect changes relative to the stored record. They cannot establish that files are authentic if someone also changed that record. For record fields and rule origins, see [Provenance](/reference/provenance/).

## Skills

Sync takes every skill of a source from the one revision its source record locks, the library release or the commit that `ref` names, so skills that call each other always match. It never moves a source for a skill: when `skills.include` newly names a skill that revision doesn't have, sync fails with `skill-not-in-revision` and writes nothing. For a library source, run the `code-rules project update <source>` the message names, which moves the source to its newest library release; for a source with `ref`, set `ref` to a revision that has the skill. A source whose repository has no `release/<number>` tags needs a `ref`, and fails with `ref-required` without one.

Sync, update, and build then install each included skill, and each skill in `.code-rules/skills/`, into every skill target, in the same step that replaces `vendor/` and `generated/`, so recovery restores or finishes them together. Each installed copy is a folder named by the skill's install name, the `name` in its `SKILL.md`, such as `.claude/skills/build-feature/`, with the skill's files, their executable bits, and a marker line after the `SKILL.md` frontmatter that names the skill and, for a source's skill, the commit it came from. Commit the installed copies with the rest of the project, and don't edit them: the next sync or build replaces them. To change a skill you author, edit `.code-rules/skills/<name>/` and run `code-rules project build`.

`generated/installed-skills.json` records the folders Code Rules installed. A later command replaces or removes only those, and removes a skill target's `skills/` folder once it's empty. When a folder it would install already exists and Code Rules didn't install it, such as a personal skill with the same name, the command fails with `skill-folder-not-managed` and changes nothing: move or delete that folder, or stop including the skill. Two skills with the same install name fail with `skill-name-clash`. A skill target folder that is a symbolic link fails with `unsafe-path`.

Before installing a skill you author in `.code-rules/skills/`, sync, update, and build check its frontmatter, and fail on the first problem, naming its `SKILL.md`: `name` must equal the folder's name, `syllabus-skill-type` must be `playbook` or `review`, no other `syllabus-` key may appear, and each line of `syllabus-rules-used` and `syllabus-skills-used` must be an ID, once, in the form [Skill](/concepts/skill/#rules-a-skill-cites) describes. A bare rule ID must name a rule you author, and a source-qualified one a source in the configuration that's a library and, when that source includes the rule's whole group, a rule the group has or retired, so a misspelled rule ID fails. Sync keeps only the rules a source includes, so a cited rule in a group the source doesn't include in full can't be checked; the skill lists it as not included. A bare skill ID in `syllabus-skills-used` must name another skill you author, and a source-qualified one, such as `pstack:skills/how`, a skill that source installs; authored skills can't use each other in a cycle. A source's skills aren't checked again: their library checked them, and a skill from a repository that isn't a library installs as it is.

Sync and update install every skill a source's included skill uses, by its `syllabus-skills-used` and the source's [`skills.uses`](/reference/configuration/#declare-the-skills-a-skill-uses), and the skills those use, from the same revision. They fail, writing nothing, when a used skill isn't in the revision, naming the skill and the skill it uses, and when skills use each other in a cycle. The source record lists why each skill is installed, and build and check ask for sync when `skills.uses` or a vendored `syllabus-skills-used` no longer matches it.

When a skill cites rules in `syllabus-rules-used`, its installed `SKILL.md` ends with a generated **Rules this skill applies** section. It lists each cited rule with its version and a link to its file in `generated/rules/`, or, when the project replaces it, the authored rule that does. It says instead when the project excluded the rule, with the exclusion's reason, when the project doesn't include it, and when its library retired it. Cited rules never join `generated/RULES.md` or the group pages. A library's skill cites rules in its own library, so they resolve to the source that supplies the skill. When a skill you author cites a rule its library retired, sync, update, and build warn and install it anyway, so the update that retires the rule can apply; check fails until you remove the citation.

`generated/SKILLS.md` lists every skill the project installs: what it does, where it came from, where it's installed, how many rules it applies, the skills it uses, and, for a source's skill, whether `skills.include` selects it or other skills use it. `generated/RULES.md` points to it. With no skill targets, sync records and lists the skills, with the rules each one applies, installs no copies, and warns that it installed none.

### Check installed skills

`code-rules project check` compares every skill target folder that `generated/installed-skills.json` lists, or that the project would install, with what build would write, and reports a problem for each folder that differs, by its path relative to the Code Rules directory, such as `../.claude/skills/build-feature`, with the skill's ID:

| Problem kind | Meaning |
| --- | --- |
| `missing-skill` | The project installs the skill, and the folder is missing. |
| `modified-skill` | The folder's files, or a file's executable bit, differ from the skill's source. |
| `unexpected-skill` | The record lists the folder, but the project no longer installs the skill. |

Run `code-rules project build` to repair them. It replaces the installed copies, including any edits to them, so move edits you want to keep into the skill's source first.

## Keep an authored rule's group

An authored rule needs its group's metadata, from `rules/<group-id>/_group.yaml` or an imported library. Sometimes only an imported rule supplied it, such as when you [fork](/reference/cli/#fork-a-library-rule) an individually selected rule and the library later retires it, or you deselect or remove the library. When a sync or update would leave your authored rules in a group with no metadata, it writes the group's `_group.yaml` to `rules/<group-id>/_group.yaml` in the same step that replaces `vendor/` and `generated/`, lists the file as added, and prints a warning naming it. The metadata comes from the first source whose record in `vendor/` lists the group's metadata, configured sources first: from the library release that now supplies that source's library-wide files, when it still has the group, and otherwise from the last imported copy in `vendor/`, which is also what a source you removed from the configuration supplies. The file is then yours to edit like any authored group metadata. A vendored copy must match the checksum its source record lists: when it was changed or deleted since sync imported it, the command fails without changing anything. Restore it by running `code-rules project sync` with the previous configuration, then make your change and sync again, or write `rules/<group-id>/_group.yaml` yourself.

## Recover from an interrupted update

Sync, update, and build keep the previous output while installing replacement files. If installation fails, the command restores the previous directories. If the process stops during replacement, the next sync, update, or build recovers the previous output before starting its own work. Update installs the pins and exclusions it writes to `config.yaml`, and each fork it [replaces with the newest version](/reference/cli/#replace-a-fork-with-the-newest-version), in the same step, so recovery restores or finishes `config.yaml`, the replaced authored rules and their asset directories, `vendor/`, and `generated/` together. Likewise, recovery keeps or removes a `rules/<group-id>/_group.yaml` that sync or update added together with `vendor/` and `generated/`. A failed or interrupted update may leave empty directories under `rules/`, such as a group's first `assets/` directory that [replacing a fork](/reference/cli/#replace-a-fork-with-the-newest-version) created. They're harmless: every command ignores them, and you can delete them.

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

Sync, update, and build prepare and validate the complete replacement before installing it. Immediately before replacement, they check that the original input and output files have not changed. If another process changed them, the update stops.

Code Rules uses these temporary directories beside your configuration:

| Directory | Purpose |
| --- | --- |
| `.code-rules-lock` | Allows only one cooperating Code Rules command to write at a time. |
| `.code-rules-transaction` | Holds the recovery journal and previous output during replacement. The journal records the update's progress. |
| `.code-rules-cleanup` | Marks a completed update whose backups can be deleted. |

After completing an update, Code Rules renames the transaction directory to the cleanup directory before deleting backups. It also discards abandoned staging files when no journal or backups exist.

Avoid editing managed directories during an update. Replacing `vendor/`, `generated/`, and, for update, `config.yaml` takes separate filesystem operations, so another program can briefly see a mixture of old and new files. Check detects an active update instead of accepting mixed output as consistent.

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
