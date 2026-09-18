# Repository addresses: fifth Go slice

Status: merged in PR #15 at `ef88c4c6e7d7061da428a8c5783e84e7b3c6906e`. Its walkthrough covered only this slice.

Base: `39c327f103a7eab19468d815d5250acb5f7f5de7`, the approved PR #14 merge into `go-migration`.
TypeScript reference: `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`.
Engineering corpus: `e2166f90333157fd3e14c24d3e43287ece858e4b`.

## Scope and calls

`rules.ParseRepository(input json.RawMessage, location string) (Repository, error)` accepts a JSON string containing an explicit HTTPS, SSH, or SCP-style address.
Raw JSON preserves malformed Unicode escapes until validation, consistent with the other configuration-field parsers.
Failures return a zero `Repository` and a typed `ValidationError`. Diagnostics do not echo the address or credentials.

```go
repository, err := rules.ParseRepository(
    json.RawMessage(`"git@github.com:Team/Rules.git"`),
    "sources.team.repository",
)
if err != nil {
    return err
}
// repository.Identity is "github.com/team/rules".
link, known := repository.FileURL("abc123", "techs/go/errors.md", false)
// known is true; link is https://github.com/Team/Rules/blob/abc123/techs/go/errors.md.
```

`Repository.Web` is nil when the host or transport has no recognized web convention.
`FileURL` returns an empty string and false for those repositories. The lab renders that absence as JSON null.
Callers supply a resolved commit and a validated repository-relative path; link construction does not validate or resolve either input.

The parser preserves the pinned reference's validation order, transport distinctions, case rules, and URL escaping.
It validates authored segments before URL normalization can erase dot segments or encoded separators.
GitHub identities use lowercase paths; GitLab preserves path case.
Unknown hosts retain transport, port, username, and path distinctions.
The caller's transport address remains unchanged.

No new runtime dependencies. The existing `github.com/nlnwa/whatwg-url` v0.6.2 dependency supplies JavaScript-compatible URL parsing.
No fetches, filesystem operations, version-range parsing, configuration assembly, or CLI commands belong to this slice.

## Contract evidence

Repository parsing supplies prerequisite evidence for:

- `formats.identities.04`: repository path safety and case handling.
- `imports.git.05`: rejecting embedded credentials, without claiming Git isolation or fetch safety.
- `builds.render.03`: encoded repository links, without claiming complete Markdown rendering.

Full capability acceptance remains pending. The reference and global comparison policy remain unchanged.
There are no proposed behavior differences in this slice.
PR #14's documented CR-only YAML compatibility gap remains open and separate from repository parsing.

## Validation and walkthrough

The 101 new shared fixtures specify independent values or exact errors for both implementations.
They cover known and custom hosts, transport aliases, ports, IPv6, IDNA, case, escaping, credentials, path traversal, malformed Unicode, and file/raw links.
All 390 earlier cases remain in the same comparison suite.
Direct Go tests also check zero results, typed errors, unchanged input, and malformed JSON.
HTTP tests verify null links, error boundaries, and credential privacy in logs.

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

The browser uses the existing Code Rules gallery styling and invokes actual Go functions.
Try the three GitHub transports, an unsafe path, an unknown host, and encoded file/raw links.
Earlier operations remain available to regression tests but do not appear in this walkthrough.

[Feedback](feedback.md#rules-used) lists the relevant rules. [Go conventions](../_internal/go-conventions.md) govern errors and logging.
Every new function has a concise comment, including helpers and test callbacks.

The next slice handles [exact refs and strict version tags](refs.md). Configuration parsing still requires compatible npm version-range semantics.
Human review and merge separate migration iterations.

### Local evidence

- All 491 shared comparisons pass: 101 new repository cases and all 390 earlier cases. Earlier approved differences remain unchanged.
- Go race tests, vet, and Staticcheck v0.8.1 pass on Go 1.27.1.
- A 15-second fuzz run completed 993,894 executions without a failure.
- Autoreview found no actionable issues in the repository parser, adapter, or walkthrough.
- All 20 browser examples return their expected values or errors. Desktop and narrow layouts have no horizontal overflow or browser console errors.
- `bun run check` passes: formatting, lint, typecheck, all 444 tests, documentation build, and link checks.
- All six installed-package tests pass.

### Review follow-up

CodeRabbit identified a duplicate fixture ID and a test branch that assumed file-link fixtures always succeed. The comparison runner now checks ID uniqueness. An invalid-address link fixture reproduces the test-helper failure and checks exact typed errors after the fix.

Devin also identified a missing required repository field at the lab boundary. Missing fields now return AdapterError; explicit null remains a domain ValidationError. The file-link adapter retains the pinned reference location `repository`, while standalone parsing accepts a caller-supplied location. HTTP tests cover both boundaries.
