---
title: "How imports work"
description: "How library rules become project guidance, what happens during updates, and what imports support."
---

An **import** copies selected rules from a shared library into your project.

Imports let you reuse your team's engineering practices across projects without writing and maintaining the same rules in each one. You can also import rules from third-party libraries whose engineering practices you want to adopt. Each project can choose which rules to use, add its own rules, and replace imported rules to fit its needs.

This page explains which files Code Rules imports and how it combines imported rules with your local rules and exceptions. It also covers the checks that protect your project during an update and the limits on what you can import. For step-by-step instructions, see [Import rules](/guides/select-rules/).

## From library rules to project guidance

When you run `code-rules project sync`, Code Rules:

1. Reads your configuration to find the libraries, groups, and rule versions you selected. A **group** collects related rules, such as testing practices or TypeScript conventions.
2. Copies the selected library files into your project. The copy of one library is called a **snapshot**; each rule in it comes from the library release that published its version.
3. Combines the imported rules with your local rules and configured exceptions, then generates files for your agents to read.

These files live in the **Code Rules directory**, `.code-rules/` at the project root:

| Directory | What it contains |
| --- | --- |
| `vendor/` | Original files copied from the libraries you selected. |
| `local/` | Rules you author for this project, including replacements for imported rules. |
| `generated/` | The rules and reading indexes your agents use. |

For the complete layout, see [Project files](/reference/files/). Once the library files are stored, `code-rules project build` can regenerate guidance without network access.

## What gets copied

Each library has a **source name** in your configuration, such as `team`. Code Rules stores that library's imported files under `vendor/team/`.

The copy includes:

- Rules from the selected groups and the individually selected rules, including rules your project excludes or replaces.
- Supporting files, such as images and examples, from the library's designated asset directories.
- Group metadata, which describes each group and when to read it.
- The library manifest, `rule-library.yaml`, and its declared license and notice files.

Keeping the original rules lets you review library changes even when your project uses a replacement. Make project-specific changes in `local/`; syncing replaces the imported files.

The snapshot contains no Git history or `.git` directory. Code Rules reads the original files through temporary Git storage and removes that temporary storage afterward.

## How Code Rules selects the rules your agents read

Your configuration selects groups from each library, and optionally individual rules. You can name groups individually or use one of these selectors:

| Selection | Groups to import |
| --- | --- |
| `"*"` | Every technology and practice group. |
| `"practices/*"` | Every practice group. |
| `"techs/*"` | Every technology group. |

Code Rules finds the matching groups in the imported library before applying your exceptions. It records both your selection and the groups actually imported. Invalid groups and rules without group metadata cause an error instead of being silently skipped.

Code Rules then decides which rules are **active**, meaning included in the generated guidance:

1. Starts with the imported rules, from your selected groups and your individually selected rules, and discovers local groups from their `_group.yaml` files.
2. Removes rules you explicitly excluded.
3. Adds the local rules your exclusions name as `replacedBy`, in place of the rules they exclude.
4. Adds your remaining local rules.

Each rule's ID includes its source name. For example, `team:practices/testing/check-retries` identifies the `check-retries` rule from the `team` library. This keeps rules from different libraries distinct, even when their filenames match.

A replacement contributes its complete local definition: ID, title, metadata, body, attribution, and links to supporting files. It must belong to the same group as the rule it replaces. Each local rule appears only once, even when used as a replacement.

Code Rules does not read rule text to detect contradictory instructions. Two libraries can supply conflicting rules, and both remain active unless you configure an exclusion or replacement. The order of libraries in your configuration does not establish priority. See [Resolve conflicting rules](/guides/conflicting-guidance/).

## Where agents read the result

Code Rules writes the active rules and reading indexes to `generated/`. Agents start at `generated/RULES.md`, open relevant groups, and read the applicable rules in full.

The generated files also include library summaries, retained license files, and **provenance**: records of where rules came from and which rules they replaced. Replacement targets and reasons stay in configuration and provenance, outside the rule guidance. Group descriptions remain labeled by source so you can see which library supplied them.

The same files support implementation and review. Your project chooses how to check that agents follow the rules; importing does not enforce compliance.

