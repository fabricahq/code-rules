# Migration feedback

Status: PRs #11 through #14 were human-approved and merged. Repository addresses and links are the current slice.

## Rules used

Corpus: `https://github.com/josh-padnick/code-rules` at `e2166f90333157fd3e14c24d3e43287ece858e4b`.
Private rule contents stay outside this public repository. Relevant paths:

- `README.md`
- `practices/code-design/express-operations-as-meaningful-steps.md`
- `typescript/write-focused-user-oriented-tests.md`
- `typescript/avoid-snapshot-tests.md`
- `typescript/avoid-type-and-non-null-assertions.md`
- `typescript/distinguish-null-from-undefined.md`
- `typescript/keep-functions-pure-and-focused.md`
- `typescript/prefer-single-object-function-args.md`
- `typescript/use-unknown-instead-of-any.md`
- `typescript/testing-cover-degenerate-and-boundary-cases.md`
- `typescript/annotate-public-return-types.md`
- `typescript/prefer-immutable-data.md`
- `typescript/model-complex-variants-as-discriminated-unions.md`
- `go/testing-cover-degenerate-and-boundary-cases.md`
- `go/testing-regression-test-every-bug-fix.md`
- `go/errors-include-useful-diagnostic-data.md`
- `go/errors-use-contract-errors-deliberately.md`
- `go/comment-non-obvious-struct-fields.md`
- `go/comments-package-doc-vs-file-header.md`

The user-selected corpus supersedes the product AGENTS.md link to the Fabrica application corpus for this migration.
No application-domain overlay applies merely because the CLI uses the Fabrica GitHub organization.
The product's `_internal/rules/comment-role-result-and-constraints.md` controls TypeScript comments.
For future Go files, use native Go package/file comments; TypeScript `@fileoverview` and TSDoc syntax do not transfer to Go.
The enduring Go decisions now live in [Go conventions](../_internal/go-conventions.md); their implementation is reviewed with the first Go slice.

Runtime filesystem observations are intentional comparison evidence, not stored golden snapshots that can be regenerated to approve behavior.
The comparison itself is pure; the runner owns filesystem and process effects.

## PR #11 review follow-up

The user requested logging and error conventions after reviewing independent feedback.
The [Go conventions](../_internal/go-conventions.md) record the implementation choices; keep future edits there.
The portable log-once and structured-logging ideas informed the design. Application-specific logging modules and environment variables were not imported.
Relevant additional corpus paths:

- `_domains/fabrica/go/logging-log-once-at-the-boundary.md`
- `_domains/fabrica/go/logging-use-slog-with-structured-attrs.md`

Evidence: `internal/logging/logging_test.go`, `cmd/rules-lab/main_test.go`, and the exact shared identity expectations.
The review fixes retain typed validation errors, preserve full path context, and distinguish HTTP body-size failures from read failures.
No new error-code taxonomy or dependency wrapper is introduced.

## Proposals from this iteration

- Keep independent intended-value assertions alongside differential results. Identical reference runs can share a failure.
  Evidence: `_tools/go-migration.test.ts`, test for both implementations returning the same wrong version.
- Preserve incomplete coverage explicitly. A scenario exercising a capability must not approve all of that capability's requirements.
  Evidence: report `coverage.exercisedBy` and pending `requiredScenarios`.

## Draft assertion corrections

Before this first proposal was review-ready, the initial self-check exposed three incorrectly drafted expectations.
The initial config includes `schemaVersion: 1` (`src/authoring/project.ts`).
Missing noninteractive input exits 2 (`src/authoring/cli.ts` and `src/cli.ts`).
Imported notices use `licenses/notices/001.md` (`src/builds/license-output.ts`).
The assertions now match those source contracts. No baseline, production behavior, comparator, or approved difference changed.

## Group metadata slice

- Treat a reproduced Unicode mismatch as an open defect, even when earlier slice notes excluded that edge case from coverage. Do not turn a coverage limit into an approved behavior change.
  Evidence: the lone-surrogate finding, approved rejection, and regression coverage recorded in [group metadata](group-metadata.md#unicode-behavior).
- Preserve format semantics at the JSON boundary. Raw fields retain case-sensitive keys and last-key-wins behavior. Reject unknown fields in sorted order before decoding their values.
  Evidence: metadata cases for field case, repeated keys, and a rejected unknown field containing `1e400`.
- Distinguish document syntax from adapter syntax. Sending metadata text as a string lets the lab invoke Go on malformed documents.
  Evidence: adapter tests for malformed metadata text versus an incorrectly typed envelope input.
- Use one nonblank string for group and rule reading guidance, as requested during review. Group metadata trims surrounding whitespace and rejects legacy arrays.
  Evidence: `internal/rules/group_metadata_test.go` and the HTTP metadata test.

## Rule document boundary slice

- Keep each interactive walkthrough scoped to its own PR. Earlier operations stay in regression tests and the development adapter.
- Separate preservation of authored text from interpretation of YAML fields. A successful split does not establish a valid complete rule.
  Evidence: direct Go cases for empty captures and uninterpreted YAML, plus exact comparisons against the pinned rule parser for complete rules.

## Complete rule parser slice

- Plan about 20-30 additional capability-sized PRs. Keep helpers together when they complete one reviewable operation.
- Keep YAML syntax parsing in a maintained dependency. Adapt scalar types at the node boundary so legacy YAML coercions cannot silently accept invalid rule fields.
- Compare both sides with independent expectations. Record user-approved diagnostic differences separately; do not normalize error strings or advance the reference.
- Preserve the original metadata and body independently of interpreted values. URL normalization affects attribution data, never the authored text.
- The user explicitly requested strict rule frontmatter during review. Reject unknown top-level and attribution fields, report names deterministically, and retain separate expectations for the permissive reference.

Evidence and the approved Unicode behavior live in [complete rule parsing](rule-parser.md).

## Repository address slice

- Validate authored path segments before URL normalization. Otherwise a URL parser can erase traversal and encoded separators before validation sees them.
- Keep duplicate-source identity separate from the transport address. Known hosts can share identities across standard transports; generic sources retain transport and path distinctions.
- Return an explicit absent web convention instead of guessing a custom host's routes.

Evidence: [repository address fixtures](../tests/migration/repositories/cases.json) and [slice scope](repositories.md).

## Version validation and constraints

- Use Go errors to distinguish invalid input from a valid negative answer. TagVersion returns an error for an invalid version; constraint matching returns false without an error when a valid version is outside the range.
- An explicitly approved API or format improvement can supersede TypeScript behavior. Keep independent intended results and pinned reference results separate instead of disguising the change as parity.
- Use the requested dependency's native language when that is the product decision. The user chose HashiCorp syntax and matching semantics over npm compatibility.

Evidence: [version constraints](version-constraints.md) and its shared fixtures.

- PR #17 review: trim surrounding whitespace in version constraints, preserving internal spacing. This user-approved normalization applies to parsed configuration versions too.

- Keep dependency integration coverage small: test the behavior we add and representative delegation, rather than duplicating go-version's operator and ordering tests. The PR #17 walkthrough follows the same scope.
