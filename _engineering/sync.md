# Project operations

`internal/project` owns the complete lifecycle of a consuming project's Code Rules state:

- `Initialize`, `AddLocalGroup`, `AddLocalRule`, and `AddSource` create or update authored project state without fetching libraries or generating output.
- `HasLocalRuleGroup` supports prompts with an advisory read; rule creation revalidates under writer ownership.
- `Sync` imports all configured Git sources, validates and renders the full result, then installs vendor and generated files together.
- `Build` verifies persisted source identity and original-byte digests, then regenerates offline.
- `Check` compares generated output and the managed project guide in one optimistic snapshot, without writes or Git access. It returns typed problems with config-relative paths and repair actions. Staleness is a report; invalid input or concurrent edits are errors. The CLI formats messages and repair commands and selects the exit status.

`Sync` requests complete imports through `imports.ImportLibraries`; Git revisions and temporary repository ownership stay private to `imports`. Both `Build` and `Sync` request complete output through `build.Generate`.

Snapshot encoding, guide rendering, file inventories, and the combined check machinery are private implementation details. Ordinary callers request complete project operations.

`internal/filetxn` owns the shared storage protocol. Its `WithWriter` and `Writer.Apply` operations hide lock ownership, interrupted-operation recovery, staging, installation, and rollback. Preserve these deep operations rather than spreading their protocol across callers. Authored-file publication uses its separate `Edit` operation because exclusive creation and one guarded replacement differ from replacing whole managed trees. Both operations share writer ownership. Concurrent changes cause refusal before installation; once replacement begins, it completes or rolls back.

The [sync and recovery reference](../docs/src/content/docs/reference/sync.md) owns the filesystem contract and limitations. Tests include concurrent changes, malformed journals, cancellation, byte-exact retention, and failed multi-source sync. Run the [Go validation commands](../CONTRIBUTING.md#validate-changes).
