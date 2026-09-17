# Native project CLI

`cmd/code-rules` exposes `sync`, `build`, `check`, contextual `--help`, and `--version` through Cobra. `--config` defaults to `.code-rules/config.json`. Duplicate scalar flags and unexpected arguments are usage errors.

Human-readable results are the default. Add the global `--json` flag to any command for one indented JSON response on stdout, including usage errors, operation failures, help, and version. JSON mode never prompts. The TypeScript reference remains unchanged; this is an approved native CLI output change.

- Success: `{"ok":true,"value":{...}}`.
- Failure: `{"ok":false,"error":{"kind":"usage|operation|validation|cancelled|stale_output","message":"..."}}`. Validation errors include `location` when available.
- A stale check also includes `value` with the added, changed, and removed paths. No files were written.
- Help and version use `value.text`. Warnings stay in `value.warnings`.

Human-mode failures go to stderr. JSON-mode failures go to stdout with no duplicate diagnostic on stderr. If writing the response itself fails, stderr carries the output failure and the process exits 1. Exit statuses remain:

- `0`: success or clean check.
- `1`: operation failure, or a successful check that found stale output. A stale check prints its differences on stdout without an error on stderr.
- `2`: invalid command usage.

The entry point cancels on SIGINT/SIGTERM and waits for command cleanup before exiting. Command execution owns no global streams or working-directory changes; embedding callers can provide isolated streams and a path anchor.

## Walkthrough

Build both executables into the same directory:

```sh
go build -o dist/code-rules ./cmd/code-rules
go build -o dist/rules-lab ./cmd/rules-lab
dist/rules-lab -serve -port 4391
```

Open `/walkthrough/pr34`. It runs the compiled `code-rules` executable directly, without a shell. The process report shows the actual arguments, stdout, stderr, and exit status; the file selector shows before/after files. Its outer JSON `ok` only reports whether the lab captured an invocation. Read `exitCode` to determine the command outcome.

Git sources and config paths in the walkthrough stay confined to disposable fixtures. The production command has no such fixture restriction. Setup uses the actual executable's version so a seeded clean check is meaningful.

## Validation and scope

Process tests compile and run the real entry point. They check help/version, unknown commands and flags, duplicate config flags, missing config, stale and clean checks, offline builds, and empty-source sync with no runtime binaries available on PATH. All nine walkthrough scenarios execute the compiled CLI.

Project/library authoring and interactive prompts follow in later slices. The existing TypeScript package entry point remains unchanged until native packaging is reviewed. The browser walkthrough supplies the command and filesystem evidence needed here without requiring another runbook runtime.

Cobra help and version requests take precedence over surplus positional arguments and perform no project operation. Ordinary operational invocations still reject unexpected positional arguments and missing or repeated scalar flags. This intentionally follows Cobra help behavior.
