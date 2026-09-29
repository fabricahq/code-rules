---
title: "How imports work"
description: "How library rules become project guidance, what happens during updates, and what imports support."
---

An **import** copies selected rules from a shared library into your project.

Imports let you reuse your team's engineering practices across projects without writing and maintaining the same rules in each one. You can also import rules from third-party libraries whose engineering practices you want to adopt. Each project can choose which rules to use, add its own rules, and replace imported rules to fit its needs.

This page explains which files Code Rules imports and how it combines imported rules with your local rules and exceptions. It also covers the checks that protect your project during an update and the limits on what you can import. For step-by-step instructions, see [Import rules](/guides/select-rules/).

## From library rules to project guidance

When you run `code-rules project sync`, Code Rules:

1. Reads your configuration to find the libraries, revisions, and groups you selected. A **group** collects related rules, such as testing practices or TypeScript conventions.
2. Copies the selected library files into your project. Each copy comes from one Git commit and is called a **snapshot**.
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

- Rules from the selected groups, including rules your project excludes or replaces.
- Supporting files, such as images and examples, from the library's designated asset directories.
- Group metadata, which describes each group and when to read it.
- The library manifest, `rule-library.yaml`, and its declared license and notice files.

Keeping the original rules lets you review library changes even when your project uses a replacement. Make project-specific changes in `local/`; syncing replaces the imported files.

The snapshot contains no Git history or `.git` directory. Code Rules reads the original files through temporary Git storage and removes that temporary storage afterward.

## How Code Rules selects the rules your agents read

Your configuration selects groups from each library. You can name groups individually or use one of these selectors:

| Selection | Groups to import |
| --- | --- |
| `"*"` | Every technology and practice group. |
| `"practices/*"` | Every practice group. |
| `"techs/*"` | Every technology group. |

Code Rules finds the matching groups at the selected library revision before applying your exceptions. It records both your selection and the groups actually imported. Invalid groups and rules without group metadata cause an error instead of being silently skipped.

Code Rules then decides which rules are **active**, meaning included in the generated guidance:

1. Starts with the imported rules from your selected groups and discovers local groups from their `_group.yaml` files.
2. Removes rules you explicitly excluded.
3. Substitutes your local rules for imported rules you explicitly replaced.
4. Adds your remaining local rules.

Each rule's ID includes its source name. For example, `team:practices/testing/check-retries` identifies the `check-retries` rule from the `team` library. This keeps rules from different libraries distinct, even when their filenames match.

A replacement contributes its complete local definition: ID, title, metadata, body, attribution, and links to supporting files. It must belong to the same group as the rule it replaces. Each local rule appears only once, even when used as a replacement.

Code Rules does not read rule text to detect contradictory instructions. Two libraries can supply conflicting rules, and both remain active unless you configure an exclusion or replacement. The order of libraries in your configuration does not establish priority. See [Resolve conflicting rules](/guides/conflicting-guidance/).

## Where agents read the result

Code Rules writes the active rules and reading indexes to `generated/`. Agents start at `generated/RULES.md`, open relevant groups, and read the applicable rules in full.

The generated files also include library summaries, retained license files, and **provenance**: records of where rules came from and which rules they replaced. Replacement targets and reasons stay in configuration and provenance, outside the rule guidance. Group descriptions remain labeled by source so you can see which library supplied them.

The same files support implementation and review. Your project chooses how to check that agents follow the rules; importing does not enforce compliance.

## What changes when you update

Each source's `vendor/<source-name>/_source.json` records the commit it imported. Running `code-rules project sync` again imports that same commit, so every checkout of the project gets the same rules. `code-rules project sync` resolves a source's revision again only when the source is new or you changed its repository or `ref`.

`code-rules project update` moves each source without a `ref` to the library's newest release. Sources pinned with a `ref` stay where they are.

With the same configuration and commit, an import produces the same paths and file contents. With unchanged imported files, local rules, tool version, and rendering options, a build produces the same generated guidance. Reordering libraries, groups, or rules in configuration does not change their generated order.

