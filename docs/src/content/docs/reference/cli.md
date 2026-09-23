---
title: "CLI commands"
description: "Syntax, arguments, options, and results for every supported Code Rules command."
---

Find each command's syntax and options below. Every command also accepts `--json` and `-h` / `--help`; [shared behavior](#shared-options-and-prompts) and [output formats](#command-output) are documented at the end.

## Project commands

`code-rules project` manages rules for a **project**, the codebase whose rules Code Rules manages. Initialize from the project root. In Git repositories, other project commands also work from subdirectories.

Configuration lives in `.code-rules/config.json` at the project root. See [Working directories](#working-directories) for repository discovery and input-file paths.

### project init

```sh
code-rules project init [options]
```

Create project configuration, the local rules directory, and the managed Code Rules guide. Rerunning init preserves valid configuration and local rules and refreshes an unmodified guide.

| Option | Meaning |
| --- | --- |
| `--non-interactive` | Accepted; init runs without prompts even when this flag is omitted. |

Init does not fetch libraries or generate rule guidance. It refuses to overwrite a manually edited Code Rules guide. For file locations and ownership, see [Project and group guides](/reference/files/#project-and-group-guides).

### project add library

```sh
code-rules project add library ALIAS [options]
```

Record a library in project configuration without fetching it. `ALIAS` is the source name, such as `team`. Run `code-rules project sync` afterward to import the selected rules.

| Option | Meaning |
| --- | --- |
| `--repository URL` | Required. An accepted HTTPS or SSH [Git repository address](/reference/configuration/#repository-addresses). |
| `--ref REF` | Required. Exact tag name, full Git commit, or version range such as `>= 1.2.0, < 2.0.0`. Uses the [configuration version syntax](/reference/configuration/). |
| `--groups GROUP` | Required. Repeat for multiple group IDs, or supply one selector: `*`, `practices/*`, or `techs/*`. Quote wildcard values so your shell does not expand them. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Bare versions supplied to `--ref`, such as `v1.2.3`, select literal tags. Ranges use operators such as `>=` or `~>`. The CLI records exact selections in the configuration's `ref` field and ranges in its `version` field.

Library addition preserves existing source exceptions and local files. Pass each group ID as a separate option, rather than a comma-separated flag value:

```sh
code-rules project add library team \
  --repository https://github.com/example/rules.git \
  --ref '>= 1.2.0, < 2.0.0' \
  --groups practices/testing \
  --groups techs/typescript \
  --non-interactive
```

The repository address and groups are illustrative; replace them with a library you can access.

### project add group

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

### project add rule

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

Create the group first with `project add group`. A missing group fails before metadata prompts. Without `--body-file`, complete the [template](/reference/rule-authoring/#markdown-template) and remove its `code-rules:draft` marker before building. Existing rules are not overwritten.

### project sync

```sh
code-rules project sync [options]
```

Fetch the configured library revisions, validate their files, and replace `vendor/` and `generated/` in the Code Rules directory with the complete updated result. Sync also refreshes an older, unedited managed Code Rules guide.

Accepts the [shared options](#shared-options-and-prompts) only.

Sync needs access to every configured repository and uses your existing Git credentials. It resolves exact tags again and selects the highest matching version for a version range. If a source fails, the previous complete output is preserved. For update and recovery behavior, see [Sync and recovery](/reference/sync/).

### project build

```sh
code-rules project build [options]
```

Regenerate `generated/` from project configuration, verified imported files, and local rules. Build works offline and does not change imported revisions. It also creates a missing managed Code Rules guide or refreshes an older, unedited guide alongside generated output. A manually edited guide stops the build before it changes output.

Accepts the [shared options](#shared-options-and-prompts) only.

Run sync first if the imported repository, revision selection, or groups no longer match configuration, or if the stored library files need repair.

### project check

```sh
code-rules project check [options]
```

Check generated guidance and the managed Code Rules guide without writing files or contacting repositories. Reports stale, missing, or unexpected output and invalid inputs.

Accepts the [shared options](#shared-options-and-prompts) only.

Use check in CI to detect files that need regeneration. It uses recorded commits and does not check whether remote tags moved. Success means the managed files agree with their inputs; it does not establish that application code follows the rules.

## Library commands

`code-rules library` manages a **library**, an independently maintained collection of rule groups that projects can import. Initialize from the library repository root. Other library commands find that root from subdirectories. Use `--directory PATH` to target another library; see [Working directories](#working-directories).

### library init

```sh
code-rules library init [options]
```

Create `rule-library.json` and an authoring README without overwriting existing authored files. Optionally copy explicitly supplied license terms into the library.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. Init must target the repository root. |
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
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. Init must target the repository root. |
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
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. Init must target the repository root. |
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
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. Init must target the repository root. |
| `--non-interactive` | Accepted; library check does not prompt. |

Reports group and rule counts and file-specific errors. Empty groups are valid. Unfinished marked drafts fail. Undeclared licenses produce warnings; invalid declarations and missing declared files fail validation. Library check validates the format, not writing quality or legal permissions. Use the [authoring rubric](/reference/rule-authoring/#authoring-rubric) to review guidance quality.

<span id="help-and-version"></span>

## Help, version, and license

```sh
code-rules --help
code-rules COMMAND --help
code-rules --version
code-rules --license
```

Use `-h` or `--help` to inspect command syntax and options. For example, run `code-rules project --help` to find project commands or `code-rules library add rule --help` for rule authoring options.

| Option | Meaning |
| --- | --- |
| `-h`, `--help` | Show help for the selected command. |
| `-v`, `--version` | At the root, print the executable's version. |
| `--license` | At the root, print the full embedded MIT license and copyright notice. |
| `--json` | Return help, version, or license text in `value.text` inside a JSON response. |

Running `code-rules` or a command group such as `code-rules library` without a subcommand displays help. The `project`, `project add`, `library`, and `library add` groups organize commands; they do not perform operations themselves. Help never prompts or writes files.

The root `--version` flag prints the tool version. To select a library revision, use `project add library --ref` instead.

## Working directories

In Git repositories, run `code-rules project init` or `code-rules library init` from the repository root. Initialization from a subdirectory fails without writing files. This also applies to the target of `library init --directory`.

Other commands find the nearest ancestor containing a `.git` directory or file, including worktrees and submodules. Project commands use that root's `.code-rules/config.json`; library commands use its `rule-library.json`. They stop at that repository boundary, even if its configuration is missing. A nested `.code-rules/` does not override the root's configuration.

Outside Git, commands use the current directory, or the directory selected by `--directory` for library commands. They do not search parent directories for configuration. Custom project configuration locations are not supported.

Relative input paths such as `--body-file`, `--license-file`, and `--notice-file` resolve from the directory where you run the command. Paths inside project configuration remain relative to `.code-rules/`.

## Shared options and prompts

Every command accepts `--json`, which prints one JSON response and disables prompts. Every command also accepts `-h` / `--help`.

Authoring commands accept `--non-interactive`. Without it, commands can prompt for missing required metadata or source selections when running in a terminal. With `--non-interactive`, `--json`, or no terminal, missing required inputs cause an error. Positional arguments such as `ID` and `ALIAS` must always be supplied.

Invalid interactive answers repeat the same question while retaining earlier answers. Explicit flags are validated without prompting for replacement values. Existing groups, rules, and library aliases fail before prompts.

Both init commands and both check commands run without prompts. Only `library check` accepts `--non-interactive`; project `check`, `sync`, and `build` do not need or accept it.

String options accept one value and cannot be repeated, except `--groups`, which accepts repeated group IDs. For a value beginning with `-`, use the equals form, such as `--description='-prefixed text'`.

Scaffolding commands validate paths and detect collisions before writing. They preserve existing authored files and roll back failed creation attempts. They do not publish content or convert arbitrary third-party material. For the authoring workflow, see [Set up your first project](/start-here/set-up-project/) or [Create your first library](/start-here/create-library/).

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
| `error` | On failure, an object with `kind` and `message`, plus `location` when available. Domain failures include a stable `code`, such as `needs-init`, `missing-group`, or `guide-edited`. |

Project check returns `value.status` as `up_to_date` or `out_of_date`, and a `value.problems` list. Each problem has `kind`, `path`, `message`, and `nextStep`, which contains a suggested repair command. Paths are relative to the Code Rules directory. Both generated guidance and the managed Code Rules guide must be current for success.

Authoring results include `value.nextSteps`, an ordered list of instructions and copyable commands. Human output shows those steps after initialization and rule or group creation.

Only sync and build report `added`, `changed`, and `removed` file lists. When they refresh the managed Code Rules guide, `value.guide` reports its path relative to the Code Rules directory and whether it was `created`. Help, version, and license return their text in `value.text`.

In human mode, operational errors go to stderr. An out-of-date check prints its status, problems, and next steps on stdout. In JSON mode, errors go in the response; stderr is reserved for failures writing that response. Unreleased preview builds also print a non-production warning with their source commit to stderr before every command, including help, version, and JSON commands. JSON output on stdout is unchanged. See [testing PR preview builds](https://github.com/fabricahq/code-rules/blob/main/_engineering/releasing.md).

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | An operation failed, inputs are invalid, or a project check found stale output. |
| `2` | Invalid command usage, such as an unknown command, unsupported option, or missing required CLI input. |

Usage errors point to the relevant help page. Operational errors identify the affected file or rule when available and describe the problem. For repair commands and interrupted updates, see [Sync and recovery](/reference/sync/).

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
