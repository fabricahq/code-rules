# Go conventions

Read these conventions before changing Go code. [CONTRIBUTING.md](../CONTRIBUTING.md#validate-changes) owns validation commands.

## Errors and output

Return values and an `error`; check the error before using a result. Domain code returns errors without printing or exiting. The CLI owns human diagnostics, structured JSON, and exit-status mapping. Only `main` exits the process.

Use typed errors and sentinels when callers need to select behavior. Use `errors.As` for `*authored.ValidationError` and `errors.Is` for sentinels. Branch on error identities rather than message text. Add context where the operation and relevant safe identifiers are known, preserving deliberate contract errors with `%w`.

The caller supplies diagnostic field paths such as `sources.team.groups`; parsers add precise element or file context. Display the outer error so this context survives. Keep secrets and file contents out of operational diagnostics. Never show text that Git, a Git server, or the GitHub CLI printed, in output or errors: match it with `gitexec.Mentions` and report a fixed message of your own. Run Git through `gitexec.Isolated` for other people's repositories and `gitexec.Owned` for the user's own.

Human command output is the default. `--json` produces one stdout response, including failures, and never prompts. Keep these contracts tested through `internal/cli.Run` and real subprocesses. Expected validation failures are command results, not duplicated log events.

## Ownership and effects

Keep parsing and rendering independent of filesystem and network effects. Keep filesystem snapshots, resource limits, cancellation, and write ownership explicit at the effectful operation. Preserve original retained bytes where the format promises preservation.

Collect interactive inputs before acquiring writer ownership. Validate all inputs before installation. Exercise refusal and interrupted/concurrent-write behavior through the production operation, rather than testing only isolated helpers.

## The public package

`libraryformat` is the only package other modules can import, and tools such as catalogs depend on it. Its parsers take bytes or text and never run Git or access files or the network, including through the internal packages they use. Code Rules reads libraries through it, so never parse the same format a second way. Export only what other tools need, and keep helpers that internal parsers share in an internal package, such as `internal/authored`. Give callers error identity only for decisions they make, such as `ErrNotReleaseTag` and `*UnsupportedReleaseRecordError`; the `*authored.ValidationError` its parsers return serves Code Rules' own diagnostics. Its exported API is part of the [release policy's](../.release-planner/policy.md) public contract, and its tests use only that API, except for regression fixtures that check exact diagnostic locations.

## Comments and validation

Use a `Package ...` comment for package documentation. Separate other file headers from `package` with a blank line. Explain result ownership, nil semantics, and non-obvious constraints. Follow the [local comment rule](rules/comment-role-result-and-constraints.md).

The [Go workflow](../.github/workflows/go.yml) checks workflows with actionlint, then formatting, vet, pinned Staticcheck, race-enabled tests, and command builds. Parser regression data lives under each parser's `testdata` directory, such as `libraryformat/testdata`; it does not require a legacy runtime. Review error ownership, actionable context, byte preservation, and cancellation when changing effects.
