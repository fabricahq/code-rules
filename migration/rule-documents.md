# Rule document boundaries: third Go slice

Status: implemented and locally validated on `codex/go-rule-documents`, awaiting human review.

## Baseline and scope

PR #12 was human-approved and merged into `go-migration` at `5c2c10a23cf556d9533bed316397338213f22fbd`. This slice starts there.
The TypeScript reference remains `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`; the engineering corpus remains `e2166f90333157fd3e14c24d3e43287ece858e4b`.

Add `rules.SplitDocument(text, location string) (DocumentText, error)` in `internal/rules/document.go`.
It separates the required YAML frontmatter envelope from a rule's Markdown body. `DocumentText` returns `Frontmatter` and `Body` as the exact captured text, without delimiter lines.

The source behavior is `documentText` in `src/builds/rule-document.ts`:

- Require an opening `---` at the very start, followed by LF or CRLF.
- Close on the first matching `---` line preceded by LF or CRLF, followed by LF, CRLF, or end of input.
- Preserve captured whitespace, Unicode, and newline bytes. Do not trim or normalize either part.
- Return the existing typed validation error when the envelope is missing or malformed.
- Allow empty captures and uninterpreted YAML. Splitting does not establish that a rule is valid.

YAML syntax, required rule fields, impact values, tags, attribution, path validation, Markdown interpretation, file reads, and file writes are outside this slice. The full rule parser will compose those checks later.
No dependency is added. This is preparatory evidence for `builds.resolve` and `builds.render.05` (metadata preservation), not completion of either capability or an acceptance scenario.

## Verification

Remeasure the merged baseline with Go tests and all 182 shared function cases.
Add independent explicit expectations for delimiter and byte-preservation cases. For cases with valid rule metadata, compare Go's result with the raw metadata/body returned by the pinned public TypeScript `rule` function. Malformed-envelope cases fail before that function reaches YAML parsing.
Test splitter-only behavior, including empty captures and uninterpreted YAML, directly in Go. The full TypeScript rule parser rejects some of those inputs later; the narrower Go function intentionally does not run those later checks. Do not present these tests as full rule-parser parity.

Keep every earlier regression case in the combined runner. Do not change the reference, normalize output, or approve new behavior differences.
Run formatting, Go vet, Staticcheck, race tests, shared comparisons, repository checks, installed-package checks, independent review, and browser exercises.

## Interactive review

The browser page shows only this PR's new split operation. Earlier adapter operations and regression tests remain available to automation, but do not appear in this walkthrough.
Provide editable Markdown and examples for ordinary frontmatter, CRLF, whitespace preservation, malformed delimiters, empty body, and uninterpreted YAML.
Show returned strings plus visible newline/byte information so exact preservation is inspectable. Invoke the real Go function on every request.

## Handoff

Validation completed:

- 27 new shared cases compare exact results with the pinned TypeScript rule parser; all 209 shared cases pass with no new approved differences.
- Six direct Go cases cover the narrower splitter contract, including empty captures, uninterpreted YAML, whitespace, and raw bytes.
- Go tests with the race detector, vet, and Staticcheck v0.8.1 pass on Go 1.27.1.
- `bun run check` passes, including 444 tests, formatting, lint, typechecking, documentation build, and local link checks.
- All six installed-package tests pass.
- Independent read-only review reports no actionable findings.
- All ten browser presets invoke the real Go function successfully or produce their expected validation error. CRLF and whitespace remain visible in the returned JSON.

Run the interactive review with `go run ./cmd/rules-lab -serve -port 4391`, then open `http://127.0.0.1:4391/`.

Open one PR into `go-migration` and stop for human review. Do not merge it without new approval.
After this slice, the next recommendation is YAML frontmatter interpretation and rule metadata validation. That work must evaluate a YAML dependency against the pinned reference before adopting it.
