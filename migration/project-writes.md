# Recoverable managed project writes

This slice adds `project.ReadTree`, `RequireIdle`, and `WithWriter`. Inside the
writer callback, `Apply` replaces complete `vendor` and/or `generated` trees.
The caller owns the open filesystem root. Authored configuration and local rules
are never replacement targets.

## Ownership and failure behavior

- The exclusive lock records host, PID, and a random invocation token. Only ESRCH
  for an owner on this host allows reclamation. Incomplete, foreign, live, and
  permission-denied ownership fails closed. A recovery claim serializes contenders.
- Reads reject observed symbolic links, hard links, special files, unsafe portable
  paths, and case/Unicode collisions. Bounds are 64 MiB per file, 256 MiB per tree,
  30,000 entries, and 64 path components. Empty directories count as input.
- Apply stages all output, checks caller inputs and target digests, then writes a
  version-2 journal before any live replacement. Failed operations restore prior
  trees when doing so cannot destroy later edits.
- Recovery validates every restoration first, then moves live output into a
  transaction-owned holding directory and validates the displaced bytes before
  deleting anything. Apply also verifies the backup after its rename. Changed current output or backups
  remain available for manual recovery. A malformed journal is not permission to
  delete a tree.
- Cancellation during replacement runs recovery without the canceled context before
  returning. If later edits prevent safe rollback, the error requests manual recovery
  and retains the affected files.
- After the synced commit marker, cleanup is cleanup-only. Retiring the journal
  before deleting backups prevents a later retry from rolling back committed work.
- This does not promise simultaneous visibility across two renames or durability
  against power loss. External replacement of the owned lock/control directories
  while a writer is active, or writes through previously opened descriptors after
  their files have moved into transaction-owned storage, are unsupported. Root handles confine filesystem access.

Go reads legacy version-1 journals and uses the same tree digest, including
UTF-16 filename ordering. Review found a reference defect: contradictory `existed`
and `before` values could delete intact output. Go rejects that state and keeps
recovery material. This safety correction intentionally does not reproduce the defect.

## Review and tests

Run `go run ./cmd/rules-lab -serve` and open `/walkthrough/pr29`. Each invocation
creates a disposable project. Inspect actual before/after content, proposed output,
and the change report. Error responses use `ok: false`; `observation` is separately
labelled filesystem evidence captured after the failed call. Unexpected setup or
filesystem failures remain HTTP 500 responses with redacted boundary logging.

Scenarios cover initial/repeated apply, stale-file removal, busy writers, concurrent
edits, second rename failure, cancellation, invalid targets, and unsafe paths. Two additional presets seed interrupted transaction state and show successful recovery or preserved post-interruption edits.
Automated tests also kill a child running the real Apply operation after its first
backup rename, reclaim its dead lock, and restore the original output. A private
rename boundary makes that test deterministic without a production fault option.
Synthetic interruption tests cover post-interruption edits, interrupted quarantine
recovery, and committed cleanup. Deterministic late-edit regressions exercise both
backup and rollback rename boundaries. Additional tests preserve an empty target
created before installation and verify rollback after cancellation between replacements. The rollback holding directory extends the
journal with version 2; legacy journals are atomically upgraded before quarantine.
TypeScript rejects version 2 and preserves the recovery material instead of
deleting a holding directory it cannot interpret.

Full project build/check and imports remain later slices. No production CLI is
introduced here. This PR depends on the snapshot slice for its shared path checks.

Validation: the full Go race suite, vet, and Staticcheck pass. Native presets and
browser checks cover successful apply and rollback. Independent review reproduced
and then verified both the malformed-journal and Unicode-ordering corrections.