`code-rules project update` reports each changed rule with its change, versions, and summary, and asks you to accept major changes and retirements of rules the project uses. Both commands report changed files, including group metadata and revision records.

### How rule versions are resolved

Every library keeps the branch `code-rules/released` pointed at its newest [release commit](/reference/rule-library-format/#release-commits). When a source has no `ref`, Code Rules reads that branch to find the newest release.

A rule's version at a release commit is its highest version tag on that commit or an earlier release commit. Every release commit contains exactly the released content, so this version describes the imported file. A rule absent from the release commit was retired, and its `<rule-id>@retired` tag records why; its last version remains in the library's history but isn't imported.

To find these versions, Code Rules lists only tags under `techs/` and `practices/` and fetches the release history without file contents it doesn't need. It records each imported rule's version in `_source.json` and generated provenance, and shows it in generated guidance.

### Tracing rules to their source

Code Rules records the revision you requested, the exact commit it imported, and each imported rule's version. See [Provenance](/reference/provenance/) for these records.

Links to original files on GitHub.com and GitLab.com use the imported commit, so moving a tag does not change their destination. For other Git hosts, links point to the stored files; provenance retains the repository address and commit.

Code Rules preserves rule attribution in both imported and generated files and keeps attribution links valid. It verifies and retains the license and notice files declared in the library manifest, includes them in integrity checks, and reports their changes during updates.

Generated rules link to retained terms under `generated/libraries/<source-name>/licenses/`. See [License rules](/guides/license-rules/) for how library authors declare those files.

## Checks before updating your files

Code Rules fetches and validates all selected libraries before replacing your project's imported files or generated guidance. If any library cannot be fetched or validated, your previous complete set of rules stays in place.

Validation rejects:

- Invalid or reserved source names, repeated repositories, and duplicate rule IDs that include the same source name.
- Missing groups or rules named in an exclusion or replacement.
- A rule that is both excluded and replaced, or a local replacement file used for multiple targets.
- Invalid metadata, unsafe file paths, and symbolic links.

Selected library rules must pass validation even if you exclude or replace them. File checks also ensure that paths stay within their allowed directories.

Code Rules detects concurrent writes and interrupted updates so a mixture of old and new output cannot pass a consistency check. For file replacement and recovery behavior, see [Sync and recovery](/reference/sync/).

## Git access and supported files

Imports require Git 2.30 or later on macOS or Linux. Code Rules accepts HTTPS and SSH repository addresses, including scp-style SSH addresses and nested repository paths. See [Repository addresses](/reference/configuration/#repository-addresses) for accepted formats.

Code Rules uses your Git credentials and certificate and host-key verification settings. It does not store credentials in configuration or provenance. A valid repository address does not guarantee that the repository is reachable or appropriate for your network.

Git URL rewrites still apply. If a rewrite uses another protocol, your Git configuration must explicitly allow that protocol. Executable `ext` helpers are always disabled. Code Rules passes Git arguments separately and applies the same path checks and resource limits across hosts.

Code Rules reads original Git file contents without checking out the library. It does not run library scripts, Git hooks, or checkout filters. Selected symbolic links, submodules, and Git LFS pointers are unsupported; Code Rules does not fetch submodule contents.

### Supporting files

Code Rules copies supporting material from [two asset locations](/reference/rule-library-format/#supporting-assets):

- **A rule's own assets:** the adjacent `assets/<rule-name>/` directory. Code Rules copies this directory in full when it imports the rule.
- **Shared assets:** the library-root `assets/` directory. Code Rules copies this directory in full when a selected rule or its Markdown assets link to it.

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
| Git's listing of rule tags | 8 MiB and 20,000 records, including extra records Git uses to identify commits behind annotated tags. |

The retained-file limits apply after fetching. They do not cap network traffic or Git's temporary disk use. Finding rule versions uses the same Git access settings and deadline as fetching.

When finding a library's newest release, Code Rules can report:

| Error | Meaning and next step |
| --- | --- |
| `releases-not-found` | The source has no `ref`, but the library has no `code-rules/released` branch because it hasn't published its first release. Ask the maintainer to release, or pin a commit. |
