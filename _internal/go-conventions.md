# Go conventions

These conventions govern the Go implementation and its development commands.
The first examples live in [rules](../internal/rules/identity.go), [logging](../internal/logging/logging.go), and the [rules lab](../cmd/rules-lab/main.go).

## Error handling

Return ordinary Go values and an `error`. Check the error before using a result.
Domain packages return errors and do not log, print, or exit.

Use a typed error or sentinel when a caller needs to select behavior.
`*rules.ValidationError` identifies invalid input and carries a diagnostic field location.
Use `errors.As` to inspect that type and `errors.Is` for sentinel errors.
Do not branch on message text. Add finer error identities when a concrete caller needs them.

Keep diagnostic context at the layer that knows it:

- The caller supplies the field path, such as `sources.team.groups`.
- The selection parser adds the original element index before sorting.
- The path parser adds the full rule path when group validation fails.

Preserve a deliberate contract error with `%w`:

```go
if err := ValidateGroupID(group, location); err != nil {
    return "", fmt.Errorf("rule path %s: %w", quote(path), err)
}
```

Inspect the wrapped error with `errors.As`, but display the outer `err.Error()` so the operation context survives.
For an implementation failure that callers can only report or abort, add useful context with `%v`.
Translate implementation errors into domain identities only where the meaning is known.
Do not expose a dependency's concrete error type accidentally.

Error context should name the operation and relevant safe identifiers.
Do not include file contents, credentials, or raw request bodies in operational errors.
Validation diagnostics may quote the rejected value for the caller; do not copy those diagnostics into logs.

The lab keeps expected input failures in its JSON response, allowing subsequent requests to run.
Unexpected HTTP failures return a generic 500 response and are logged by the handler.
Unexpected stream or startup failures return to `main`, which logs once and exits 1.
Oversized HTTP bodies receive 413; other body-read failures receive 400 without internal details.
The lab's response envelope and exit codes do not define the future Cobra CLI contract.

Keep exact diagnostic expectations in migration tests, even when callers branch on types.
An intentional text change must retain separate reference and candidate expectations.

## Logging

Use the standard library's [`log/slog`](https://pkg.go.dev/log/slog).
`internal/logging.New` owns handler configuration and returns a `*slog.Logger` without changing global state.
Each command reads its environment at startup, passes stderr to `New`, and injects the logger into code that owns effects.
Pure domain code needs no logger or logging interface.

| Setting | Default | Accepted values |
| --- | --- | --- |
| `CODE_RULES_LOG_LEVEL` | `info` | Slog levels: `debug`, `info`, `warn`, `error`, including standard numeric offsets such as `INFO+2`; case-insensitive |
| `CODE_RULES_LOG_FORMAT` | `text` | `text` or `json`; case-insensitive |

Empty settings select defaults. Invalid settings fail startup with one diagnostic using the default text logger.
Configuration errors name the setting without echoing its value.

```sh
CODE_RULES_LOG_LEVEL=debug CODE_RULES_LOG_FORMAT=json go run ./cmd/rules-lab -serve
```

Logs go to stderr. Stdout is reserved for command results, including the lab's JSON-lines protocol.
Help and usage text are user-facing command output, not operational logs.
The server emits its loopback URL at INFO; a higher threshold suppresses that startup record.
The JSON-lines adapter is quiet on stderr unless an unexpected failure stops it.

Use stable event messages with separate attributes:

```go
logger.Info("rules lab listening", "url", "http://"+listener.Addr().String())
logger.Error("rules lab failed", "error", err)
```

- **DEBUG:** optional investigation details. The lab logs invocation success/failure without inputs or diagnostic text.
- **INFO:** normal lifecycle events, such as the listening address.
- **WARN:** a recoverable operational failure. An undeliverable HTTP response does not stop the server.
- **ERROR:** an unexpected failed operation or process failure that needs attention.

The boundary that finishes handling an error owns its log. Return it or log it, not both.
Expected invalid input belongs in the response and is not an ERROR log.
Do not log request bodies, rule contents, user-supplied diagnostic locations, or secrets, even at DEBUG.
Pass request context to HTTP log calls. The standard HTTP server uses the same handler through `slog.NewLogLogger`.
Only `main` exits the process. Libraries and logging helpers never do.

## Comments and validation

Use a `Package ...` comment for package documentation. Separate file headers from `package` with a blank line.
Add a short header to each authored Go file and document field ownership, nil semantics, and other non-obvious constraints.

The [Go workflow](../.github/workflows/go.yml) checks formatting, `go vet`, pinned Staticcheck defaults, race-enabled tests, and native/reference comparisons.
Staticcheck is a development tool; it adds no dependency to the runtime module.
Run the [slice validation commands](../migration/identities.md#validation) before updating its PR.
Review remains responsible for error ownership, useful context, logging privacy, and comment accuracy.
