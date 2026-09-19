---
title: "CLI commands"
description: "Syntax, arguments, options, and results for every supported Code Rules command."
---

Find each command's syntax and options below. Every command also accepts `--json` and `-h` / `--help`; [shared behavior](#shared-options-and-prompts) and [output formats](#command-output) are documented at the end.

## Project commands

In a Git repository, `project init` and `library init` must target the repository root. Running either from a subfolder fails before creating files and shows a command to run from the root. This also applies to the target of `library init --directory`.

Other project and library commands locate the nearest ancestor containing a `.git` directory or file. This supports normal repositories, worktrees, and submodules without invoking Git. Discovery stops at that boundary: it never uses an outer repository's configuration, even if the nearest repository has not been initialized. A nested `.code-rules` directory does not override the repository root.

Project configuration lives in the repository root's `.code-rules/config.json`; custom configuration locations are not supported. Library commands use the repository root's `rule-library.json`. If the required files are missing, initialize that repository from its root.

Outside Git, commands continue to use the current directory (or the explicit library `--directory`). They do not search parent directories for configuration. Relative input paths such as `--body-file`, `--license-file`, and `--notice-file` always resolve from the directory where you ran the command.
Keep replacement files within `.code-rules/local/`.

### init

```sh
code-rules project init [options]
```

Create project configuration, the local rules directory, and the managed project guide. Rerunning init preserves valid configuration and local rules and refreshes an unmodified guide.

| Option | Meaning |
| --- | --- |
| `--non-interactive` | Accepted; init runs without prompts even when this flag is omitted. |

