# Imports implementation plan

Status: approved implementation plan for the second subsystem, following [Builds](builds.md).
This plan records the approved scope of the Imports implementation PR.

## Outcome

Fetch the groups a project selects from its libraries and return snapshots that Builds can consume.
Each snapshot identifies the requested ref and the exact commit that supplied its files.
Keep original source bytes available for Workspace to install later.

For example, a project selects `practices/testing` from two libraries, each at its own tag.
Imports resolves both tags and returns separate snapshots, even when their rule filenames match.
Builds then applies the project's exclusions, replacements, and local rules and creates one aggregate for the group.

The first increment ends with an import-to-build demonstration using real Git repositories.
The first increment does not ship `sync` or write into a consuming project's `vendor/` or `generated/` directories.

## Existing contracts

The [configuration reference](../docs/src/content/docs/reference/configuration.md) owns source aliases, repositories, refs, groups, and exceptions.
The [file reference](../docs/src/content/docs/reference/files.mdx) owns library layout and the proposed workspace layout.
The [import reference](../docs/src/content/docs/reference/imports.md) describes the complete user workflow across all three subsystems.

`src/builds/index.ts` exports the offline `buildRules` operation and its input and output types.
`LibrarySnapshot` contains repository identity, the requested ref, the resolved commit, selected groups, and a text file map.
Builds validates authored rules and applies project decisions without accessing Git or the filesystem.

Two existing details need attention before Imports can supply those snapshots:

- Configuration parsing and library manifest validation live inside Builds.
  Imports must share their contracts without importing private Builds helpers or duplicating validation policies.
- A text map cannot preserve binary attachments.
  Imports needs original bytes, and Builds needs a way to recognize vendored attachments without decoding them as text.

## Ownership

| Owner | Responsibility |
| --- | --- |
| Imports | Resolve refs, fetch Git objects, select files, preserve bytes, and report snapshot identity. |
| Builds | Validate rule content, resolve project exceptions, and render aggregates and provenance. |
| Workspace | Load project inputs, verify saved files, compare revisions, and safely install a complete result. |
| CLI | Parse arguments, present progress and errors, and call the subsystems. |

Imports may use temporary storage that it creates and cleans up.
Imports must not modify the project's configuration, lock state, vendored files, or generated files.
Failure returns no usable partial result.
Workspace will later preserve the installed ruleset if fetching, building, or installation fails.

## Proposed interface

Expose one operation, `importLibraries`, from `src/imports/index.ts`.
It accepts raw `unknown` project configuration and optional cancellation and returns all imported libraries keyed by source alias.
An empty source selection succeeds without invoking Git.

Each imported library carries:

- A `LibrarySnapshot` for Builds, including the resolved commit and selected groups.
- The original contents of each imported file, keyed by its path within the library, ready for Workspace to write into `vendor/<source>/`.

The operation resolves sources in a stable order and returns only after every source succeeds.
Start with sequential fetching; concurrency must not change results or diagnostics if added later.
Keep Git process handling private to Imports.

Move configuration parsing into a small shared module with a documented entry point.
Keep exclusions and replacements in that model, but leave their application in Builds.
Move only the manifest parsing needed by both subsystems into a shared library-format module.
Preserve existing Builds error behavior and validation order through its public tests.
These supporting modules are not additional subsystems.

### Text and attachments

Preserve every selected file as bytes in the import result.
Decode required JSON and Markdown as strict UTF-8; reject invalid text with its source and path.
Never replace binary content with empty strings or decode it with replacement characters.

Add an optional file-path inventory to `LibrarySnapshot` so Builds can resolve links to binary attachments.
When omitted, the inventory defaults to the existing text-map keys, preserving direct Builds callers.
The import result derives the inventory from the same byte map it returns to Workspace.
Builds validates inventory paths and requires every supplied text file to appear in an explicit inventory.
Workspace will later authenticate that inventory against the files it loads.

Do not finalize the `_source.json` serialization format in this increment.
Workspace will own serialization and verification when it gains an implementation.

## Resolve one immutable revision

