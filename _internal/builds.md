# Builds implementation

## Scope

Builds is the first of three subsystems: Imports, Builds, and Workspace.
The first increment accepts configuration, in-memory library snapshots, local files, and a tool version.
It returns the complete generated file set without network access, filesystem writes, or CLI argument handling.

`src/builds/index.ts` owns the public `buildRules` interface.
Configuration and authored JSON/YAML metadata are validated before rules are combined.
All source groups remain independent until explicit exclusions and replacements determine the active rules.
Output ordering, provenance, and relocated Markdown links must be deterministic.

Imports will fetch libraries and establish snapshot identity.
Workspace will load files, verify recorded digests and filesystem containment, detect concurrent changes, and safely install or compare results.
Builds checks snapshot identity against configuration but cannot authenticate caller-supplied content or identify symlinks from a string map.
The CLI and both other subsystems remain unimplemented.

## Implementation and verification

1. Define the in-memory interface and validate configuration, snapshot identities, group metadata, and rule frontmatter.
2. Resolve active rules, preserve source and local origins, and render the index, groups, and provenance.
3. Test through `buildRules` using original examples covering multiple sources, exceptions, local-only groups, deterministic output, licensing links, and invalid inputs.
4. Run strict TypeScript checks, focused Bun tests, and the existing documentation validation.

The builder returns paths relative to `generated/`.
Snapshot file paths are relative to the library root; local file paths are relative to `local/`.
Replacement configuration retains its documented `local/` prefix.
The snapshot envelope contains `repository`, `ref`, `resolvedCommit`, `groups`, and `files`.
Workspace is responsible for mapping verified `_source.json` records into that envelope.

Run `bun run builds:example` to create a temporary workspace from original example rules.
Inspect its index and group Markdown, checking source labels, active obligations, exception reasons, and provenance.
Compare a second build with reordered sources to confirm that its bytes are identical.

## Rules informing the implementation

Apply the relevant [Fabrica TypeScript rules](https://github.com/fabricahq/app/tree/main/_rules/typescript).
Keep processing pure and focused, validate unknown input, annotate public return types, colocate implementation and tests, and assert public behavior.
Cover empty and single-item collections and invalid relationships explicitly.
Use focused assertions rather than broad snapshots.
App-specific React, Wails, and logger requirements do not apply to this offline module.

## Initial implementation limits

The snapshot envelope is an internal Builds interface, not a finalized `_source.json` wire format.
Source maps contain UTF-8 text; Imports and Workspace will own copying binary attachments.
The Markdown renderer relocates standard Markdown links, images, and reference definitions while preserving unrelated body text.
Relative links in raw HTML are rejected with an instruction to use Markdown syntax.
Per-rule attribution and extra frontmatter are retained; library default license files must be present when declared.
The builder does not infer legal obligations from prose or execute imported content.

TypeScript checks project source strictly.
`skipLibCheck` bypasses upstream declaration errors in Bun 1.4.2's types; it does not disable checking our code.