Init does not fetch libraries or generate rule guidance. It refuses to overwrite a manually edited project guide. For file locations and ownership, see [Project and group guides](/reference/files/#project-and-group-guides).

### add source

```sh
code-rules project add library ALIAS [options]
```

Record a library in project configuration without fetching it. `ALIAS` is the source name, such as `team`. Run `code-rules project sync` afterward to import the selected rules.

| Option | Meaning |
| --- | --- |
| `--repository URL` | Required. An accepted HTTPS or SSH [Git repository address](/reference/configuration/#repository-addresses). |
| `--ref REF` | Exact tag, full commit SHA, or version range such as `>= 1.2.0, < 2.0.0`. Saved as `ref` or `version` in configuration as appropriate. |
| `--groups GROUP` | Required. Repeat for multiple group IDs, or supply one selector: `*`, `practices/*`, or `techs/*`. Quote wildcard values so your shell does not expand them. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Source addition preserves existing source exceptions and local files. Pass each group ID as a separate option, rather than a comma-separated flag value:

```sh
code-rules project add library team \
  --repository https://github.com/example/rules.git \
  --ref '>= 1.2.0, < 2.0.0' \
  --groups practices/testing \
  --groups techs/typescript \
  --non-interactive
```

The repository address and groups are illustrative; replace them with a library you can access.

### local add group

```sh
code-rules project add group ID [options]
```

Create a local group with `_group.json` metadata and an authoring README. `ID` is a group path such as `practices/testing` or `techs/typescript`.

| Option | Meaning |
| --- | --- |
| `--name TEXT` | Required. Group display name. |
| `--description TEXT` | Required. What the group covers. |
| `--when-to-read TEXT` | Required. When an agent should read the group. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

A group can exist before it has rules. Local groups do not need a library source declaration. Existing group files are preserved.

### local add rule

```sh
code-rules project add rule ID [options]
```

Create a rule in an existing local group. `ID` includes the group path and rule name, such as `practices/testing/check-retries`, without `.md`.

| Option | Meaning |
| --- | --- |
| `--title TEXT` | Required. Action-oriented rule title. |
| `--when-to-read TEXT` | Required. When an agent should read the rule. |
| `--impact LEVEL` | Required. One of `CRITICAL`, `HIGH`, `MEDIUM-HIGH`, `MEDIUM`, `LOW-MEDIUM`, or `LOW`. |
| `--impact-description TEXT` | Required. The consequence the rule helps prevent. |
| `--body-file PATH` | Optional UTF-8 Markdown body, without frontmatter. Relative paths start at your working directory. Omit to create an unfinished draft. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Create the group first with `local add group`. A missing group fails before metadata prompts. Without `--body-file`, complete the [template](/reference/rule-authoring/#markdown-template) and remove its `code-rules:draft` marker before building. Existing rules are not overwritten.

### sync

```sh
code-rules project sync [options]
```

Fetch the configured library revisions, validate their files, and replace `vendor/` and `generated/` with the complete updated result.

| Option | Meaning |
| --- | --- |

Sync needs access to every configured repository and uses your existing Git credentials. It resolves exact tags again and selects the highest matching version for a version range. If a source fails, the previous complete output is preserved. For update and recovery behavior, see [Sync and recovery](/reference/sync/).

### build

```sh
code-rules project build [options]
```

Regenerate `generated/` from configuration, verified imported files, and local rules. Build works offline and does not change imported revisions.

| Option | Meaning |
| --- | --- |

Run sync first if the imported repository, revision selection, or groups no longer match configuration, or if the stored library files need repair.

### check

```sh
code-rules project check [options]
```

Check generated guidance and the managed project guide without writing files or contacting repositories. Reports stale, missing, or unexpected output and invalid inputs.

| Option | Meaning |
| --- | --- |

Use check in CI to detect files that need regeneration. It uses recorded commits and does not check whether remote tags moved. Success means the managed files agree with their inputs; it does not establish that application code follows the rules.

## Library commands

Library commands work on a shared rule library, independently of a consuming project. They use the current directory unless you pass `--directory PATH`.

### library init

```sh
code-rules library init [options]
```

Create `rule-library.json` and an authoring README without overwriting existing authored files. Optionally copy explicitly supplied license terms into the library.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Library root. Default: your working directory. Library commands do not accept `--config`. |
| `--spdx EXPRESSION` | Library SPDX expression. Must be supplied together with `--license-file`. |
| `--license-file PATH` | UTF-8 license text to copy to `LICENSE.md`. Relative to your working directory, even with `--directory`. |
| `--notice-file PATH` | Optional UTF-8 notice text to copy to `NOTICE.md`. Requires both license options. Relative to your working directory. |
| `--non-interactive` | Accepted; library init runs without prompts even when this flag is omitted. |

If you omit the license options, the manifest leaves the license undeclared. Init does not infer license terms, create a Git repository, commit, or publish the library.

### library add group

```sh
code-rules library add group ID [options]
```

Create a group with `_group.json` metadata and an authoring README in the library. `ID` is a group path such as `practices/testing` or `techs/typescript`.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Library root. Default: your working directory. Library commands do not accept `--config`. |
| `--name TEXT` | Required. Group display name. |
| `--description TEXT` | Required. What the group covers. |
| `--when-to-read TEXT` | Required. When an agent should read the group. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Empty groups are valid. Existing group files are preserved.

### library add rule

```sh
code-rules library add rule ID [options]
```

Create a rule in an existing library group. `ID` includes the group path and rule name, such as `techs/javascript/prefer-for-of`, without `.md`.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Library root. Default: your working directory. Library commands do not accept `--config`. |
| `--title TEXT` | Required. Action-oriented rule title. |
| `--when-to-read TEXT` | Required. When an agent should read the rule. |
| `--impact LEVEL` | Required. One of `CRITICAL`, `HIGH`, `MEDIUM-HIGH`, `MEDIUM`, `LOW-MEDIUM`, or `LOW`. |
| `--impact-description TEXT` | Required. The consequence the rule helps prevent. |
| `--body-file PATH` | Optional UTF-8 Markdown body, without frontmatter. Relative paths start at your working directory. Omit to create an unfinished draft. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Create the group first with `library add group`; rule creation does not create missing groups. Without `--body-file`, complete the draft and remove its `code-rules:draft` marker before validation. Existing rules are not overwritten.

### library check

```sh
code-rules library check [options]
```

Validate the library manifest, all groups and rules, supporting assets, and declared license and notice files. Works offline and does not change files.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Library root. Default: your working directory. Library commands do not accept `--config`. |
| `--non-interactive` | Accepted; library check does not prompt. |

Reports group and rule counts and file-specific errors. Empty groups are valid. Unfinished marked drafts fail. Undeclared licenses produce warnings; invalid declarations and missing declared files fail validation. Library check validates the format, not writing quality or legal permissions. Use the [authoring rubric](/reference/rule-authoring/#authoring-rubric) to review guidance quality.

## Help and version

```sh
code-rules --help
code-rules COMMAND --help
code-rules --version
```

Use `--help` on any command to inspect syntax and options. There is no `help` subcommand.

| Option | Meaning |
| --- | --- |
| `-h`, `--help` | Show help for the selected command. |
| `-v`, `--version` | At the root, print the executable's version. |
| `--json` | Return help or version text in `value.text` inside a JSON response. |

Running `code-rules` or a command group such as `code-rules library` without a subcommand displays help. The `add`, `local`, `local add`, `library`, and `library add` groups organize commands; they do not perform operations themselves. Help never prompts or writes files.

The root `--version` flag prints the tool version. Use `project add library --ref RANGE` to select a library version range.

## Shared options and prompts

Every command accepts `--json`, which prints one JSON response and disables prompts. Every command also accepts `-h` / `--help`.

Authoring commands accept `--non-interactive`. Without it, commands can prompt for missing required metadata or source selections when running in a terminal. With `--non-interactive`, `--json`, or no terminal, missing required inputs cause an error. Positional arguments such as `ID` and `ALIAS` must always be supplied.

Both init commands and both check commands run without prompts. Only `library check` accepts `--non-interactive`; project `check`, `sync`, and `build` do not need or accept it.

String options accept one value and cannot be repeated, except `--groups`, which accepts repeated group IDs. For a value beginning with `-`, use the equals form, such as `--description='-prefixed text'`.

Scaffolding commands validate paths and detect collisions before writing. They preserve existing authored files and roll back failed creation attempts. They do not publish content or convert arbitrary third-party material. For the authoring workflow, see [Set up a project](/start-here/set-up-project/) or [Create a rule library](/guides/create-library/).

## Command output

Human-readable output is the default. Add `--json` for scripts:

```sh
code-rules project check --json
```

JSON mode writes one response to stdout:

| Field | Meaning |
| --- | --- |
| `ok` | `true` for success; `false` for failure or an out-of-date project check. |
| `value` | The command's result, when available. An out-of-date check still includes its report here. |
| `error` | On failure, an object with `kind` and `message`, plus `location` for validation errors when available. |

Project check returns `value.status` as `up_to_date` or `out_of_date`, and a `value.problems` list. Each problem has `kind`, `path`, `message`, and `nextStep`, which contains a suggested repair command. Paths are relative to the configuration directory. Both generated guidance and the managed project guide must be current for success.

Only sync and build report `added`, `changed`, and `removed` file lists. Help and version return their text in `value.text`.

In human mode, operational errors go to stderr. An out-of-date check prints its status, problems, and next steps on stdout. In JSON mode, errors go in the response; stderr is reserved for failures writing that response.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | An operation failed, inputs are invalid, or a project check found stale output. |
| `2` | Invalid command usage, such as an unknown command, unsupported option, or missing required CLI input. |

Usage errors point to the relevant help page. Operational errors identify the affected file or rule when available and describe the problem. For repair commands and interrupted updates, see [Sync and recovery](/reference/sync/).

## Authoring results

Domain failures include `error.code`, such as `needs-init`, `missing-group`, or `guide-edited`. Authoring results include `value.nextSteps` with ordered instructions and copyable commands. Human output shows those next steps after setup, group creation, and rule creation.

Existing groups, rules, and library aliases fail before prompts. Invalid interactive answers repeat the same question while retaining earlier answers. Explicit flags are validated without prompting for replacement values.

## Migrating from earlier commands

Only the `project` and `library` command trees are supported. Earlier command paths now fail with an unknown-command error.
Update existing scripts to use the scoped commands below. Use `-h` or `--help` for help; there is no `help` subcommand.

| Earlier command | Canonical command |
| --- | --- |
| `code-rules init` | `code-rules project init` |
| `code-rules add source` | `code-rules project add library` |
| `code-rules local add group` | `code-rules project add group` |
| `code-rules local add rule` | `code-rules project add rule` |
| `code-rules sync` | `code-rules project sync` |
| `code-rules build` | `code-rules project build` |
| `code-rules check` | `code-rules project check` |

Library commands and the on-disk configuration and directory layout are unchanged.
