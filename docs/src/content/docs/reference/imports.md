---
title: "How imports work"
description: "From pinned library snapshots to one effective ruleset per group."
---

An import resolves each configured commit or tag and combines the resulting upstream snapshots with explicit project decisions.
It produces ordinary files that agents can read without running the importer.

## Resolve the active rules

For `groups: "*"`, discover every technology and practice group at the resolved revision before applying project exceptions.
For `"practices/*"` or `"techs/*"`, discover every group of that kind.
Record the original selector and concrete group IDs in a snapshot complete for that scope.
Reject malformed groups and orphan rule files rather than silently dropping them.

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
Use resolved commits for remote source links so a moved tag does not change what a link points to.
GitHub.com and GitLab.com have recognized file and image URL formats.
Other hosts use relative links to the retained source files; their repository address and resolved revision remain in provenance.

## Validate before writing

Reject invalid or reserved source names, repeated repositories, missing selected groups, missing targets, duplicate qualified IDs, a rule both excluded and replaced, and a replacement file reused for multiple targets.
Reject malformed metadata and paths or symlinks that escape the allowed directories.
Treat library contents as data rather than executing their scripts.

Stage and validate the complete result across all sources before installing it.
If any source cannot be fetched or validated, preserve the previous complete ruleset.
Detect concurrent writes and interrupted installations so mixed output cannot pass a consistency check.

## Keep sources reviewable

When a source selects all groups, the update report must identify added and removed groups as well as rule changes.

Copy selected upstream group source files into `vendor/<source-name>/`, including rules hidden by project exceptions.
That retained text lets an update report expose upstream changes that a replacement would otherwise hide.

Preserve each rule's attribution from its metadata or body in the vendored source and generated rule file.
Keep declared and otherwise applicable license and notice files with the source snapshot, including files referenced by individual rules.
Include them in snapshot digests and report changes during updates.
Generated rules link to their applicable preserved terms; see [License rules](/guides/license-rules/).
Keep attribution links valid after relocation.
For recognized hosts, references to unvendored upstream documents point to the resolved commit.
For other hosts, retain the referenced documents and images in the snapshot or author explicit URLs. Generation rejects missing relative destinations.

## Fetch through Git, independently of the host

The repository field accepts explicit HTTPS and SSH Git addresses, including scp-style SSH and nested namespaces.
See [Repository addresses](/reference/configuration/#repository-addresses) for the grammar and credential boundary.
The offline builder validates addresses and renders links; downloading remains planned work.

The importer will invoke Git with separate arguments rather than interpolating repository text into a shell command.
It must restrict actual transports to HTTPS and SSH, including after Git URL rewrites, and retain normal certificate and host-key verification.
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
A factory may use review findings in its own validation workflow without changing the import format.
