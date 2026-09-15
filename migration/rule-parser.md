# Complete rule parsing: fourth Go slice

Status: implemented and locally validated on `codex/go-rule-parser`; draft review pending the Unicode decision below.

PR #13 merged after CodeRabbit reported no actionable findings and all 14 checks passed. This slice starts at `301caa59654e8170f66ef6347eba4c0951d44560`.
The TypeScript reference remains `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`; the engineering corpus remains `e2166f90333157fd3e14c24d3e43287ece858e4b`.
Plan for about 20-30 additional capability-sized PRs across the remaining migration.

## Contract

`rules.Parse(text, path, source string) (Rule, error)` validates the path, splits the document, parses YAML, validates metadata, rejects a blank body, and validates attribution. Failure returns a zero `Rule` and a typed `ValidationError`. Success preserves the original frontmatter and body bytes. The function reads and writes no files.

`Rule` contains a source-qualified ID, group, relative path, title, typed impact, impact description, scalar `whenToRead`, attribution, raw metadata, and body. Tags are validated and retained in raw metadata. Unknown rule fields also remain in raw metadata, matching the reference. Group JSON remains strict.

Validation order matches the reference: path, document envelope, YAML, forbidden license declarations, title, impact, impact description, tags, reading guidance, body, attribution. Text values retain their authored whitespace. Attribution URLs use JavaScript-compatible normalization and reject credentials and non-HTTP(S) schemes.

This supplies rule-input evidence for `builds.resolve`, metadata preservation in `builds.render.05`, and invalid metadata in `authoring.project.05`/`authoring.library.05`. Full capability acceptance remains pending.

## Dependencies

- `go.yaml.in/yaml/v4` v4.0.0-rc.6 supplies YAML syntax parsing and nodes. This is the YAML organization's recommended import path, currently a release candidate. The version is pinned and covered by shared fixtures. Recheck compatibility before upgrading.
- `github.com/nlnwa/whatwg-url` v0.6.2 supplies URL parsing and normalization. Go's standard `net/url` does not match the reference's JavaScript URL behavior.

The YAML node adapter retains the reference's YAML 1.2 core scalar types. For example, `1_000`, `0b10`, and plain dates remain strings. It checks duplicate keys and aliases throughout extension fields without expanding aliases. Explicit collection tags and merge defaults retain their reference behavior. Raw text is never reserialized.

## Approved diagnostic difference

The user approved Go-native YAML diagnostics. Both implementations reject malformed YAML with `ValidationError` at the same rule location. Fixtures specify each implementation's exact diagnostic separately. Domain errors continue to match the pinned reference. No baseline, global comparison policy, or output normalization changes.

## Pending Unicode decision

The reference accepts UTF-16 surrogate escapes in YAML. Go rejects both lone surrogates such as `\uD800` and paired escapes such as `\uD83D\uDC39`. Literal Unicode and the YAML scalar escape `\U0001F439` work. Rejection is proposed, following the approved group-metadata policy, but requires explicit approval for this rule format. This acceptance difference is not yet counted as approved. Exact reference and proposed results are retained in [pending-unicode.json](../tests/migration/rules/pending-unicode.json), separate from approved comparison fixtures. The file records a proposal and does not participate in the passing parity count.

## Review and verification

Shared fixtures retain all 209 earlier cases and add complete-rule expectations for fields, types, aliases, duplicate keys, tags, attribution, URL normalization, Unicode, whitespace, and line endings. Both sides compare with independent expected values. Syntax-error fixtures separately record approved diagnostic differences.

The browser walkthrough shows only `rules.Parse`. Every example invokes the native function through the development adapter. No discovery, wildcard expansion, Markdown rendering, or filesystem changes happen here.

Run:

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

Open one PR into `go-migration` and stop for human review. The next recommended slice is configuration parsing.

### Local evidence

- All 354 shared function cases pass, including 145 complete-rule cases. Sixteen cases use the approved native YAML diagnostic wording; 124 earlier approved differences remain unchanged.
- Go race tests, vet, Staticcheck v0.8.1, and module verification pass on Go 1.27.1.
- A 20-second parser fuzz run completed 794,659 executions without a failure.
- `bun run check` passes: formatting, lint, typecheck, all 444 tests, documentation checks/build, and link checks.
- All six installed-package tests pass.
- All 18 browser examples return their intended values or typed errors. Desktop and narrow layouts have no horizontal overflow.

These are local implementation checks, not approval of the pending Unicode difference or completion of a whole migration capability.

Independent read-only autoreview reported one actionable finding: the pending surrogate-escape acceptance difference. That finding is accepted as a merge blocker. No other actionable findings were reported. Approval and exact shared fixtures are required before this draft becomes ready.
