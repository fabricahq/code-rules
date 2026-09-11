---
title: "How imports work"
description: "From pinned library snapshots to one effective ruleset per group."
---

The Imports module implements fetching and file preservation.
The complete workflow below also depends on Builds and the proposed Workspace subsystem; the `sync` command has not shipped.

An import resolves each configured commit or tag and combines the resulting upstream snapshots with explicit project decisions.
It produces ordinary files that agents can read without running the importer.
This checkout implements fetching and offline generation. Digest checks and safe installation remain planned; see [Project status](/status/).

## Resolve the active rules

The builder receives configuration, supplied library snapshots, and local rules. It selects and resolves rules without network access.

For `groups: "*"`, discover every technology and practice group at the resolved revision before applying project exceptions.
For `"practices/*"` or `"techs/*"`, discover every group of that kind.
Record the original selector and concrete group IDs in a snapshot complete for that scope.
Reject malformed groups and orphan rule files rather than silently dropping them.

1. Load each resolved source's selected groups and assign source-qualified IDs, then load declared local-only groups.
2. Resolve each source's library-relative exclusions to qualified IDs and remove those rules.
3. Apply each source's replacements as complete local definitions, preserving the qualified target IDs.
4. Add the remaining local rules.
5. Render the root index, group pages, individual effective definitions, library summaries and license copies, and provenance.

Sort sources, groups, and rule IDs consistently so reordering configuration does not change the result.
Preserve source-labeled group metadata and rule provenance; source order does not establish precedence.

A local replacement is not also an additional rule.
Replacements stay within their target group in the first release.

Record both the requested ref and resolved commit for every snapshot.
Use resolved commits for source links so a moved tag does not change what a link points to.

## Validate before writing

Reject invalid or reserved source names, repeated repositories, missing groups or targets, and duplicate qualified IDs.
Reject rules that are both excluded and replaced, and replacement files reused for multiple targets.
The builder rejects malformed metadata and unsafe relative paths. It validates selected source rules even when exclusions or replacements make them inactive.
The planned workspace layer must check filesystem containment and symlinks. An in-memory text map cannot establish those properties.
Treat library contents as data rather than executing their scripts.

The planned installation workflow stages and validates the complete result across all sources before writing project files.
If any source cannot be fetched or validated, preserve the previous complete ruleset.
Detect concurrent writes and interrupted installations so mixed output cannot pass a consistency check.

## Keep sources reviewable

When a source selects all groups, the update report must identify added and removed groups as well as rule changes.

Copy selected upstream group source files into `vendor/<source-name>/`, including rules hidden by project exceptions.
That retained text lets an update report expose upstream changes that a replacement would otherwise hide.

Preserve each rule's attribution from its metadata or body in the vendored source and generated rule file.
Library authors must declare the applicable license and notice files in the library manifest. The builder verifies their presence and preserves their text.
The planned import workflow includes those files in snapshot digests and reports changes during updates.
Generated rules link to copies under `generated/libraries/<source-name>/licenses/`; see [License rules](/guides/license-rules/).
Keep attribution links valid after relocation.
References to unvendored upstream documents should point to the resolved commit rather than a mutable tag or broken local path.

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

Imports follows standard relative Markdown links, images, and reference definitions to existing files in the same commit.
It also follows links in retained Markdown attachments, without adopting those attachments as rules.
All Markdown inspected for references must be UTF-8.
It does not discover dependencies in arbitrary prose or custom frontmatter.

Within a selected group, ordinary `.md` files are rules, including files in nested directories.
Markdown filenames starting with `_` and manifest-declared license files are attachments instead.
Use `_README.md` for group introduction text that does not follow the rule template.
