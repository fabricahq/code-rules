# Offline project build and check

`project.Build(ctx, options)` and `project.Check(ctx, options)` compose the native
Go pipeline against an actual project. `ConfigPath` defaults to
`.code-rules/config.json`; its parent contains `local`, `vendor`, and `generated`.
Custom configuration filenames use the same sibling layout.

Both operations validate configuration, verify persisted snapshot identity and
bytes, load native library semantics, resolve local definitions, and prepare output.
The loaded catalog's exact rule and supporting bytes must equal the owned verified
snapshot. This closes a transient-edit gap that a final filesystem recheck alone
cannot detect. Source digests establish integrity against the record, not Git origin.

Build owns the writer lock, rechecks original inputs/output, and replaces only
`generated`. Check refuses active writers or pending recovery and returns the same
added/changed/removed paths without writes, locks, recovery, Git, or network access.
Paths are relative to `generated/`; empty arrays mean matching output. Invalid input
returns an error, never a clean check result. Repeated builds return empty changes.

Defaults preserve 750-line indexes, whole short-group delivery at 8 KiB, and an
explicit development tool version when none is supplied. Configuration, snapshots,
local precedence, licensing, natural ordering, and rendering use the existing Go
packages. No TypeScript fallback or production Cobra command is introduced.

## Walkthrough and validation

Run `go run ./cmd/rules-lab -serve` and open `/walkthrough/pr30`. Editable local and
library fixtures are installed into a disposable project. Optional setup builds and
post-setup edits run before the operation being reviewed. Results expose the native
change report plus actual before/after project files.

Ten presets cover local/imported builds, missing/clean/stale checks, repeat build,
invalid local rules, modified vendor bytes, pending recovery, and a custom config.
Tests remove PATH to prove Build/Check do not invoke Git, Node, Bun, or shell tools.
They compare the complete tree before/after checks and preserve old output on errors.
Independent review reproduced a transient vendor edit/revert; regression checks now
reject parsed bytes differing from the verified immutable snapshot.
