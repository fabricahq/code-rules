# Migration feedback

Status: proposed with the harness PR; no human-approved lessons yet.

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
That language adaptation remains part of the first Go slice's review.

Runtime filesystem observations are intentional comparison evidence, not stored golden snapshots that can be regenerated to approve behavior.
The comparison itself is pure; the runner owns filesystem and process effects.

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
