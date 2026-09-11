---
title: "How imports work"
description: "From pinned library snapshots to one effective ruleset per group."
---

An import resolves each configured commit or tag and combines the resulting upstream snapshots with explicit project decisions.
It produces ordinary files that agents can read without running the importer.

## Resolve the active rules

1. Load each resolved source's selected groups and assign source-qualified IDs, then load declared local-only groups.
2. Resolve each source's library-relative exclusions to qualified IDs and remove those rules.
3. Apply each source's replacements as complete local definitions, preserving the qualified target IDs.
4. Add the remaining local rules.
5. Pass resolved rules to Builds to generate the root index, group pages with full rules or applicability summaries, and individual effective definitions.

Sort sources, groups, and rule IDs consistently so reordering configuration does not change the result.
Preserve source-labeled group metadata and rule provenance; source order does not establish precedence.

A local replacement is not also an additional rule.
Replacements stay within their target group in the first release.

Record both the requested ref and resolved commit for every snapshot.
Use resolved commits for source links so a moved tag does not change what a link points to.

## Validate before writing

Reject invalid or reserved source names, repeated repositories, missing selected groups, missing targets, duplicate qualified IDs, a rule both excluded and replaced, and a replacement file reused for multiple targets.
Reject malformed metadata and paths or symlinks that escape the allowed directories.
Treat library contents as data rather than executing their scripts.

Stage and validate the complete result across all sources before installing it.
If any source cannot be fetched or validated, preserve the previous complete ruleset.
Detect concurrent writes and interrupted installations so mixed output cannot pass a consistency check.

## Keep sources reviewable

Copy selected upstream group source files into `vendor/<source-name>/`, including rules hidden by project exceptions.
That retained text lets an update report expose upstream changes that a replacement would otherwise hide.

Preserve each rule's attribution from its metadata or body in the vendored source and generated rule file.
Keep declared and otherwise applicable license and notice files with the source snapshot, including files referenced by individual rules.
Include them in snapshot digests and report changes during updates.
Generated rules link to their applicable preserved terms; see [License rules](/guides/license-rules/).
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
A factory may use review findings in its own validation workflow without changing the import format.
