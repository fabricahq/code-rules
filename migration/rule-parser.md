# Complete rule parsing: fourth Go slice

Status: implemented and locally validated on `codex/go-rule-parser`; human-approved for merge after the Unicode fix and lone-surrogate decision.

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

## Unicode behavior

Valid pairs such as `\uD83D\uDC39` now decode to 🐹, matching the reference. A preliminary parse masks pairs with equally wide valid escapes to locate double-quoted scalar spans. Only those spans are repaired in a parsing copy, followed by one normal decode. The number of document parses is constant, even with thousands of escaped fields. Other scalar styles and escaped backslashes retain literal text. The returned frontmatter is unchanged.

The reference accepts lone surrogates such as `\uD800`, which are not valid Unicode characters. Go rejects them. After reviewing this remaining difference, the user authorized merging PR #14. The shared suite retains the exact reference value and the approved Go error separately. Valid pairs continue to match the reference. No pending Unicode cases remain.

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

- All 390 shared function cases pass, including 181 complete-rule cases. Sixteen cases use the approved native YAML diagnostic wording; 124 earlier approved differences remain unchanged.
- Go race tests, vet, Staticcheck v0.8.1, and module verification pass on Go 1.27.1.
- A 20-second parser fuzz run completed 794,659 executions without a failure.
- `bun run check` passes: formatting, lint, typecheck, all 444 tests, documentation checks/build, and link checks.
- All six installed-package tests pass.
- All 18 browser examples return their intended values or typed errors. Desktop and narrow layouts have no horizontal overflow.

These are implementation checks, not completion of a whole migration capability.

Independent review identified the surrogate-escape mismatch. Valid pairs are fixed, with a clean focused autoreview. The user approved rejecting malformed lone surrogates before merge.

The focused review of strict-field validation is clean. A proposed change to put license errors ahead of non-string field names was rejected: field-name shape belongs to structural YAML validation. Regression fixtures document both error-ordering cases.

Devin identified rejection of valid surrogate pairs as a compatibility bug. Thirteen shared regression cases now cover paired escapes, four/eight-digit forms, repeated pairs, preceding Unicode, quotes in anchors, tags, scalar styles, and literal backslashes. The paired-escape case is part of the passing comparison suite; the approved lone-surrogate case is also in the passing comparison suite.

### Review follow-up: parsing cost

Devin identified repeated full-document decoding for separate quoted escape pairs. The new location pass replaces that retry loop. A 2,000-tag native test preserves authored metadata, and a benchmark measures 500/1,000/2,000 tags without timing assertions. Shared fixtures also cover verbatim tags, anchor comments, CRLF, and a leading YAML BOM.

The supported line-ending examples use LF and CRLF. A supplementary probe found a pre-existing library difference for CR-only YAML: Go treats lone CR as a line break, while the pinned parser can reject a mapping using it. That case is not counted as parity or an approved difference; full YAML compatibility remains an open acceptance requirement. No baseline or existing fixture was changed to conceal it.
