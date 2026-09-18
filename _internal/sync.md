# Project operations

`internal/project` coordinates persisted snapshots and generated output:

- `Sync` imports all configured Git sources, validates and renders the full result, then installs vendor and generated files together.
- `Build` verifies persisted source identity and original-byte digests, then regenerates offline.
- `Check` compares expected generated content without writing. The CLI also supplies its managed project guide to `CheckWithFiles` so both checks share one optimistic snapshot.

`WithWriter` and `Writer.Apply` hide lock ownership, interrupted-operation recovery, staging, installation, and rollback. Preserve these deep operations rather than spreading their protocol across callers. Concurrent changes cause refusal before installation; once replacement begins, it completes or rolls back.

The [sync and recovery reference](../docs/src/content/docs/reference/sync.md) owns the filesystem contract and limitations. Tests include concurrent changes, malformed journals, cancellation, byte-exact retention, and failed multi-source sync. Run the [Go validation commands](../README.md#validate-changes).
