# Native library authoring

PR36 adds native `library init`, `library add group`, `library add rule`, and `library check` commands. They target `--directory`, independently of consumer configuration. Required metadata is explicit in this slice; interactive collection follows separately.

Initialization creates a manifest and orientation without overwriting valid existing files. Optional SPDX terms require publisher-supplied license text. The license and optional notice retain their exact UTF-8 bytes, including CRLF. No license text is inferred or generated.

Group and rule creation reuse protected authoring publication. New groups include a README for agents alongside _group.json. Rules require an existing group; missing groups return an error instructing the caller to create the group first. Group READMEs are authoring documentation, excluded from rule loading and adoption; existing files are never overwritten. This reserved filename has the same meaning in third-party libraries: its prose does not require rule frontmatter or undergo rule/link validation. Its bytes remain subject to filesystem safety and size limits. Missing rule bodies use the embedded canonical template plus a draft marker. `library check` rejects that marker until the author completes the draft and removes it.

The read-only check validates the manifest, declared terms, all groups and rules, and every file under techs/, practices/, and assets/. Unreferenced assets still need valid owners and valid local links. Cross-rule links are rejected under the approved migration policy. Binary attachments are preserved. Unrelated repository files such as .git/ and docs/ are outside this scope.

`/walkthrough/pr36` executes eleven native CLI scenarios and shows transcripts, exit codes, and all before/after files. It covers original terms, a complete library, draft rejection, collisions, missing groups, orphan assets, missing destinations, cross-rule links, and unrelated repository content. The production npm entry point remains unchanged.

Library checking validates one captured set of original bytes and directory entries.
Catalog counts and unused-content checks use that same snapshot. A final comparison
rejects observed changes before returning. It does not lock ordinary editors or
promise that files remain unchanged after their final read.
