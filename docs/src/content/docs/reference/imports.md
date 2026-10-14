---
title: "How imports work"
description: "How library rules and skills become project guidance and installed skills, what happens during updates, and what imports support."
---

An **import** copies selected rules and skills from a shared library, or skills from a plain Git repository, into your project.

Imports let you reuse your team's engineering practices across projects without writing and maintaining the same rules and skills in each one. You can also import rules and skills from third-party libraries and repositories whose practices you want to adopt. Each project can choose which rules and skills to use, add its own, and replace imported ones to fit its needs.

This page explains which files Code Rules imports and how it combines imported rules and skills with your local ones and your exceptions. It also covers the checks that protect your project during an update and the limits on what you can import. For step-by-step instructions, see [Import rules](/guides/select-rules/).

## From library rules to project guidance

When you run `code-rules project sync`, Code Rules:

1. Reads your configuration, combined with your [preset](/reference/configuration/#extend-a-preset) when you extend one, to find the sources, groups, rules, skills, and versions you selected. A **group** collects related rules, such as testing practices or TypeScript conventions.
2. Copies the selected files into your project. The copy of one source is called a **snapshot**; each rule and skill in it comes from the library release that published its version, or, for a plain repository, from the recorded commit.
3. Combines the imported rules with your local rules and configured exceptions, then generates files for your agents to read.
4. Installs each selected skill, and the skills it uses, into your [install targets](/reference/configuration/#install-targets), such as `.claude/skills/`.

These files live in the **Code Rules directory**, `.code-rules/` at the project root:

| Directory | What it contains |
| --- | --- |
| `vendor/` | Original files copied from the libraries and plain repositories you selected. |
| `local/` | Rules and skills you author for this project, including replacements for imported ones. |
| `generated/` | The rules and reading indexes your agents use, including `SKILLS.md`. |

Installed skills live outside it, in the target folders at the project root. For the complete layout, see [Project files](/reference/files/). Once the library files are stored, `code-rules project build` can regenerate guidance and installed skills without network access.

## What gets copied

Each library has a **source name** in your configuration, such as `team`. Code Rules stores that library's imported files under `vendor/team/`.

The copy includes:

- Rules from the selected groups and the individually selected rules, including rules your project excludes or replaces.
- Supporting files, such as images and examples, from the library's designated asset directories.
- Group metadata, which describes each group and when to read it.
- The library manifest, `rule-library.yaml`, and its declared license and notice files.
- Each installed skill's whole folder, `skills/<name>/`, including skills your project excludes or replaces, and the skills that installed skills use.
- The rules that installed skills cite, with their asset directories and group metadata.

Keeping the original rules and skills lets you review library changes even when your project uses a replacement. Make project-specific changes in `local/`; syncing replaces the imported files.

The snapshot contains no Git history or `.git` directory. Code Rules reads the original files through temporary Git storage and removes that temporary storage afterward.

A skill's folder is copied as Git tracks it: only tracked files, so ignored files such as `__pycache__/` are never copied, and with each file's executable bit, so a script such as `scripts/verify.sh` stays runnable in the vendored and installed copies. Code Rules never runs it.

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

## How Code Rules chooses the skills to install

Skills don't belong to groups, so selecting a group never installs a skill. Code Rules installs:

1. The skills in each source's `skills` list, or every skill in the source for `"*"`.
2. Every skill those skills use, transitively, from the same source: each skill named in a library skill's `cr-skills-used` list, plus each skill your configuration's [`skillUses`](/reference/configuration/#declare-a-skills-dependencies) adds. A skill reached only this way is a **dependency**; the records and `generated/SKILLS.md` say which skill uses it.
3. Your local skills in `local/skills/`, and the skills they name in `cr-skills-used`, which are project IDs and can come from any configured source.

It then removes the skills you excluded, and adds the local skills your exclusions name as `replacedBy`, in place of the skills they exclude. Each skill's `cr-skills-used` and `cr-rules-used` lists come from the version the project installs.

Code Rules also imports every rule in each installed library skill's `cr-rules-used` list, from the same library, and every rule a local skill's `cr-rules-used` names by project ID, from the named source, even when your project doesn't select its group. These **cited rules** are reference-only: each gets its generated rule file, and the installed skill's [generated section](/reference/files/#installed-skills) links to it, but `RULES.md` and the group pages don't list it, so installing a skill never widens the rules your agents apply to all work. A cited rule your project also selects is an ordinary imported rule. A cited rule your project excludes or replaces follows that choice: the installed skill says it's excluded, or links to your replacement. Rules cited by a skill you exclude aren't imported for it.

Code Rules does not read rule text to detect contradictory instructions. Two libraries can supply conflicting rules, and both remain active unless you configure an exclusion or replacement. The order of libraries in your configuration does not establish priority. See [Resolve conflicting rules](/guides/conflicting-guidance/).

## Where agents read the result

Code Rules writes the active rules and reading indexes to `generated/`. Agents start at `generated/RULES.md`, open relevant groups, and read the applicable rules in full.

The generated files also include library summaries, retained license files, and **provenance**: records of where rules came from and which rules they replaced. Replacement targets and reasons stay in configuration and provenance, outside the rule guidance. Group descriptions remain labeled by source so you can see which library supplied them.

The same files support implementation and review. Your project chooses how to check that agents follow the rules; importing does not enforce compliance. Likewise, Code Rules installs skills but doesn't run them, decide when agents use them, or check that agents followed them.

## What changes when you update

Each source's `vendor/<source-name>/_source.json` records the version of every rule and skill it imported, or, for a plain repository, the commit. Running `code-rules project sync` again imports those same versions, so every checkout of the project gets the same rules and skills. `code-rules project sync` chooses a rule's version only for a newly selected rule, a new source or changed repository, a pin you add or change, or a `ref` you add, change, or remove; see [project sync](/reference/cli/#project-sync).

`code-rules project update` previews each rule's and skill's newest version, and applies it once you confirm, unless the rule or skill is [pinned](/reference/configuration/#pin-a-rule). A plain repository that uses `follow` moves to its branch's newest commit. Sources that use `ref` don't move. A preset moves to its library's newest library release.

With the same configuration and recorded versions, an import produces the same paths and file contents. With unchanged imported files, local rules and skills, tool version, and rendering options, a build produces the same generated guidance and installed skills. Reordering libraries, groups, or rules in configuration does not change their generated order.

`code-rules project update` reports each changed rule and skill with its change, versions, and summary, and applies the changes only after you confirm. Both commands report changed files, including group metadata, version records, and installed skill folders.

### How rule versions are resolved

The newest library release is the one with the highest `release/<number>` tag. Each [library release](/reference/rule-versions/#library-releases) tag's message records every rule's version and the changes that library release published.

To choose a rule's version, Code Rules reads the rule's history from the release records in the `release/<number>` tags, and picks the newest version, or the one the rule is pinned to. It then imports the rule's [Markdown file and asset directory](/reference/rule-versions/#what-a-version-covers) from the tagged commit of the library release that published that version. Library-wide files, including the shared files rules link to, come from the one library release that `vendor/<source-name>/_source.json` records as `release`, which is never older than the library release of any imported rule version. A new source or newly selected rules take them from the newest library release, `code-rules project update` moves them there, and `code-rules project sync` keeps the recorded one; see [What a version covers](/reference/rule-versions/#what-a-version-covers).

Skills are resolved the same way, from the release record's `skills` field, which holds their versions, changes, and retirements; see [Release record](/reference/rule-versions/#release-record). A skill version covers the skill's whole folder, which Code Rules imports from the tagged commit of the library release that published that version. When a later library release was published by a version of Code Rules that doesn't write the `skills` field, Code Rules takes skill versions from the newest release record that has it, and warns.

A rule or skill absent from a library release was retired, and the release record's `retired` entry records why. A rule or skill the project pinned before its retirement keeps importing its pinned version. When a pinned skill cites a rule the library has since retired, the rule is imported at its last version, and the installed skill marks it as retired.

To find versions, Code Rules lists only `release/` tags, fetches their messages without the library's history, and fetches only the files it imports. It records each imported rule's version, library release, and commit in `_source.json` and generated provenance, and shows the version in generated guidance.

### Tracing rules to their source

Code Rules records the versions you requested, and each imported rule's exact version and commit. See [Provenance](/reference/provenance/) for these records.

Links to original files on GitHub.com and GitLab.com use the imported commit, so moving a tag does not change their destination. For other Git hosts, links point to the stored files; provenance retains the repository address and commit.

Code Rules preserves rule attribution in both imported and generated files and keeps attribution links valid. It verifies and retains the license and notice files declared in the library manifest, includes them in integrity checks, and reports their changes during updates.

Generated rules link to retained terms under `generated/libraries/<source-name>/licenses/`. See [License rules](/guides/license-rules/) for how library authors declare those files.

## Plain repositories

A source whose repository has no `rule-library.yaml` at the revision Code Rules reads is a [plain repository](/reference/configuration/#plain-repositories). It supplies only skills, and has no release tags, versions, or change notes:

- **Revision.** With `follow`, `code-rules project sync` reads the commit recorded in `_source.json`, and `code-rules project update` reads the newest commit on the branch. With `ref`, Code Rules reads that tag or commit. Either way, `_source.json` records the exact commit.
- **Skills.** Each folder directly inside `skillsRoot` that has a `SKILL.md` file is a skill, with the ID `skills/<folder-name>`. Code Rules copies each installed skill's folder, as Git tracks it, to its path in the repository under `vendor/<source-name>/`, such as `vendor/pstack/pstack/skills/architect/`.
- **Dependencies.** A plain repository's skills declare none that Code Rules reads, even when their `SKILL.md` has `cr-skills-used`. Declare them with [`skillUses`](/reference/configuration/#declare-a-skills-dependencies). Their `cr-rules-used` lists are ignored too, so they cite no rules.
- **Validation.** Lenient: where a skill doesn't follow the Agent Skills format, such as a `name` that doesn't match its folder, a missing `description`, or a link outside the skill's folder, sync warns and installs the skill as it is. The checks that protect your project still fail: unsafe paths, symbolic links, submodules, Git LFS pointers, and the [import limits](#import-limits-and-version-errors).
- **Licenses.** Code Rules looks for terms in this order: the skill's `license` field in `SKILL.md`, a `LICENSE*` file in the skill's folder, and `LICENSE*` and `NOTICE*` files at the repository root. It records what it finds in provenance, and retains the root files, when present, under `generated/libraries/<source-name>/licenses/`, like a library's declared files. When a skill has none of these, sync fails with `license-not-found`.

A source's fields must fit what its repository is: `groups`, `rules`, and `pins` are rejected on a plain repository, and `follow` and `skillsRoot` on a library.

## Checks before updating your files

Code Rules fetches and validates all selected sources, and the preset, before replacing your project's imported files, generated guidance, or installed skills. If any of them cannot be fetched or validated, your previous complete set of rules and skills stays in place.

Validation rejects:

- Invalid or reserved source names, repeated repositories, and duplicate rule IDs that include the same source name.
- Missing groups, or rules or skills named in an exclusion, pin, `rules`, `skills`, or `skillUses` entry that the source doesn't supply. An entry naming a rule the library retired produces a warning instead, when the source would otherwise import that rule: its group is selected, it is listed in `rules`, or the last sync imported it or recorded it as retired. Offline checks accept the same exclusions, because the snapshot records those retired rules in `retiredRules`.
- A missing `replacedBy` file or folder, or one local file or folder named as the replacement for more than one rule or skill.
- Two installed skills with the same name, and installed skills that would overwrite a folder Code Rules doesn't own; see [Who owns the target folders](/reference/sync/#who-owns-the-target-folders).
- Skills whose dependencies form a cycle, including through `skillUses`.
- Invalid metadata, unsafe file paths, and symbolic links.

Selected library rules and skills must pass validation even if you exclude or replace them. A library skill gets the same checks as `code-rules library check` applies, except that importing doesn't check the value of `cr-skill-type`, so a project can install a skill whose type a newer Code Rules added; see [Skill format](/reference/skill-format/#validation). File checks also ensure that paths stay within their allowed directories.

Code Rules detects concurrent writes and interrupted updates so a mixture of old and new output cannot pass a consistency check. For file replacement and recovery behavior, see [Sync and recovery](/reference/sync/).

## Git access and supported files

Imports require Git 2.30 or later on macOS or Linux. Code Rules accepts HTTPS and SSH repository addresses, including scp-style SSH addresses and nested repository paths. See [Repository addresses](/reference/configuration/#repository-addresses) for accepted formats.

Code Rules uses your Git credentials and certificate and host-key verification settings. It does not store credentials in configuration or provenance. A valid repository address does not guarantee that the repository is reachable or appropriate for your network.

Git URL rewrites still apply. If a rewrite uses another protocol, your Git configuration must explicitly allow that protocol. Executable `ext` helpers are always disabled. Code Rules passes Git arguments separately and applies the same path checks and resource limits across hosts.

Code Rules reads original Git file contents without checking out the library. It does not run library scripts, skill scripts, Git hooks, or checkout filters. It preserves a skill file's executable bit, but never runs the file. Selected symbolic links, submodules, and Git LFS pointers are unsupported; Code Rules does not fetch submodule contents.

### Supporting files

Code Rules copies supporting material from [two asset locations](/reference/rule-format/#supporting-assets):

- **A rule's own assets:** the adjacent `assets/<rule-name>/` directory. Code Rules copies this directory in full when it imports the rule.
- **Shared assets:** files in the library-root `assets/` directory. Code Rules copies the files a selected rule or its Markdown assets link to, including files they link to in turn, from the library release that supplies the source's library-wide files.

Markdown links, images, and reference links must point to files within the allowed locations. Missing files and links into another rule's private assets cause an error. Code Rules preserves external URLs as links without downloading their contents.

A rule or Markdown attachment cannot link to another rule document on disk, even if that rule is also selected. Each rule must work independently because projects can exclude or replace it. Put shared supporting explanations in the library's shared assets directory.

Within a selected group, Markdown files outside asset directories count as rules, including files in nested folders. Declared license and notice files and the group-root `README.md` are exceptions. Put other supporting Markdown, such as `_README.md`, in an asset directory.

Markdown assets, license files, and notice files must use UTF-8 text. Other assets can be binary files; Code Rules preserves their original bytes.

A skill's folder can hold any files, such as `scripts/`, `references/`, and `assets/`; its `assets/` folder is the skill's own and follows none of the rule asset rules above. Links in a skill's files must be absolute URLs or relative paths that stay inside the skill's own folder, so each installed copy works on its own. A library skill with any other link fails validation; a plain repository's skill gets a warning.

## Import limits and version errors

Imports reject paths that become identical when letter case is ignored and Unicode names are normalized to NFC. This includes directory names and prevents ambiguous paths across filesystems.

Each library or plain repository import has these limits:

| Resource | Limit |
| --- | --- |
| Fetching and finding rule versions | 120 seconds per library. |
| Entries in the Git file tree | 10,000 entries. |
| Git file-tree listing | 8 MiB. |
| Each retained file | 8 MiB. |
| All retained files combined | 64 MiB. |
| Git's listing of release tags | 8 MiB and 20,000 records, including extra records Git uses to identify commits behind annotated tags. |
| Each release tag | 8 MiB. |
| Each [release record](/reference/rule-versions/#release-record) | 10,000 entries each in `rules`, `changes`, and `retired`, 10,000 each in `skills.versions`, `skills.changes`, and `skills.retired`, and 20,000 in `libraryFiles`. A record over a limit fails with `invalid-release-tag`. |

The retained-file limits apply after fetching. They do not cap network traffic or Git's temporary disk use. Finding rule versions uses the same Git access settings and deadline as fetching.

When finding versions, a preset, or license terms, Code Rules can report:

| Error | Meaning and next step |
| --- | --- |
| `releases-not-found` | The library has no `release/<number>` tags, because it hasn't published its first library release. Ask the maintainer to publish a library release, or import a commit with the source's `ref`. |
| `version-not-found` | A pin names a version the rule or skill never published, the tag or commit in `ref` doesn't exist, the branch in `follow` doesn't exist, or the server says it doesn't have a commit that `vendor/<source-name>/_source.json` records, because the library rewrote its history. Check the pin or `ref`. For a missing recorded commit, ask the library's maintainer to restore it, or set `ref` to a tag or commit the library still has; deleting `vendor/<source-name>/` and syncing is the last resort, because it chooses versions again. A fetch that fails for any other reason, such as a dropped connection, reports that reason's code instead, never this one. |
| `ref-is-branch` | The source's `ref` names a branch. `ref` accepts only a tag or a full commit SHA, so every import can be reproduced. |
| `unsupported-release-record` | A library release's [release record](/reference/rule-versions/#release-record) uses a newer format than this Code Rules reads, because a later Code Rules published it. Upgrade Code Rules. |
| `preset-not-found` | The library release that `extends` reads has no `presets/<name>.yaml` for the configured `preset`. Check the preset's name, or set `extends.ref` to a library release that has it. |
| `license-not-found` | A skill from a plain repository has no license terms: no `license` field, no `LICENSE*` file in its folder, and no `LICENSE*` or `NOTICE*` file at the repository root. Ask the repository's owner to add a license, or stop selecting the skill, and any skill whose `skillUses` names it. |

When Code Rules can't read the library's repository, it can report:

| Error | Meaning and next step |
| --- | --- |
| `connection-failed` | Git couldn't reach the repository's host, such as when the host name doesn't resolve, the connection is refused or times out, or the TLS connection fails. Check the repository address and your network connection. |
| `https-certificate-failed` | The HTTPS server's certificate couldn't be verified. Check that your system trusts it: Git's `http.sslCAInfo` setting, your system's certificate store, and any proxy that intercepts TLS. |
| `ssh-host-key-failed` | The SSH host key couldn't be verified. Check the server's entry in your `known_hosts` file. |
| `not-found-or-no-access` | Git or the server reported that the repository doesn't exist or that your Git credentials can't read it; servers report both the same way. Check the address and your credentials. |
| `object-fetch-refused` | The server refused to send a file by its object ID, which Code Rules needs to read one version of each rule without downloading the whole repository. GitHub.com and GitLab.com allow it; a self-hosted server needs Git protocol version 2 or `uploadpack.allowAnySHA1InWant`. |
| `git-failed` | Another Git failure, including one whose cause Code Rules doesn't recognize. The message says which step failed, such as fetching library files. |

These messages never include what Git or the server printed, which can hold credentials, such as one that `url.*.insteadOf` rewriting adds to the repository address. Code Rules reads that text only to choose the code and message, as [Text from Git, servers, and the GitHub CLI](/reference/cli/#text-from-git-servers-and-the-github-cli) describes. To read Git's message, run `git ls-remote` with the repository address yourself.
