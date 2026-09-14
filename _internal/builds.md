# Builds implementation

## Scope

Builds is the first of three subsystems: Imports, Builds, and Workspace.
The first increment accepts configuration, in-memory library snapshots, local files, and a tool version.
It returns the complete generated file set without network access, filesystem writes, or CLI argument handling.

`src/builds/index.ts` owns the public `buildRules` interface.
Configuration and authored JSON/YAML metadata are validated before rules are combined.
All source groups remain independent until explicit exclusions and replacements determine the active rules.
Output ordering, provenance, and relocated Markdown links must be deterministic.

Imports fetches libraries and establishes snapshot identity.
Workspace will load files, verify recorded digests and filesystem containment, detect concurrent changes, and safely install or compare results.
Builds checks snapshot identity against configuration but cannot authenticate caller-supplied content or identify symlinks from a string map.
Workspace and the CLI remain unimplemented.

## Internal operations

`build.ts` coordinates configuration validation, rule resolution, and generated-file rendering.
Keep `index.ts` as the caller-facing interface; the modules below are implementation details.

- `../configuration.ts` interprets source selections and exception declarations, retaining field locations in diagnostics.
- `resolve.ts` validates snapshots and resolves imported, replaced, and local definitions into groups.
  Each active rule is stored in its group once; provenance derives its rule list from those groups.
- `render.ts` builds the root index, adaptive group pages, individual effective definitions, library READMEs, and provenance without mutating resolved groups.
- `index-pages.ts` splits oversized indexes at entry boundaries and checks every output page against its UTF-8 byte budget.
- `rule-document.ts` validates YAML and rule metadata while retaining the original frontmatter and body text.
- `rule-attribution.ts` validates optional external attribution entries.
- `../formats/manifest.ts` validates library-wide SPDX declarations and verifies the declared files exist.
- `license-output.ts` maps source license and notice files to fixed paths under `generated/libraries/<source>/licenses/` and preserves their contents.
- `markdown.ts` relocates references and renders active definitions.
  Its rewrite traversal edits children before serializing their owning outer node, then applies non-overlapping source edits from right to left.
- `validation.ts` supplies shared shape and path guards, errors, and group metadata parsing.

Preserve validation order during refactors, including malformed excluded rules and duplicate repositories preceding later source-field errors.
Test through `buildRules` so internal helpers can move without changing caller-facing tests.

## Implementation and verification

1. Define the in-memory interface and validate configuration, snapshot identities, group metadata, and rule frontmatter.
2. Resolve active rules, preserve source and local origins, and render the index, groups, and provenance.
3. Test through `buildRules` using original examples covering multiple sources, exceptions, local-only groups, deterministic output, licensing links, and invalid inputs.
4. Run strict TypeScript checks, focused Bun tests, and the existing documentation validation.

The builder returns paths relative to `generated/`. Group pages live under `groups/<group-id>.md`, while individual effective definitions live under `rules/`.
Imported library READMEs and unchanged license copies live under `libraries/<source>/`.
`RULES.md` and `provenance.json` remain at the generated root; group IDs and rule IDs do not change with output layout.
Snapshot file paths are relative to the library root; local file paths are relative to `local/`.
Replacement configuration retains its documented `local/` prefix.
The snapshot envelope contains `repository`, `ref`, `resolvedCommit`, `groups`, and `files`.
It also accepts `groupSelection`, recording the original list or `"*"`, `"practices/*"`, or `"techs/*"`. Wildcard configuration requires an explicitly marked complete snapshot.
Resolution expands group metadata and rules within the selected scope, compares their IDs with the recorded `groups`, validates their contents, and applies the existing source-scoped exceptions.
Legacy snapshots without the field remain valid for explicit lists. A completeness marker is a supplier assertion, not remote authentication.
Workspace is responsible for mapping verified `_source.json` records into that envelope.

Run `bun run builds:example` to execute `tests/manual/builds.ts` and create a temporary workspace from original example rules.
Inspect its group indexes and linked full rule files, checking source labels, active obligations, exception reasons, and provenance.
Compare a second build with reordered sources to confirm that its bytes are identical.

## Rules informing the implementation

Apply the relevant [Fabrica TypeScript rules](https://github.com/fabricahq/app/tree/main/_rules/typescript).
Keep processing pure and focused, validate unknown input, annotate public return types, colocate implementation and tests, and assert public behavior.
Cover empty and single-item collections and invalid relationships explicitly.
Use focused assertions rather than broad snapshots.
App-specific React, Wails, and logger requirements do not apply to this offline module.

## Initial implementation limits

The snapshot envelope is an internal Builds interface, not a finalized `_source.json` wire format.
Snapshot `files` contains UTF-8 text; optional `filePaths` lists all retained files, including binary attachments.
Builds uses the inventory for Markdown links and requires declared license and notice text in the text map and rejects inventories that omit supplied text or hide selected rule text.
Imports preserves original file bytes; Workspace will own installing and verifying them.
The Markdown renderer relocates standard Markdown links, images, and reference definitions while preserving unrelated body text.
Relative links in raw HTML are rejected with an instruction to use Markdown syntax.
Per-rule attribution and extra frontmatter are retained; library license files must be present when declared.
The builder does not infer legal obligations from prose or execute imported content.

TypeScript checks project source strictly.
`skipLibCheck` bypasses upstream declaration errors in Bun 1.4.2's types; it does not disable checking our code.

## Applicability indexes

Every source rule, local addition, and replacement requires a non-empty `whenToRead` string.
Rule selection remains the agent's judgment; metadata must describe the intended work before a violation exists.
Full definitions retain obligations, implementation and validation guidance, attribution, and relocated links.

`buildRules` accepts optional `indexMaxBytes` (positive safe integer, default 24 KiB).
The root index and group pages fit that UTF-8 byte limit. Oversized summary indexes split into complete numbered sibling parts.
`groupInlineMaxBytes` (non-negative safe integer, default 8 KiB) selects full inline delivery when the complete group page fits both budgets. Zero forces summary indexes.
The inline limit counts the entire rendered page, including metadata, examples, attribution, and navigation. It is a provisional delivery default, not a measured compliance threshold.
Both formats place rule titles at heading level 3 under `## Rules`, following `## How to use this group`. Full definitions put the body under **Guidance**, followed by **Source and attribution** and its **Source metadata** subsection. Body headings nest beneath Guidance in standalone and inline files.
Both formats use the same resolved definitions and retain stable individual rule files. Embedded fragment links point to standalone definitions to avoid repeated heading collisions.
Impact describes credible consequences; it neither selects rules nor assigns finding severity. `impactDescription` appears as **Why it matters**.
A single oversized entry or part directory fails explicitly; rule bodies are never truncated.
Output paths use `rules/<source-name>/<rule-path>.md`, preserving replacement IDs independently of local source filenames.
The complete returned map excludes obsolete definitions; Workspace will own stale-file comparison and installation, including removing obsolete index parts.
The CLI `check` command and private-library migration are still future work. The builder tests verify that metadata edits change generated output deterministically.
