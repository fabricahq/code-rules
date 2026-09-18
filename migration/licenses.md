# Library license declarations

PR #19 targets `go-migration` after #18 merged. The review stack is authorized; merging remains subject to human approval.

`ReadLibraryLicense(files map[string][]byte, source string)` validates the required `rule-library.json`, format version 1, declared license/notice paths, file presence, and an optional SPDX expression. The Go return type is `(*LicenseDeclaration, error)`: missing licensing produces nil; a declaration produces one object. A missing expression is nil, not an inferred license. Notices retain declaration order, remove duplicates, and exclude the license file. `LicensePaths` returns a deduplicated UTF-16-sorted inventory. Inputs are not mutated.

File presence is separate from contents: empty, CRLF, and binary term files are accepted unchanged. Manifest text must be UTF-8. Paths are contained and never cleaned. Failures return no partial declarations. Unknown license fields are rejected; unknown top-level manifest fields retain the pinned reference's behavior.

SPDX grammar is delegated to [GitHub go-spdx](https://github.com/github/go-spdx), pinned at v2.7.0. The adapter retains case-sensitive identifiers and case-insensitive operators from the reference instead of accepting the dependency's broader normalization shortcuts. It preserves the authored expression. SPDX recognition is syntax validation, not legal advice or a determination that file contents match the declaration. The user approved using the newer identifier and exception lists bundled with go-spdx v2.7.0. Newer recognized identifiers are accepted even when the pinned TypeScript dependency rejects them; this is an explicit migration difference. Shared cases for `MVT-1.1` and `MIT WITH PCRE2-exception` retain both outcomes.

43 independently specified shared cases cover valid compound expressions, WITH exceptions, deprecated identifiers, custom LicenseRef/DocumentRef, case rules, malformed expressions, notices, missing files, and manifest failures. All 661 accumulated comparisons pass, including 202 approved differences. Direct Go tests verify byte preservation and Unicode path order. Existing TypeScript diagnostics encode file and nested field with two colons; the reference adapter recovers that field without modifying the diagnostic message.

Walkthrough: `/walkthrough/pr19`. It supplies manifest text and file names to the real Go function. It reads no user files. Run `go run ./cmd/rules-lab -serve -port 4391`.

This is partial evidence for `formats.licenses`. Actual filesystem loading, output copying, and end-to-end license propagation remain separate. Source and engineering corpus pins are unchanged. Go race tests, vet, and shared fixtures run for this slice; stack-wide checks and independent review remain required before merge.

The adapter validates standard exceptions attached to custom LicenseRef and DocumentRef terms before delegating the remaining grammar to go-spdx. This preserves the reference behavior despite go-spdx v2.7.0 not parsing WITH after custom references. Shared fixtures cover both reference forms and reject a license used as an exception or a plus suffix on custom terms.

Incomplete expressions ending in an opening parenthesis or a DocumentRef colon return ValidationError before reaching go-spdx v2.7.0, which otherwise dereferences a missing token. Regression fixtures cover bare, nested, and trailing-space inputs.

## Singular Go license model

The user approved replacing zero-or-one license arrays with an optional declaration. `ReadLibraryLicense` returns nil or one declaration; loaded catalogs expose `license: null` or `license: {...}`. Multiple notice files still belong to that one declaration. The pinned TypeScript reference remains unchanged, and successful shared fixtures retain its exact array results in `referenceExpected` beside the new Go expectations.

The singular field also applies to resolved sources, effective rules, and Go-generated `provenance.json`. The TypeScript provenance shape remains plural until cutover; this is an approved Go output-shape difference. License/notice contents and generated destinations are unchanged.
