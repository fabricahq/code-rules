# Identity syntax: first Go slice

Status: implemented and locally tested; submitted for human review. Independent automated review and human merge remain pending.

## Scope and revisions

This slice implements three pure operations in [internal/rules](../internal/rules/identity.go): group ID validation, rule-path validation with owning-group extraction, and parsing explicit/wildcard group selections.
It introduces a root Go module with no third-party dependencies. The TypeScript runtime is unchanged.

- Branch: `codex/go-identities`; PR target: `go-migration`.
- Exact integration base: `5ba1d51c7214c9d22979a882fb6fb46b2120d8fe`, the human-authorized merge of PR #10.
- TypeScript reference: `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`.
- Rule corpus: `josh-padnick/code-rules` at `e2166f90333157fd3e14c24d3e43287ece858e4b`.
- The submitted head is the PR head SHA. Evidence generated after commit records that exact revision outside this source document.

No product CLI, Cobra dependency, wildcard expansion, group discovery, filesystem effects, or import/rendering code is introduced.
Cobra is the chosen library for the future product CLI. The `flag` usage here belongs only to the development lab.

### Contract mapping

The stable inventory remains unchanged. This slice supplies partial evidence without marking any full capability complete.

| Contract | Evidence in this slice | Still pending |
| --- | --- | --- |
| `formats.identities.01` | Empty, one, and multiple selected IDs | Full group/metadata inputs |
| `formats.identities.02` | Explicit lists and the three wildcard strings | Expansion against library contents |
| `formats.identities.03` | Duplicate rejection and error ordering | Group existence and missing groups |
| `formats.identities.04` | Traversal, portable separators, ASCII sorting | Case collisions and general UTF-16 ordering |
| `formats.identities.05` | Missing/null group values both fail | Complete configuration shapes and unknown fields |

The function boundary supports Unicode scalar text. JSON containing lone UTF-16 surrogates is not parity-verified: Go's JSON decoder replaces them with U+FFFD while JavaScript preserves the code unit. This is an open compatibility problem for full configuration migration, not an approved difference. Do not expand the scope's acceptance claim until it is resolved and tested.

## User-approved diagnostic improvement

The user requested clearer reasons for invalid examples, while keeping the validation code simple.
Go now returns `group: invalid group ID "techs/assets": "assets" is a reserved group name`.
The same explanation applies to `practices/assets` and to either ID in an explicit selection. Syntax validation still runs first; other malformed IDs remain syntax errors.

Malformed group IDs now report the existing group regular expression. Rule errors explain the required `.md` location, the reserved `assets` directory, or the existing filename regular expression. Containment errors spell out the path restrictions; selection shape errors list the accepted array and wildcard forms.
The messages reuse the compiled expressions and existing validation checks. Acceptance rules and validation order are unchanged; no character-by-character diagnosis is introduced.

The 59 affected shared cases store both `referenceExpected` (the unchanged pinned TypeScript message) and `expected` (the approved Go message). Each side is checked exactly, without normalizing or ignoring diagnostics.
This is a function-level diagnostic change only. The global CLI comparison policy and pinned TypeScript source are unchanged; future CLI integration must carry this approval into its own exact expectations.

## Package ownership

`internal/rules` owns rule and group concepts and their invariants. The name describes the subject rather than the parsing activity. This slice implements only identity syntax and group selections.
`GroupFromPath` makes its path input explicit; `ParseGroupSelection` retains the group qualifier. Repository fetching, project configuration, filesystem operations, and rendering will receive separate placement decisions as they are implemented.
The development adapter keeps its existing `ruleGroup` JSON operation name; it now calls `rules.GroupFromPath`.

## API and Go patterns to review

```go
func ValidateGroupID(value, location string) error
func GroupFromPath(path, location string) (string, error)
func ParseGroupSelection(input json.RawMessage, location string) (GroupSelection, error)
```

