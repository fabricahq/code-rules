# Sync complete projects

`project.Sync(ctx, options, gitOptions)` fetches every configured source, validates its selected library content, resolves local definitions, and prepares generated output. Only after that succeeds does it install both `vendor/` and `generated/` through the recoverable project writer.

The returned change report prefixes each path with its managed directory. Repeating a sync against unchanged sources returns empty lists. Removing a source from configuration removes its retained files on the next successful sync. Local files and configuration are never rewritten.

Import or render failures return an error and no partial change report. The writer checks for concurrent edits before installation and preserves recovery evidence if an interrupted write cannot safely be reconciled. Multiple directory renames do not become one atomically visible filesystem operation.

Shared snapshot values now belong to `internal/library`. `project.Snapshot` remains an alias for callers of the existing persistence API. This lets project orchestration call the importer without an import cycle.

## Evidence

Real Git tests cover exact license and binary bytes through sync and persisted snapshot decoding, repeat sync, a moved tag, source retirement, offline checking without Git on PATH, malformed local rules, failed fetch, and cancellation. The walkthrough uses real local Git repositories and a disposable project, showing a change report plus complete before/after files.

Run `go test -race ./internal/project ./internal/imports ./cmd/rules-lab` and open `/walkthrough/pr33` in the native lab. CLI commands remain the next slice.
