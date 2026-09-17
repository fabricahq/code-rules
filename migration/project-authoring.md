# Native project authoring

PR35 adds `internal/authoring` and native CLI commands for project-owned source files:

- `init`: create missing configuration and local orientation; preserve existing valid bytes.
- `add source ALIAS`: validate and record a source without fetching it.
- `local add group ID`: create group metadata with trimmed, singular reading guidance.
- `local add rule ID`: create a supplied body or an unfinished canonical draft. The group must already exist; otherwise return an error with the group-creation command.

All commands accept `--config` and `--non-interactive`. This slice requires explicit metadata flags. Interactive collection and library authoring follow separately. The npm entry point still uses TypeScript.

## File ownership

Authoring uses the same project writer lock as build and sync. New files are published with exclusive links; existing files are never silently replaced. Source configuration edits carry exact prior bytes. The publisher claims and verifies the old file, then installs its replacement exclusively. Uncertain editor content and interrupted stages are retained for manual recovery. At most one file can be replaced per operation. This is recoverable publication, not a promise of simultaneous visibility or power-loss durability.

Observed symlinks, hard links, case aliases, stale prior bytes, and pending authoring recovery fail closed. Rule creation accepts local group metadata or a selected group from verified persisted snapshots. It does not fetch a missing group. Unfinished drafts are explicit templates to complete, not generated engineering policy.

## Review

Build both native binaries into one directory, then start rules-lab. `/walkthrough/pr35` runs real command sequences in disposable projects, stops at the first nonzero exit, and displays stdout, stderr, status, and before/after files. No source is fetched by this walkthrough.

Validation includes the actual compiled CLI without runtime tools on PATH, idempotent initialization, source-only edits, local build/check, missing-group refusal, no-overwrite failures, unsafe input refusal, and stale prior-byte rejection. The shared canonical draft body is compiled into the Go binary.

If publication completes but stage or writer-lock cleanup fails, the result still lists the committed paths and includes a visible `warnings` array. It does not misreport a completed source addition as an operation that should be retried. Pending cleanup stages must be inspected before another authoring operation.