- **Direct functions and ordinary return values.** No service interfaces, dependency containers, filesystem abstractions, or generic result wrappers in the rules package.
- **One contract error.** `*ValidationError` lets a future CLI distinguish invalid input with `errors.As`. It carries a field location and the reference diagnostic, with the approved clarifications above. Successful validation returns `nil`.
- **JSON belongs at the configuration boundary.** `ParseGroupSelection` accepts `json.RawMessage` because shape, null, missing values, and element types are part of that operation. Its decoded `any` values remain local; callers receive a concrete `GroupSelection`.
- **Explicit selection invariant.** A nonempty `Pattern` means an unexpanded wildcard and nil `Groups`. Otherwise `Groups` is non-nil, sorted, and may be empty. The zero value is not a successfully parsed selection; callers check the error first.
- **Stable validation order.** Paths check containment, then rule syntax, then group syntax. Selections check all text entries, duplicates, and then group IDs at their original indices before sorting. Parsing owns its returned slices.
- **Portable path meaning.** No `filepath.Clean`, Unicode normalization, case folding, or filesystem lookup. A rule path uses `/` on every OS.
- **Narrow compatibility helpers.** Diagnostic quoting follows JSON spelling; whitespace matches JavaScript `trim`. Tests cover BOM/NEL and escaped controls because the obvious Go defaults differ.

The relevant rule paths are recorded in [feedback.md](feedback.md#rules-used), especially the Go error, boundary-testing, struct-comment, and package-comment rules; meaningful operation steps; and the TypeScript testing, unknown-value, immutability, and return-type rules.
Native Go comments adapt the existing TypeScript comment convention. Private corpus contents are not copied into this repository.

## Interactive walkthrough

From the repository root:

```sh
go run ./cmd/identity-lab -serve
```

Open the printed loopback URL. The page follows the Code Rules gallery's neutral colors, thin borders, and typography. It includes editable inputs, presets, function signatures, returned values, typed errors, the exact request, and a walkthrough of the implementation boundaries.
The page is embedded in the binary and uses no remote scripts or assets. Every invocation calls the native package through a fixed local endpoint. Inputs cannot select a file path or command to execute.
The development server binds to `127.0.0.1`, applies the standard library's cross-origin protection, and bounds request size and HTTP timeouts.

For automation, the same adapter reads JSON lines on stdin:

```sh
go build -o /tmp/code-rules-identity-lab ./cmd/identity-lab
printf '%s\n' '{"operation":"ruleGroup","input":"techs/go/errors.md","location":"rule"}' | /tmp/code-rules-identity-lab
```

The response is `{"ok":true,"value":"techs/go"}`. Validation failures use `ok: false` with a `ValidationError`; malformed adapter requests use `AdapterError`.
The group-ID adapter echoes its input on success because the underlying Go validator returns only an error.
Domain errors stay in the JSON response and do not terminate the stream. Process exit 1 means the adapter could not read/write the stream or run the server.

## Validation

```sh
go vet ./...
go test -race ./...
go build -o /tmp/code-rules-identity-lab ./cmd/identity-lab
bun tests/migration/identities/compare.ts /tmp/code-rules-identity-lab
bun run check
bun run test:package
```

Go 1.27.1 was used locally; CI reads the Go version directly from `go.mod`.
In a restricted local environment, set `GOCACHE` and `GOMODCACHE` to writable temporary directories. This changes cache locations only.

The [86 shared cases](../tests/migration/identities/cases.json) contain independent expected results. Go tests invoke the package directly. The [comparison runner](../tests/migration/identities/compare.ts) invokes the existing TypeScript functions and compiled Go adapter and checks each against those expectations.
The runner rejects tracked or untracked TypeScript input drift, rejects approved-difference changes, bounds candidate execution, checks response count, and reports the binary hash. The Go process receives a PATH with no interpreters available.
The source and lockfile pin plus binary hash are provenance inputs, not proof that an arbitrary supplied binary came from the declared Go sources.

Additional Go tests cover result ownership, malformed JSON, adapter errors, real HTTP dispatch, cross-origin rejection, and oversized requests.
The native package imports no filesystem/network/process APIs; the tests do not claim a syscall audit. End-to-end file parity remains owned by the existing migration harness and later capabilities.

The browser was exercised with valid group IDs, reserved names, nested rule paths, traversal, sorted selections, duplicates, empty-entry precedence, and malformed JSON. Returned results came from the live binary.

Independent automated review did not complete: the first attempt exceeded the reviewer context window, and automatic approval review blocked a retry over possible private-rule disclosure. No independent-review approval is claimed.

Local logs are retained under `/private/tmp/code-rules-migration-evidence/`; paths are machine-local and should be regenerated by other reviewers.

## Handoff

Stop after this PR for human review of the API and Go patterns. Do not merge or begin another capability automatically.
After approval, the next candidate slice is group/rule metadata parsing that consumes these identities. Reconfirm contract boundaries and the unresolved Unicode compatibility before starting.
