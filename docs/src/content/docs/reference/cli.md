---
title: "CLI commands"
description: "Choose a Code Rules command and find its current options."
---

Code Rules has two command trees: `project` manages the rules a codebase uses, and `library` manages rules published for other projects. For current arguments and flags, run a command with `--help`.

## Project commands

| Command | What it does |
| --- | --- |
| `code-rules project init` | Create `.code-rules/config.yaml`, `local/`, and the managed project guide. It does not fetch libraries or generate agent guidance. |
| `code-rules project add library ALIAS` | Record a Git library, revision choice, and selected groups in configuration. Run `project sync` afterward to fetch its rules. |
| `code-rules project add group GROUP_PATH` | Create a local group, such as `practices/testing`, with metadata and an authoring guide. |
| `code-rules project add rule RULE_PATH` | Create a rule in an existing local group. The path omits `.md`, for example `practices/testing/check-retries`. Without `--body-file`, it creates a draft you must finish before building. |
| `code-rules project sync` | Fetch every configured library revision, then regenerate `vendor/` and `generated/`. Use it after changing a source, revision, or selected groups. |
| `code-rules project build` | Regenerate `generated/` from local rules and verified vendor snapshots without contacting Git. Use it after changing local rules or exceptions. |
| `code-rules project check` | Report stale or invalid inputs and output without writing files or contacting Git. Use it in CI after installing the intended CLI version. |

`project init` must run from a Git repository root. The other project commands also work from its subdirectories. For a project outside Git, run all project commands from its root. See [Working directories](#working-directories).

For the full add-library workflow, follow [Import rules](/guides/select-rules/). To write a local rule, follow [Write a rule](/guides/write-rules/). For sync, build, and repair steps, see [Sync and recovery](/reference/sync/).

## Library commands

| Command | What it does |
| --- | --- |
| `code-rules library init` | Create `rule-library.yaml` and an authoring README. You can supply license text with `--spdx` and `--license-file`; init does not infer or publish terms. |
| `code-rules library add group GROUP_PATH` | Create group metadata and an authoring guide in the library. |
| `code-rules library add rule RULE_PATH` | Create a rule in an existing library group. Without `--body-file`, it creates a draft you must finish before checking the library. |
| `code-rules library check` | Validate the manifest, groups, rules, assets, and declared terms without changing files. It checks format, not writing quality or legal permission. |

`library init` must run from the Git repository root. Other library commands work from its subdirectories. Use `--directory PATH` to target another library. Outside Git, run from the library root or supply that directory. For the publishing workflow, see [Create your first library](/start-here/create-library/).

<span id="help-and-version"></span>

## Help, version, and license

These commands show the current options and tool information:

```sh
code-rules --help
code-rules project add library --help
code-rules project add rule --help
code-rules library init --help
code-rules library add rule --help
code-rules --version
code-rules --license
```

Use `-h` or `--help` with any command. Run `code-rules project --help` or `code-rules library --help` to list their subcommands. There is no `help` subcommand. The root `--version` reports the CLI version; `project add library --ref` selects a library revision.

## Working directories

In a Git repository, init must target the nearest repository root. Other commands find the nearest ancestor with a `.git` directory or file, including worktrees and submodules. They stop there even if that repository has no Code Rules configuration. A nested `.code-rules/` does not override the Git root's configuration.

Outside Git, commands use the current directory. Library commands can target another directory with `--directory PATH`. Project commands have no custom configuration path.

Relative `--body-file`, `--license-file`, and `--notice-file` paths start where you run the command, even when you use `--directory`. Paths in project configuration are relative to `.code-rules/`.

## Shared options and prompts

Every command accepts `--json`, `-h`, and `--help`. Authoring commands also accept `--non-interactive`. Without `--non-interactive`, an authoring command can prompt for missing inputs in a terminal. With `--json`, `--non-interactive`, or no terminal, the command errors if inputs are missing. Positional paths and library aliases are always required.

To select several groups with `project add library`, repeat `--groups` once for each path. Quote a wildcard selector such as `--groups '*'` so the shell does not expand it. The CLI's `--ref` option accepts an exact tag, a full commit SHA, or a version range. It writes exact choices under `ref` and ranges under `version` in configuration. See [Configuration](/reference/configuration/) for the stored fields.

## Command output

Human-readable output is the default. For scripts, add `--json`:

```sh
code-rules project check --json
```

JSON mode writes one response to stdout with `ok`, a `value` when a result is available, and an `error` on failure. An out-of-date check returns `ok: false` with `value.status: "out_of_date"` and a `value.problems` list. Invalid configuration or unreadable inputs instead return an error without that report. Sync and build results include `added`, `changed`, and `removed` file paths.

In human mode, operational errors go to stderr. A stale check prints its problems on stdout. Check confirms that managed files match their inputs; it does not check whether application code follows the rules. For output details and repair commands, see [Sync and recovery](/reference/sync/).

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | Operation failed, input is invalid, or project output is stale. |
| `2` | Invalid command usage, such as an unknown command, unsupported flag, or missing required CLI argument. |

## Migrating from earlier commands

Older unscoped commands such as `code-rules sync` and `code-rules add source` are unsupported. Use `code-rules project sync` and `code-rules project add library`. Current project configuration lives in `.code-rules/config.yaml`; library metadata uses YAML files.

For the implementation and tests behind these command contracts, see [Inspect implementation and tests](/for-agents/#inspect-implementation-and-tests).
