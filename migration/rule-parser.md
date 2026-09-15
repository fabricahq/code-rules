# Complete rule parsing: fourth Go slice

Status: implemented and locally validated on `codex/go-rule-parser`; ready for review, with the Unicode decision below still required before merge.

PR #13 merged after CodeRabbit reported no actionable findings and all 14 checks passed. This slice starts at `301caa59654e8170f66ef6347eba4c0951d44560`.
The TypeScript reference remains `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`; the engineering corpus remains `e2166f90333157fd3e14c24d3e43287ece858e4b`.
Plan for about 20-30 additional capability-sized PRs across the remaining migration.

## Contract

`rules.Parse(text, path, source string) (Rule, error)` validates the path, splits the document, parses YAML, validates metadata, rejects a blank body, and validates attribution. Failure returns a zero `Rule` and a typed `ValidationError`. Success preserves the original frontmatter and body bytes. The function reads and writes no files.

`Rule` contains a source-qualified ID, group, relative path, title, typed impact, impact description, scalar `whenToRead`, attribution, raw metadata, and body. Tags are validated and retained in raw metadata. Unknown rule fields and attribution fields are rejected, as requested during review. Field names are case-sensitive strings. Multiple unknown fields report the first name in sorted order.

Validation order is: path, document envelope, YAML, forbidden license declarations, unknown frontmatter fields, title, impact, impact description, tags, reading guidance, body, attribution. Text values retain their authored whitespace. Attribution URLs use JavaScript-compatible normalization and reject credentials and non-HTTP(S) schemes.

This supplies rule-input evidence for `builds.resolve`, metadata preservation in `builds.render.05`, and invalid metadata in `authoring.project.05`/`authoring.library.05`. Full capability acceptance remains pending.

## Dependencies

- `go.yaml.in/yaml/v4` v4.0.0-rc.6 supplies YAML syntax parsing and nodes. This is the YAML organization's recommended import path, currently a release candidate. The version is pinned and covered by shared fixtures. Recheck compatibility before upgrading.
- `github.com/nlnwa/whatwg-url` v0.6.2 supplies URL parsing and normalization. Go's standard `net/url` does not match the reference's JavaScript URL behavior.

The YAML node adapter retains the reference's YAML 1.2 core scalar types. For example, `1_000`, `0b10`, and plain dates remain strings. It checks duplicate keys and aliases throughout the document without expanding aliases. Explicit merge defaults remain supported, and merged fields undergo the same unknown-field check. Non-string field names are rejected instead of disappearing during extraction. This structural YAML error precedes field-level checks, including forbidden license declarations; license checks still precede unknown string fields. Raw text is never reserialized.

## Approved behavior and diagnostic differences

The user approved Go-native YAML diagnostics. Both implementations reject malformed YAML with `ValidationError` at the same rule location. Fixtures specify each implementation's exact diagnostic separately. The user also requested a clearer empty-frontmatter diagnostic. Empty, whitespace-only, and comment-only metadata now names the missing YAML frontmatter and lists the required fields. Three fixtures retain the previous reference message separately. Invalid-impact diagnostics also list the six accepted values and the rejected value, as requested during review. Two fixtures retain the earlier reference diagnostics. The user also requested strict frontmatter fields. Unknown fields now fail before field-value validation; attribution entries accept only `url` and `description`. Twenty-two fixtures retain separate reference and Go outcomes for that change. Other domain errors continue to match the pinned reference. No baseline, global comparison policy, or output normalization changes.

## Pending Unicode decision

Valid pairs such as `\uD83D\uDC39` now decode to 🐹, matching the reference. The YAML scanner identifies double-quoted scalars. A parsing copy converts valid pairs to Unicode characters before retrying the decoder. Other scalar styles and escaped backslashes retain literal text. The returned frontmatter is unchanged.

The reference also accepts lone surrogates such as `\uD800`, which are not valid Unicode characters. Go rejects them. That rejection is proposed, following the approved group-metadata policy, but still requires explicit approval for this rule format. The exact reference and proposed results remain in [pending-unicode.json](../tests/migration/rules/pending-unicode.json), outside the passing comparison count.

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

- All 385 shared function cases pass, including 176 complete-rule cases. Sixteen cases use the approved native YAML diagnostic wording; 124 earlier approved differences remain unchanged.
- Go race tests, vet, Staticcheck v0.8.1, and module verification pass on Go 1.27.1.
- A 20-second parser fuzz run completed 794,659 executions without a failure.
- `bun run check` passes: formatting, lint, typecheck, all 444 tests, documentation checks/build, and link checks.
- All six installed-package tests pass.
- All 18 browser examples return their intended values or typed errors. Desktop and narrow layouts have no horizontal overflow.

These are local implementation checks, not approval of the pending Unicode difference or completion of a whole migration capability.

Independent read-only autoreview reported one actionable finding: the pending surrogate-escape acceptance difference. That finding is accepted as a merge blocker. No other actionable findings were reported. Approval and exact shared fixtures are required before this PR can merge.

The focused review of strict-field validation is clean. A proposed change to put license errors ahead of non-string field names was rejected: field-name shape belongs to structural YAML validation. Regression fixtures document both error-ordering cases.

Devin identified rejection of valid surrogate pairs as a compatibility bug. Thirteen shared regression cases now cover paired escapes, four/eight-digit forms, repeated pairs, preceding Unicode, quotes in anchors, tags, scalar styles, and literal backslashes. The paired-escape case is part of the passing comparison suite; only the lone-surrogate decision remains pending.
