# Sync implementation plan

`sync()` coordinates importing libraries, generating resolved rules, and applying changes safely.
Imports and Builds retain their existing in-memory interfaces. Sync is an orchestration function, not a third subsystem.

## Scope and ownership

- `src/sync.ts`: load a project, import every source, generate resolved rules, and apply the complete result.
- `src/project-files/`: focused helpers for contained filesystem reads, persisted snapshot records, comparison, locking, staging, and recovery.
- `src/project.ts`: offline generation and read-only consistency checks using those same helpers.
- `src/cli.ts`: thin `sync`, `build`, and `check` commands with `--config`; publishing and authoring commands remain separate work.

The configuration directory owns `config.json` and optional `local/`. Only `vendor/` and `generated/` are replaced.
Persist one versioned `_source.json` record per library, including selection identity and SHA-256 digests of original bytes.
Do not trust recorded digests as proof of remote authenticity. They detect local changes relative to the committed record.
Reject symlinks and special files, verify containment, and detect input or managed-output changes before applying replacements.
Coordinate writers with a project lock. Stage complete output before moving existing directories.
Keep a recovery journal and previous directories until the transaction commits; recover interrupted applications before the next write.
Read-only checks report pending recovery rather than changing files. Offline generation never imports or changes vendor revisions.

## Verification

Use real temporary Git libraries and project directories. Cover initial sync, repeated sync, local edits, removed sources/rules/assets,
license bytes, version constraints, failed imports/generation, modified vendor content, symlinks, concurrent changes, and cancellation.
Verify rollback and interrupted-transaction recovery, including first-time creation.
Run CLI smoke tests and the complete `bun run check` gate. Document limits of multi-directory visibility and filesystem guarantees.

## Delivery

Keep the terminology and plan in the Imports PR. Implement sync in a dependent PR based on Imports.
Keep the recap outside both PRs and update it to demonstrate the real project workflow when available.