## What changes when you update

Each source's `vendor/<source-name>/_source.json` records the version of every rule it imported. Running `code-rules project sync` again imports those same versions, so every checkout of the project gets the same rules. `code-rules project sync` chooses a rule's version only for a newly selected rule, a new source or changed repository, a pin you add or change, or a `ref` you add, change, or remove; see [project sync](/reference/cli/#project-sync).

`code-rules project update` previews each rule's newest version, and applies it once you confirm, unless the rule is [pinned](/reference/configuration/#pin-a-rule). Sources that use `ref` don't move.

With the same configuration and recorded versions, an import produces the same paths and file contents. With unchanged imported files, local rules, tool version, and rendering options, a build produces the same generated guidance. Reordering libraries, groups, or rules in configuration does not change their generated order.

`code-rules project update` reports each changed rule with its change, versions, and summary, and applies the changes only after you confirm. Both commands report changed files, including group metadata and version records.

### How rule versions are resolved

The newest library release is the one with the highest `release/<number>` tag. Each [library release](/reference/rule-versions/#library-releases) tag's message records every rule's version and the changes that library release published.

To choose a rule's version, Code Rules reads the rule's history from the release records in the `release/<number>` tags, and picks the newest version, or the one the rule is pinned to. It then imports the rule's [Markdown file and asset directory](/reference/rule-versions/#what-a-version-covers) from the tagged commit of the library release that published that version. Library-wide files, including the shared files rules link to, come from the newest library release among the imported rule versions, or from the newest library release when the source imports no rules.

A rule absent from a library release was retired, and the release record's `retired` entry records why. A rule the project pinned before its retirement keeps importing its pinned version.

To find versions, Code Rules lists only `release/` tags, fetches their messages without the library's history, and fetches only the files it imports. It records each imported rule's version, library release, and commit in `_source.json` and generated provenance, and shows the version in generated guidance.

### Tracing rules to their source

Code Rules records the versions you requested, and each imported rule's exact version and commit. See [Provenance](/reference/provenance/) for these records.

Links to original files on GitHub.com and GitLab.com use the imported commit, so moving a tag does not change their destination. For other Git hosts, links point to the stored files; provenance retains the repository address and commit.

Code Rules preserves rule attribution in both imported and generated files and keeps attribution links valid. It verifies and retains the license and notice files declared in the library manifest, includes them in integrity checks, and reports their changes during updates.

Generated rules link to retained terms under `generated/libraries/<source-name>/licenses/`. See [License rules](/guides/license-rules/) for how library authors declare those files.

## Checks before updating your files

Code Rules fetches and validates all selected libraries before replacing your project's imported files or generated guidance. If any library cannot be fetched or validated, your previous complete set of rules stays in place.

Validation rejects:

- Invalid or reserved source names, repeated repositories, and duplicate rule IDs that include the same source name.
- Missing groups, or rules named in an exclusion, pin, or `rules` entry that the source doesn't import. An entry naming a rule the library retired produces a warning instead, when the source would otherwise import that rule: its group is selected, it is listed in `rules`, or the last sync imported it.
- A missing `replacedBy` file, or one local file named as the replacement for more than one rule.
- Invalid metadata, unsafe file paths, and symbolic links.

Selected library rules must pass validation even if you exclude or replace them. File checks also ensure that paths stay within their allowed directories.

Code Rules detects concurrent writes and interrupted updates so a mixture of old and new output cannot pass a consistency check. For file replacement and recovery behavior, see [Sync and recovery](/reference/sync/).

## Git access and supported files

Imports require Git 2.30 or later on macOS or Linux. Code Rules accepts HTTPS and SSH repository addresses, including scp-style SSH addresses and nested repository paths. See [Repository addresses](/reference/configuration/#repository-addresses) for accepted formats.

Code Rules uses your Git credentials and certificate and host-key verification settings. It does not store credentials in configuration or provenance. A valid repository address does not guarantee that the repository is reachable or appropriate for your network.

Git URL rewrites still apply. If a rewrite uses another protocol, your Git configuration must explicitly allow that protocol. Executable `ext` helpers are always disabled. Code Rules passes Git arguments separately and applies the same path checks and resource limits across hosts.

Code Rules reads original Git file contents without checking out the library. It does not run library scripts, Git hooks, or checkout filters. Selected symbolic links, submodules, and Git LFS pointers are unsupported; Code Rules does not fetch submodule contents.

### Supporting files

Code Rules copies supporting material from [two asset locations](/reference/rule-format/#supporting-assets):

- **A rule's own assets:** the adjacent `assets/<rule-name>/` directory. Code Rules copies this directory in full when it imports the rule.
- **Shared assets:** files in the library-root `assets/` directory. Code Rules copies the files a selected rule or its Markdown assets link to, including files they link to in turn, from the library release that supplies the source's library-wide files.

Markdown links, images, and reference links must point to files within the allowed locations. Missing files and links into another rule's private assets cause an error. Code Rules preserves external URLs as links without downloading their contents.

A rule or Markdown attachment cannot link to another rule document on disk, even if that rule is also selected. Each rule must work independently because projects can exclude or replace it. Put shared supporting explanations in the library's shared assets directory.

Within a selected group, Markdown files outside asset directories count as rules, including files in nested folders. Declared license and notice files and the group-root `README.md` are exceptions. Put other supporting Markdown, such as `_README.md`, in an asset directory.

Markdown assets, license files, and notice files must use UTF-8 text. Other assets can be binary files; Code Rules preserves their original bytes.

## Import limits and version errors

Imports reject paths that become identical when letter case is ignored and Unicode names are normalized to NFC. This includes directory names and prevents ambiguous paths across filesystems.

Each library import has these limits:

| Resource | Limit |
| --- | --- |
| Fetching and finding rule versions | 120 seconds per library. |
| Entries in the Git file tree | 10,000 entries. |
| Git file-tree listing | 8 MiB. |
| Each retained file | 8 MiB. |
| All retained files combined | 64 MiB. |
| Git's listing of release tags | 8 MiB and 20,000 records, including extra records Git uses to identify commits behind annotated tags. |
| Each release tag | 8 MiB. |
| Each [release record](/reference/rule-versions/#release-record) | 10,000 entries each in `rules`, `changes`, and `retired`, and 20,000 in `libraryFiles`. A record over a limit fails with `invalid-release-tag`. |

The retained-file limits apply after fetching. They do not cap network traffic or Git's temporary disk use. Finding rule versions uses the same Git access settings and deadline as fetching.

When finding the newest library release, Code Rules can report:

| Error | Meaning and next step |
| --- | --- |
| `releases-not-found` | The library has no `release/<number>` tags, because it hasn't published its first library release. Ask the maintainer to publish a library release, or import a commit with the source's `ref`. |
| `version-not-found` | A pin names a version the rule never published, or the tag or commit in `ref` doesn't exist. Check the pin or `ref`. |
| `ref-is-branch` | The source's `ref` names a branch. `ref` accepts only a tag or a full commit SHA, so every import can be reproduced. |
| `unsupported-release-record` | A library release's [release record](/reference/rule-versions/#release-record) uses a newer format than this Code Rules reads, because a later Code Rules published it. Upgrade Code Rules. |

When Code Rules can't read the library's repository, it can report:

| Error | Meaning and next step |
| --- | --- |
| `connection-failed` | Git couldn't reach the repository's host, such as when the host name doesn't resolve, the connection is refused or times out, or TLS fails. The message quotes Git's reason. Check the repository address and your network connection. |
| `not-found-or-no-access` | The host answered, but the repository doesn't exist or your Git credentials can't read it; servers report both the same way. Check the address and your credentials. |
| `object-fetch-refused` | The server refused to send a file by its object ID, which Code Rules needs to read one version of each rule without downloading the whole repository. GitHub.com and GitLab.com allow it; a self-hosted server needs Git protocol version 2 or `uploadpack.allowAnySHA1InWant`. |
| `git-failed` | Another Git failure. The message quotes Git's last error line. |

Messages that quote Git replace credentials, such as a password in the repository address or a token, with `[redacted]`.
