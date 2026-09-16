# Version constraints: seventh Go slice

Status: implemented on `codex/go-version-ranges`, awaiting review.
Base: `76e30d5c29881a4f74821a5b3bb23a3e1683ca96`, the approved PR #16 merge into `go-migration`.
TypeScript reference: `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`.
Engineering corpus: `e2166f90333157fd3e14c24d3e43287ece858e4b`.

## Scope and calls

```go
constraint, err := rules.ParseVersionConstraint(
    ">= 1.2.0, < 2.0.0", "sources.team.version",
)
if err != nil {
    return err
}
matched, err := constraint.Matches("v1.5.0", "release.version")
if err != nil {
    return err
}
// matched is true. constraint.String() returns the constraint with surrounding whitespace trimmed.
```

The user explicitly requested [HashiCorp go-version](https://github.com/hashicorp/go-version) and preferred its native syntax over npm's. The implementation pins v1.9.0 and delegates constraint parsing and matching to it. No npm compatibility parser or TypeScript fallback is included.

`VersionConstraint` holds private parsed comparisons and trimmed text. Parse failures return its zero value and a typed `ValidationError`. Matching a zero constraint also returns a validation error. A parsed constraint can be reused across versions without mutating it.

`Matches` requires a complete version tag, using the existing `TagVersion` validation and bounds. Invalid versions return `false` and an error. A valid version outside the constraint returns `false, nil`. The lab distinguishes these with `ok: false` versus `ok: true, value: false`.

These functions do not fetch tags, select the highest release, inspect files, or write output. All new functions and test callbacks have explanatory comments. Pure functions do not log.

## Approved language change

| Input | Go target |
| --- | --- |
| `>= 1.2.0, < 2.0.0` | Comma-separated AND |
| `~> 1.2.3` | At least 1.2.3, below 1.3.0 for stable releases |
| `~> 1.2` | At least 1.2.0, below 2.0.0 for stable releases |
| `1.2.3` | Exactly 1.2.3; the `=` operator is optional |
| `1.2` | Exactly 1.2.0, not the entire minor series |
| `^1.2.3`, `~1.2.3`, `1.x`, `*` | Rejected |
| `1.2.3 || 2.0.0` | Rejected; no native OR expression |
| `>= 1.2.0 < 2.0.0` | Rejected; use a comma |

Constraint operands use the library's permissive version parser, including leading zeros and extra components. Actual release tags still require complete strict versions. Constraints retain the existing 1024 UTF-16-unit input limit before trimming surrounding whitespace. Internal spacing stays unchanged.

Prerelease behavior is also native to go-version. Each relational comparison must allow a candidate prerelease; `>= 1.2.3-beta.1` accepts `1.2.3-beta.2`, but adding `< 2.0.0` excludes it. Equality and inequality follow the dependency's own semantics. Do not assume an npm expression translates without checking prerelease intent.

The shipping TypeScript sources and npm documentation remain pinned. This document owns the new Go format until configuration migration and eventual cutover update the product documentation. No deployed configuration is rewritten here.

## Evidence

Ten shared fixtures cover exact and compound constraints, surrounding-space trimming, blank input, the input-length boundary, dependency parse-error translation, and a representative exact match/nonmatch. The shared runner checks the pinned TypeScript result separately from the intended Go result. Four cases retain explicit approved differences; the earlier exhaustive dependency-semantics matrix was removed at the user's request.

Direct Go tests cover zero constraints, incomplete release versions, optional v-prefix normalization, typed diagnostic locations, no partial values on failure, parsed-constraint reuse, and Unicode whitespace trimming. HTTP tests verify the serialized success/error boundary and malformed adapter inputs. Version operators, prerelease ordering, and build-metadata semantics remain owned by go-version's tests.

This supplies partial evidence for `formats.versions.01`. Tag selection, ambiguity handling, and actual Git resolution remain pending; no complete capability is marked accepted.

```sh
go test -race ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
go build -o /tmp/code-rules-lab ./cmd/rules-lab
bun tests/migration/compare-rules.ts /tmp/code-rules-lab
bun run check
bun run test:package
go run ./cmd/rules-lab -serve -port 4391
```

The walkthrough reuses Code Rules gallery styling and covers only constraint parsing and matching. It invokes native Go over the local adapter. Earlier operations remain available for regression tests.

## Planned next slices

After this PR: complete project configuration parsing; library manifests and license declarations; release selection from supplied tags; then reading a local library and expanding group selections. These are provisional boundaries within the agreed 20-30 PR plan. Each gets an interactive review and explicit merge approval.

### Local validation

- Go race tests, vet, and Staticcheck v0.8.1 pass on Go 1.27.1.
- All 589 shared comparisons pass for this slice; four constraint cases explicitly differ from TypeScript.
- All eight browser presets return the expected values or errors. Desktop and narrow layouts were inspected; no horizontal overflow or console warnings/errors were observed.
- `bun run check` passes: all 444 TypeScript tests, formatting, lint, typecheck, documentation checks/build, and local links. All six installed-package tests pass with a writable temporary npm cache.
- PR #17 is ready for review against `go-migration`. CodeRabbit acknowledged the manual full review and started processing the branch. Independent review completion and human approval remain separate.
