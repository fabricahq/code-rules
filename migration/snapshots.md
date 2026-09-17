# Persisted source snapshots

`project.EncodeSnapshots` produces vendor file bytes and one version-1 `_source.json`
per source. `project.DecodeSnapshots` verifies the exact inventory, digests, and
configured source identity before returning owned bytes. Both operate in memory
and return no partial result on failure. They accept parsed configuration.

The source record retains the TypeScript field names and SHA-256 digests. Missing
`groupSelection` falls back to explicit recorded `groups`; that cannot satisfy a
wildcard request. Ref/version selection, commit shape, selected release, and
resolved groups must agree. Unknown/null fields, unsupported versions, path
collisions, reserved records, missing/changed/extra files, and removed sources fail.

Manifest and group metadata must appear in the inventory. Rule/library content
validation remains the catalog loader's responsibility. Hashes prove consistency
with the supplied record, not that an authorized Git repository produced it.
Git origin verification and filesystem updates remain later capabilities.

The Go reader deliberately rejects unknown and null record fields, consistent
with the previously approved strict parsing policy. Generated JSON leaves <, >,
and & readable. It preserves raw binary and CRLF file bytes. Serialization order
and whitespace need not match TypeScript; logical records and original file bytes do.
Existing authored format differences (including scalar whenToRead and HashiCorp
constraints) remain explicit migration differences, not automatic conversions.

The walkthrough at `/walkthrough/pr28` shows source files and the generated record.
Its `recordChanges` and `fileChanges` are lab-only edits applied after encoding.
A null edit deletes that field/file. Error scenarios return no partial snapshot.

Validation covers round trips, independent byte ownership, deterministic output,
empty sources, legacy records, exact constraints, malformed records, source and
selection changes, binary data, CRLF bytes, and portable path collisions.
This supplies slice evidence for `files.snapshots.01` through `.04`; integrated
offline acceptance will follow in the project build/check slice.

## Validation evidence

- Go race tests, vet, and Staticcheck pass across the migration packages.
- The native lab exercises all nine scenarios, including corrupted content and binary bytes.
- Browser checks verified the original file viewer, generated source record, and corruption error.
- The existing reference comparisons pass: 661 shared rules cases, 283 approved behavior differences, real-Git version selection, and 13 Markdown link cases.
- A direct TypeScript snapshot comparison confirmed matching logical version-1 records and original-byte digests for text/CRLF and binary fixtures. Independent review also checked both read directions against the pinned reference.
- Independent review identified a reserved-record case/path collision. Encoding now validates the complete inventory after adding its record; regression cases cover both a case alias and a descendant of `_source.json`.
