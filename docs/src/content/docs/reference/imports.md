---
title: "How imports work"
description: "From pinned library snapshots to one resolved ruleset per group."
---

An **import** copies selected rules from a library into your project. Code Rules records the library's exact Git commit so you can trace where those rules came from.

The `code-rules sync` command imports your selected libraries, applies your local rules and configured exceptions, and generates the guidance your agents read. Keeping the imported files in your project lets your team rebuild and use that guidance offline.

This page explains what Code Rules imports, how it decides which rules are active, and how it validates changes before writing files. For setup instructions, see [Import rules](/guides/select-rules/).

## What a vendored library contains

`vendor/<source>/` is a selected copy of original files from one resolved Git commit, not a clone.
It contains selected groups, their assets, the manifest, and declared license and notice files. It contains no Git history or `.git` directory.
Imports reads these files through temporary Git storage, then removes that storage. The importer returns parsed catalogs and original retained bytes; sync persists those bytes into `vendor/`.

With unchanged configuration and the same resolved commit, Imports returns the same paths and bytes.
With unchanged snapshots, local files, tool version, and rendering options, Builds returns the same generated content.
Re-importing a tag can change the result if that tag moves. Re-importing a version constraint can select a newer matching tag. A full commit pin continues selecting the original content.
Safe replacement of existing project folders, including removal of stale files, belongs to the sync workflow.

## Resolve the active rules

The builder receives configuration, supplied library snapshots, and local rules. It selects and resolves rules without network access.

For `groups: "*"`, discover every technology and practice group at the resolved revision before applying project exceptions.
For `"practices/*"` or `"techs/*"`, discover every group of that kind.
Record the original selector and concrete group IDs in a snapshot complete for that scope.
Reject malformed groups and orphan rule files rather than silently dropping them.

1. Load each resolved source's selected groups and assign source-qualified IDs, then discover local `_group.json` files.
2. Resolve each source's library-relative exclusions to qualified IDs and remove those rules.
3. Remove each replaced source rule and include its complete local definition under the local rule ID.
4. Add the remaining local rules.
5. Render the root index, group pages, individual resolved definitions, library summaries and license copies, and provenance.

Sort sources, groups, and rule IDs consistently so reordering configuration does not change the result.
Preserve source-labeled group metadata and rule provenance; source order does not establish precedence.

Each local rule appears once, including when referenced by a replacement.
Generated guidance uses the local ID, title, metadata, body, attribution, and asset references.
Replacement targets and reasons remain in configuration and provenance, outside the rule guidance.
Replacements stay within their target group.

Record the requested ref or version constraint and resolved commit for every snapshot. Version selections also record the chosen tag and normalized version.
Use resolved commits for remote source links so a moved tag does not change what a link points to.
GitHub.com and GitLab.com have recognized file and image URL formats.
Other hosts use relative links to the retained source files; their repository address and resolved revision remain in provenance.

## Validate before writing

Reject invalid or reserved source names, repeated repositories, missing groups or targets, and duplicate qualified IDs.
Reject rules that are both excluded and replaced, and replacement files reused for multiple targets.
The builder rejects malformed metadata and unsafe relative paths. It validates selected source rules even when exclusions or replacements make them inactive.
The file-handling helpers check filesystem containment and reject symlinks. An in-memory text map cannot establish those properties.
Treat library contents as data rather than executing their scripts.

Sync stages and validates the complete result across all sources before writing project files.
If any source cannot be fetched or validated, preserve the previous complete ruleset.
Detect concurrent writes and interrupted installations so mixed output cannot pass a consistency check.

## Keep sources reviewable

Sync reports changed files, including group metadata and source records. A semantic report identifying added and removed groups is not available.

Copy selected upstream group source files into `vendor/<source-name>/`, including rules hidden by project exceptions.
That retained text lets an update report expose upstream changes that a replacement would otherwise hide.

Preserve each rule's attribution from its metadata or body in the vendored source and generated rule file.
Library authors must declare the applicable license and notice files in the library manifest. The builder verifies their presence and preserves their text.
Sync includes those files in snapshot digests and reports changed files during updates.
Generated rules link to copies under `generated/libraries/<source-name>/licenses/`; see [License rules](/guides/license-rules/).
Keep attribution links valid after relocation.
Code Rules rejects filesystem links to other rule documents, whether selected, excluded, or unselected. Rules must remain independently selectable. Supporting assets must be retained at their conventional paths on every host; generation rejects missing assets.

## Fetch through Git, independently of the host

The repository field accepts explicit HTTPS and SSH Git addresses, including scp-style SSH and nested namespaces.
See [Repository addresses](/reference/configuration/#repository-addresses) for the grammar and credential boundary.
Shared configuration validates addresses before Imports fetches them. Builds uses the same address parser for source links.

Imports invokes Git with separate arguments and permits HTTPS and SSH by default.
Git URL rewrites still apply. Other rewritten protocols require an explicit protocol-specific allowance in the caller's Git configuration; executable `ext` helpers are always disabled.
Imports retains the caller's certificate and host-key verification settings.
Use the caller's credentials without storing secrets in configuration or provenance.
Do not recurse into submodules or execute library scripts, hooks, or checkout filters while creating snapshots.
Apply resource limits and the existing path and installation checks to all hosts.
Address validation alone does not establish that a remote is reachable or safe for a particular network environment.

## Resolve declared exceptions

The importer uses IDs and configuration to decide which rules are active.
It does not interpret prose to discover undeclared contradictions.
Rules with matching paths from different sources are distinct and remain active.
Authors and reviewers still need to identify conflicting obligations and declare the intended exclusion or replacement explicitly.

## Read the result

Agents start at the generated index, choose relevant groups, and apply each rule within its scope.
The same files support both writing and review.
Projects choose their own review and enforcement workflow; the import format does not prescribe one.

## Fetching implementation limits

Imports requires Git 2.30 or later on macOS or Linux.
It reads original blobs without checking out a library or running checkout filters.
Selected symlinks, submodules, and Git LFS pointers are unsupported.
Imports rejects ambiguous retained paths, including case-insensitive NFC-normalized collisions in directory names.

Each library has a 120-second deadline and may contain at most 10,000 tree entries.
The tree listing may occupy up to 8 MiB; retained files may occupy up to 8 MiB each and 64 MiB in total.
Those file limits apply after fetching and do not cap network traffic or Git's temporary disk use.

Imports preserves supporting material only from [the two asset locations](/reference/rule-library-format/#supporting-assets): each selected rule's adjacent `assets/<rule-name>/` directory and the library-root `assets/` directory.
Owned directories are copied completely. Shared assets are copied completely only when a selected rule or a retained Markdown asset references them.
Imports validates standard Markdown links, images, and reference definitions against these boundaries; it does not follow arbitrary repository documents.
Missing destinations and links into another rule's private assets fail import. Links from rules or Markdown attachments to other rule documents fail import. External URLs are not fetched.

Markdown assets and declared license and notice files must be UTF-8. Binary assets retain their original bytes, but binary license and notice files are unsupported.
Within a selected group, Markdown outside asset directories defines rules, including nested rule files. Declared license and notice files are exempt.
Move supporting files such as `_README.md` into the owning rule's assets directory or the shared root directory.

Version discovery uses the same deadline and Git transport policy as fetching. The tag advertisement is limited to 8 MiB and 20,000 records, including annotated-tag peel records.
`version-not-found` means no eligible tag satisfies the range. `ambiguous-version` identifies conflicting tags at the highest matching precedence.
`ref-changed` means the selected tag moved between discovery and fetch. Correct the tags, adjust the constraint, pin an exact ref, or retry as appropriate.
