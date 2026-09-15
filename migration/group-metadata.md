# Group metadata: second Go slice

Status: implemented on `codex/go-group-metadata`; draft pending a Unicode compatibility decision. Do not merge yet.

## Scope and baseline

PR #11 was human-approved and merged into `go-migration` at `6bfcaf608bc5ce9c36af4c3c27c02d751a7fdd30`.
This branch starts at that exact integration commit.
The TypeScript reference remains `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`; the rule corpus remains `e2166f90333157fd3e14c24d3e43287ece858e4b`.

Implement `rules.ParseGroupMetadata(input json.RawMessage, location string) (GroupMetadata, error)` in `internal/rules/group_metadata.go`.
It parses one group's JSON text into its display name, description, and ordered reading guidance.
This is the next part of metadata parsing recommended after the identity slice; rule Markdown/frontmatter parsing follows separately.

The source contract is `groupMetadata` and its guards in `src/formats/validation.ts`.
This adds evidence for `formats.identities.05` (unknown fields and absent/null values), and a prerequisite for `builds.resolve.04` (local metadata).
Neither full capability nor any additional acceptance scenario is marked complete.
No file discovery, wildcard resolution, filesystem writes, product CLI, or new dependency is included.

## Implementation and review points

- Return a concrete `GroupMetadata` and the existing `*ValidationError` on invalid input.
- Require nonblank `name` and `description` strings, trimming surrounding whitespace.
- Require a `whenToRead` array of distinct, nonblank strings. Trim surrounding whitespace before checking duplicates; preserve order; an empty array is valid.
- Reject `license` and `licenses` whenever present, including null. Ignore other unknown fields, as the reference does.
- Preserve validation order: JSON syntax, object shape, license declarations, name, description, guidance item text, then duplicates.
- Decode a map of raw JSON fields rather than a struct so keys stay case-sensitive, unknown numeric values do not overflow, and duplicate JSON keys use the last value.
- Keep JavaScript whitespace handling consistent with the first slice.
- Return owned slices; failed parsing returns the zero result.

## Verification and interactive review

Add explicit metadata cases under `tests/migration/group-metadata/` and invoke them directly from Go tests.
Move the shared comparison runner to `tests/migration/compare-rules.ts` and run both identity and metadata cases against the pinned TypeScript functions and native binary.
Keep the 87 identity cases and all exact diagnostic comparisons intact.

Extend the existing embedded `cmd/rules-lab` walkthrough using its current Code Rules gallery styling.
The new operation sends JSON document text as a string to the adapter, so malformed document JSON reaches the Go parser.
Show editable metadata, useful presets, returned structured values, errors with field locations, and an explanation of the parsing steps.

Run Go formatting, vet, pinned Staticcheck, race tests, exact reference/native comparisons, the repository checks, installed-package checks, and a real browser exercise.
Open one PR into `go-migration`, then stop for human review.

## Compatibility limits

Comparison covers Unicode scalar text, including valid surrogate pairs in JSON.
Autoreview found and the live lab reproduced a defect outside those cases: `{"name":"\ud800","description":"ok","whenToRead":[]}` returns a replacement character in Go, while TypeScript preserves the lone surrogate.
This is an unresolved text-preservation defect, not an approved difference. Passing the shared suite does not establish parity for this input.
The proposed resolution is to reject malformed Unicode with a clear validation error, keeping ordinary Go strings. That changes accepted inputs and requires human approval under the migration instructions. Exact preservation would instead require a compatible text representation and serializer.
The PR remains a draft until that choice is implemented, regression-tested, and reviewed.

## Run and review

```sh
go run ./cmd/rules-lab -serve
```

The page opens with group metadata selected. Try the valid document, empty guidance, preserved order and trimmed whitespace, missing name, null list, duplicate guidance, blank-before-duplicate, license declaration, unknown field, and malformed JSON presets.
The `groupMetadata` adapter operation takes document text in its `input` string. That lets malformed document JSON reach the Go function without breaking the outer request envelope.

The new parser uses only the Go standard library. Its private helpers each own one check: nonblank text and ordered, distinct reading guidance.
The existing identity API and logger conventions remain unchanged.

## Checks

```sh
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
go test -race ./...
go build -o /tmp/code-rules-lab ./cmd/rules-lab
bun tests/migration/compare-rules.ts /tmp/code-rules-lab
bun run check
bun run test:package
```

The shared suite contains 53 metadata cases plus 87 identity cases. Four cases record the user-approved trimming difference explicitly, retaining the original TypeScript expectations.
Cases cover empty/single/multiple guidance, whitespace and Unicode scalar text, key case and repetition, ignored unknown fields, absent/null/wrongly typed values, error precedence, and forbidden license fields.
Go tests also check that parsing does not mutate input, results own their storage, failures return no partial metadata, and HTTP preserves an empty guidance array.

Relevant engineering rule paths remain in [feedback.md](feedback.md#rules-used). The [Go conventions](../_internal/go-conventions.md) govern this implementation.
Private corpus contents are not copied into the product repository.

## Validation result

- Go formatting, vet, Staticcheck v0.8.1, and race tests passed.
- The 140 shared cases include the existing 60 approved identity diagnostic differences and four approved metadata trimming differences.
- `bun run check` passed, including 444 tests, formatting, lint, type checking, docs build, and link checks. Six installed-package tests passed.
- All ten metadata presets were invoked through the real browser lab. The editor fits the default document and resets to its beginning when selecting a preset.
- Supported read-only Codex autoreview with web disabled returned one finding: the Unicode defect above. It is accepted and unresolved; review is not clean.

Local evidence is retained under `/private/tmp/code-rules-migration-evidence/`: `metadata-check.log`, `metadata-package.log`, `metadata-parity.log`, `metadata-review.txt`, `metadata-review.json`, and `metadata-unicode-repro.json`. The last file records the exact request and differing results.
Reproduce it with `bun /private/tmp/code-rules-migration-evidence/metadata-unicode-repro.ts` against the retained native lab binary.

After this slice is resolved and human-reviewed, rule Markdown/frontmatter parsing is the recommended next slice. Do not begin it while this PR awaits review.

## User-approved trimming

The user requested trimming leading and trailing spaces during interactive review. Metadata names, descriptions, and reading guidance now trim surrounding JavaScript whitespace, using the existing whitespace predicate. Internal spacing and list order remain intact. Duplicate guidance is checked after trimming, so `"Same"` and `" Same "` are duplicates. Whitespace-only text remains invalid.

The TypeScript baseline stays pinned. Shared fixtures retain its exact original results in `referenceExpected`; no comparator normalization is introduced. This approval does not resolve the separate lone-surrogate finding.
