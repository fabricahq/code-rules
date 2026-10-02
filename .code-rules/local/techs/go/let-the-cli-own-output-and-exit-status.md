---
title: Let the CLI own output and exit status
whenToRead: Before planning, writing, changing, or reviewing Go code in this repository's internal packages that reports a result, a warning, or a failure, or code in internal/cli that turns those into human output, --json responses, or exit statuses.
impact: HIGH
impactDescription: Output or exits from domain packages corrupt the single --json response that agents parse, hide warnings from embedders, and skip the cleanup and exit-status mapping the CLI guarantees.
---

## Let the CLI own output and exit status

Domain packages, such as `internal/project`, `internal/library`, and `internal/imports`, return results and errors.
They never write to standard output or standard error, prompt, or exit.
`internal/cli` alone turns their results into human output, the `--json` response, and the exit status, and only `main` exits the process.

### Implementation

- Return what a person or agent needs to see as data: a field of the result, such as `project.FileChanges.Warnings`, or an error.
- Give each failure a stable code that callers can map, such as a `*gitexec.Error` or the package's own error type, and report invalid input as the package's own `ValidationError`.
  `internal/cli` classifies the error into the response's `kind`, `code`, and `location`, recognizing invalid input through the `errs.ValidationError` interface, and chooses the exit status: `1` for a failed operation, `2` for a usage error, and `130` for an interrupt.
- `--json` writes exactly one response to standard output, including for failures, and never prompts.
  Keep human-only text, such as a progress line, in `internal/cli`, which leaves it out of JSON output.
- Report an expected failure, such as invalid configuration, once, as the command's result. Don't also log it.
- Write only to the `Streams` that `cli.Run` received, so tests and embedders can capture everything a command says.

### Rationale

Agents and scripts parse the `--json` response and branch on exit statuses, so those are the command's contract.
Text a domain package prints bypasses that contract: it's missing from the JSON response, can corrupt standard output in JSON mode, and appears even when an embedder passes its own streams.
A domain package that exits skips deferred cleanup, such as releasing the project writer or rolling back a transaction, and the CLI's exit-status mapping.

### Examples

#### Application: Reporting a tolerated problem

**Incorrect (counterexample):**

```go
// In internal/project, sync announces a warning itself.
for _, source := range in.config.Sources {
	for _, warning := range in.imported[source.Name].Warnings {
		fmt.Fprintln(os.Stderr, "Warning:", warning)
	}
}
```

The warning never reaches the `--json` response, and a test of `cli.Run` with its own `Streams` can't see it.

**Correct:**

```go
// In internal/project, install collects warnings into the report it returns.
warnings = append(warnings, item.Warnings...)
// ...
changes.Warnings = warnings
return changes, nil
```

`internal/cli` prints each warning as `Warning: ...` in human output, and the `--json` response includes them in `value.warnings`.

#### Application: Failing an operation

**Incorrect (counterexample):**

```go
dir, err := os.MkdirTemp("", "code-rules-git-*")
if err != nil {
	log.Fatalf("cannot create temporary Git storage: %v", err)
}
```

The process exits without running deferred cleanup, writes no JSON response, and exits with a status the CLI didn't choose.

**Correct:**

```go
dir, err := os.MkdirTemp("", "code-rules-git-*")
if err != nil {
	return nil, fail("temporary-storage", "Cannot create temporary Git storage.", err)
}
```

The error carries a stable code, and `internal/cli` reports it in the requested output mode with exit status `1`.

### Validation

Search non-test Go files under `internal/` for `os.Exit`, `os.Stdout`, `os.Stderr`, `os.Stdin`, `fmt.Print`, and the `log` package; there should be no matches.
For a new result field or failure, check that a test through `cli.Run` covers both the human output and the `--json` response.

Not a violation: `internal/cli` writing to the `Streams` it was given, or a separate `main` package, such as `cmd/package-binaries`, printing its own output and exiting.