Use the installed Git executable and the user's configured credentials for GitHub repositories.
Keep the public repository format as `owner/repository`.
A missing Git executable or failed authentication must identify the failed source and give a useful next step.
Never include credentials in diagnostics.

Fetch into a fresh temporary bare repository without checking out the library.
A full commit SHA selects that exact object; failure must not substitute another revision.
A tag selects `refs/tags/<name>` and must resolve to a commit, including annotated tags.
Reject branches and tags that ultimately point to trees or blobs.
Resolve the commit from the fetched object, then read every file from that commit.

Reimporting a moved tag returns its new commit.
Workspace will compare that commit with the recorded one and explain the change during `sync`.
Offline Builds continues to use its supplied snapshot without contacting Git.

Align configuration validation with Git's tag-name rules.
Cover empty path components and dot-prefixed components, which the existing validator does not reject consistently.
Retain explicit `refs/tags/<name>` as the unambiguous spelling for tags that resemble abbreviated commit hashes.

Git supports full object IDs in fetch refspecs and defines ref-name restrictions in its primary documentation:
[git-fetch](https://git-scm.com/docs/git-fetch) and [git-check-ref-format](https://git-scm.com/docs/git-check-ref-format).

## Select and preserve files

1. Read and validate `rule-library.json` from the resolved commit.
2. Require every selected group and its `_group.json`.
3. Retain every regular file under each selected group, including rules excluded or replaced by project configuration.
4. Retain the manifest's declared license and notice files.
5. Retain existing local file targets referenced by standard Markdown links, images, and reference definitions in retained Markdown.
6. Continue following those references in newly retained Markdown until no additional files remain.

Track visited paths so cyclic references terminate.
Resolve relative references against their declaring file, preserving query strings and fragments when Builds rewrites the link.
A reference to another group's file does not select that group or activate its rules.
Directory references and absent ordinary document targets retain the existing commit-pinned upstream-link behavior.
Missing manifest-declared license or notice files fail the import.

Share Markdown target interpretation with Builds where necessary so Imports selects the same paths Builds later rewrites.
Use the existing Markdown parser rather than a second regular-expression parser.
Do not follow remote links or infer file dependencies from arbitrary prose or custom frontmatter strings.

Authors must declare applicable files that are not discoverable through standard Markdown links in the manifest's preservation list (`license.notices`).
Update the licensing guide to explain that requirement and distinguish file preservation from determining legal applicability.
The tool cannot discover every applicable term by reading prose.
Do not add a new per-rule licensing schema in this increment.

Read tree entries with NUL-delimited paths and blobs by object ID.
Git documents those operations in [git-ls-tree](https://git-scm.com/docs/git-ls-tree) and [git-cat-file](https://git-scm.com/docs/git-cat-file).
Do not apply checkout filters or execute repository scripts.
Reject symlinks and submodule entries among required files and followed references.
Do not recurse into submodules or download Git LFS objects; report unsupported selected content rather than silently vendoring a pointer.

Apply contained-path validation before returning files, including reserved metadata paths and paths that collide on the supported filesystem.
Bound fetch duration, tree size, file count, and retained bytes; expose named limits and test their failure behavior.
Use a 120-second deadline, 10,000 tree entries, an 8 MiB tree listing, 8 MiB per retained file, and 64 MiB retained per library.
Cancellation or failure must terminate child processes and remove temporary repositories.

## Implementation sequence

1. **Establish shared contracts.**
   Extract configuration and manifest parsing without changing existing behavior.
   Add the attachment inventory and public Builds tests for binary-link destinations and invalid inventories.
2. **Implement Git retrieval.**
   Add `src/imports/index.ts`, a coordinator, focused Git process handling, and colocated tests.
   Verify exact commits and tags against temporary repositories before adding file selection.
3. **Implement file selection.**
   Add manifest, group, referenced-file, and byte-preservation handling.
   Test containment, unsupported entries, errors, limits, and cleanup through the Imports entry point.
4. **Connect Imports to Builds.**
   Add integration coverage and `tests/manual/imports.ts` with original example rules.
   Update the README, project status, and owning reference pages to distinguish implemented APIs from the proposed CLI.

Name internal modules after their responsibility as the implementation develops.
Avoid a generic Git-provider framework or a shared validation framework for this one caller relationship.
Every function should have a useful role or result comment, following the local comment rule.

## Verification

Use real temporary Git repositories for automated tests; CI must not depend on GitHub availability or credentials.
Route the normal GitHub URL to those fixtures using isolated Git configuration in a child process.
Keep test routing out of the public API and restore all environment state after each scenario.

The behavior matrix must cover:

- Full commits, lightweight tags, annotated tags, moved tags, and a branch sharing a tag's name.
- Missing refs, non-commit tags, invalid tag names, and exact revision identity on every returned file.
- Multiple libraries with matching rule paths, reordered sources, and an empty source selection.
- Selected groups, excluded upstream rules retained as source, local replacements, and unselected groups referenced only as attachments.
- Binary bytes unchanged, strict text decoding, linked images, encoded paths, cyclic links, and declared license files outside selected groups.
- Invalid manifests, missing groups or declared files, symlinks, submodules, LFS pointers, path collisions, limits, cancellation, and temporary-directory cleanup.
- A later source failing after an earlier source succeeds, with no partial result or project writes.

The integration test passes imported snapshots to `buildRules` and checks aggregate content, qualified IDs, license links, attachment links, and resolved-commit provenance.
Use focused assertions on behavior rather than broad output snapshots.
The manual scenario prints a temporary workspace path for inspecting vendor bytes alongside generated Markdown.
An optional private-library smoke test can exercise credentials without making network access a CI requirement.

Before the implementation PR, run `bun run check` and the manual import-to-build scenario.
Review the final diff for API simplicity and clear ownership, and confirm that imports preserve original source content.

## Completion boundary

The subsystem is ready when real Git inputs reliably produce byte-preserving imports and valid Builds output, with the failure cases above covered.
Workspace installation, offline integrity checks, update reports, caching, and the user-facing `sync` command remain subsequent work.

The Builds finalization PR has merged.
Keep the shared-parser extraction as an isolated behavior-preserving commit in this Imports PR.

## Reviewed contracts

Shared parsers throw `ValidationError` with the failing location.
Builds translates those failures to its existing `BuildError` with `invalid-input`.
Imports exposes `ImportError` with codes for invalid configuration or libraries, Git availability, repository access, refs, unsupported content, limits, cancellation, and I/O failures.
Transport failures do not reliably distinguish missing repositories from missing permissions.

Parse each ref once into a commit or tag while retaining its exact configured spelling.
Return resolved commits in lowercase.
Use explicit shallow fetches, disable automatic tag following and terminal prompts, and bound process time and output.
A refused SHA or branch-only name fails without selecting another revision.
Do not publish captured Git stderr, which can contain credentials introduced by URL rewriting.

The text map contains the manifest, selected group metadata, and selected rule Markdown.
Its keys must be a subset of the complete imported path inventory.
Both license-presence checks and Markdown relocation consult that inventory.
Markdown attachments are decoded separately for dependency discovery, never validated as rules, and retained as original bytes.
All Markdown inspected for dependencies must be UTF-8.
Underscore-prefixed Markdown filenames and declared license files are not rules, matching Builds.

The first increment supports macOS and Linux.
Reject case-insensitive NFC-normalized path collisions, including collisions in parent directory names.
Windows filesystem support remains outside this increment.
The manual scenario uses temporary Git repositories and isolated environment-based routing, like automated tests.
A public example library remains an optional network smoke test.

Read blobs through Git's batch protocol and check declared sizes before retaining content.
Use one batch process per dependency wave; recursive discovery does not require a long-lived mutable subprocess interface.
Use separate limits for retained data and subprocess output.
A shallow fetch can still download a large tree; post-fetch blob limits do not bound network traffic or Git's temporary disk usage.
Cancellation and all failures share process termination and temporary-directory cleanup.
