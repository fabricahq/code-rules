# Exact refs and version tags: sixth Go slice

Status: implemented on `codex/go-refs`, awaiting review. The walkthrough covers only this slice.

Base: `ef88c4c6e7d7061da428a8c5783e84e7b3c6906e`, the approved PR #15 merge into `go-migration`.
TypeScript reference: `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`.
Engineering corpus: `e2166f90333157fd3e14c24d3e43287ece858e4b`.

## Scope and calls

```go
ref, err := rules.ParseGitRef("refs/tags/deadbeef", "sources.team.ref")
if err != nil {
    return err
}
// ref.Kind is rules.GitRefTag; ref.Name is "refs/tags/deadbeef".

version, err := rules.TagVersion("v1.2.3-beta.1+build.007", "tag")
if err != nil {
    return err
}
// version is "1.2.3-beta.1+build.007".
```

`ParseGitRef(ref, location string) (GitRef, error)` accepts a 40-digit hexadecimal commit SHA or an exact tag, with an optional `refs/tags/` prefix. Commit SHAs become lowercase. Tag names retain their spelling and get the explicit prefix. A bare `main` means a tag; branches are not inferred. Abbreviated commit-shaped names require the explicit tag prefix. Failures return a zero `GitRef` and a typed `ValidationError` at the caller's field location.

`TagVersion(tag, location string) (string, error)` validates a strict, complete semantic version with one optional lowercase `v`. It preserves prerelease and build suffixes. Invalid input returns an empty string and a typed `ValidationError` at the caller's diagnostic location. The lab returns `ok: false` for these errors and `ok: true` with a version string for valid input. The function expects an unqualified tag name, not `refs/tags/...`.

Future tag-scanning callers can use `errors.As` to skip validation failures while continuing to report unexpected errors. They do not need a replacement parser. Callers requiring a version can report the same validation error directly.

The implementation preserves the pinned npm parser's core-number and length limits. It adds no runtime dependency. These functions do not fetch repositories, verify revision existence, order versions, expand ranges, or read or write files.

## Contract evidence

This supplies partial evidence for `formats.versions.01`: exact refs and strict version-tag validation. npm range parsing and matching, including prerelease exclusion, remain pending. `formats.versions.02` and `.03` still require tag resolution and ambiguity handling. No full capability is marked complete.

The baseline and comparison policy are unchanged. The user approved changing TagVersion from recognition to validation: 28 non-version fixtures retain the reference null result separately from the Go error. Accepted-version syntax is unchanged. The earlier rule parser's documented CR-only YAML gap remains separate.

The 88 new shared fixtures specify independent values or exact errors. They cover full and abbreviated SHAs, tag qualification, branch rejection, unsafe components, Unicode names, complete versions, prerelease/build identifiers, leading zeros, numeric limits, and the 256-character boundary. The suite also retains all 491 earlier cases. Direct Go tests check typed errors, zero failure results, and caller-owned locations. HTTP tests distinguish version validation errors from adapter failures.

## Walkthrough and validation

```sh
go test -race ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
go build -o dist/rules-lab ./cmd/rules-lab
bun tests/migration/compare-rules.ts dist/rules-lab
bun run check
bun run test:package
go run ./cmd/rules-lab -serve -port 4391
```

The browser uses the existing Code Rules gallery styling and invokes native Go. Try Full SHA, Main is a tag, Short hex, and Explicit hex tag. Then switch operations and compare Prerelease + build, Partial version, and the numeric boundary. Earlier functions remain in regression tests but are absent from this walkthrough.

The implementation follows the existing [Go conventions](../_internal/go-conventions.md), particularly typed errors, useful diagnostic locations, boundary tests, and concise function comments. No new logging is needed in pure parsing functions.

## Next boundary

npm-compatible range parsing is the next prerequisite for complete configuration parsing. An isolated evaluation of `deps.dev/util/semver` found differences from the pinned npm parser for empty OR arms, leading zeros, wildcard placement, numeric bounds, and whitespace. It is not a drop-in replacement. No dependency or partial range implementation from that evaluation is included here.

Human review and merge separate migration iterations.

### Local evidence

- All 579 shared comparisons pass: 88 new cases and 491 earlier cases. There are 196 explicit differences: the 168 earlier cases and 28 user-approved version-validation errors.
- Go race tests, vet, and Staticcheck v0.8.1 pass on Go 1.27.1.
- `bun run check` passes, including all 444 tests, formatting, lint, typecheck, documentation build, and link checks.
- All six installed-package tests pass.
- All 20 browser examples return expected results. Desktop and 390px layouts have no horizontal overflow or console errors.
- Local autoreview could not run: automatic approval review rejected its unspecified external code destination. The PR requests the explicitly authorized CodeRabbit review, with Devin as fallback. Independent review is pending.
