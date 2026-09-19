---
title: "CLI commands"
description: "Project and library commands, output, and error behavior."
---

The Go CLI implements project setup, local and library authoring, source addition, sync, build, and check. [Build or install a candidate executable](/guides/install/) to use it. Public release publication remains separate work. The conflict-review prompt and tool-update commands are not implemented.
See [Sync and recovery](/reference/sync/) for filesystem behavior.

For `sync`, `build`, and `check`, run from the consuming project's root by default.
Use `--config` to identify a configuration file elsewhere.

Use `code-rules --version` to print the installed version.
Run `code-rules --help` for a command overview, then narrow the help to the operation you need:

```sh
code-rules library --help
code-rules library add rule --help
code-rules check --help
```

Each command's help lists its accepted options, defaults, examples, and relevant behavior.
Bare command groups such as `code-rules library` also show navigation.
Help never prompts or writes files. Invalid commands and options exit with status 2 and point to the relevant help page.
Authoring help distinguishes fields prompted on a terminal from optional flags; both `init` commands currently run without prompts.

## Command output

The CLI uses human-readable output by default. Add `--json` to any command when a script needs structured results:

```sh
code-rules init
code-rules check --json
```

JSON mode prints one response on stdout and never prompts. Success returns `ok: true` with a `value`. Failure returns `ok: false` with an `error` containing `kind` and `message`, plus `location` for validation errors when available. A native project check returns `value.status` (`up_to_date` or `out_of_date`) and a `value.problems` list. Each problem identifies its `kind`, `path`, `message`, and repair command in `nextStep`. Problem paths are relative to the configuration directory. Both generated guidance and the project guide must be current for `ok: true`. Only build and sync report `added`, `changed`, and `removed` files. Help and version return their text in `value.text`.

Exit status is 0 for success, 1 for operation failure or stale output, and 2 for invalid usage. In human mode, operational errors go to stderr; an out-of-date check prints its status, problems, and next steps on stdout. In JSON mode, errors go in the JSON response; stderr is reserved for failures writing that response.

## Project agent guide

`code-rules init` creates `.code-rules/README.md` alongside configuration and local rules. This guide defines rules, groups, and libraries and gives agents commands for adding groups, adding rules, adopting libraries, building, and checking results. If a custom configuration lives outside a directory named `.code-rules`, init creates `CODE_RULES.md` beside that configuration and preserves the project's `README.md`.

After upgrading, run `code-rules init` again to refresh an older generated guide. It preserves valid configuration and local rules. If someone edited the managed guide, init refuses to overwrite it and explains how to preserve those notes separately.

Run `code-rules check` in CI to verify both generated guidance and the project guide without changing files. A missing or outdated guide fails the check; run `code-rules init` to refresh it. Rebuild stale generated guidance with `code-rules build`. Use `--config` for a custom configuration location. The guide is embedded in each native binary, and Code Rules CI executes its shell examples against the real CLI.

## Sync

```sh
code-rules sync
```

Resolve each source's exact ref or highest matching version tag to a full commit SHA, validate all libraries, and generate resolved rules together.
Record the requested ref or version constraint, selected version tag when applicable, and resolved commit in the vendored provenance.
Each sync resolves tags again. Its file report identifies changed source records and provenance, where you can inspect changed commit targets.
Use the caller's existing Git credentials for private repositories.
Accept the explicit [Git addresses](/reference/configuration/#repository-addresses) in configuration, independently of the hosting provider.
Compare updates against the previous files before applying the replacement.

Sync requires access to all configured source repositories.
If any source fails, preserve the previous complete output.

## Build

```sh
code-rules build
```

Generate resolved rules from verified vendor content, local rules, and configuration.
Build works offline and does not change imported revisions.

If any source repository, requested revision selection, or imported groups differ from the vendor snapshot, sync before building.

## Check

```sh
code-rules check
```

Render expected output without modifying files.
Fail when committed output is stale, missing, or unexpected, or when inputs are invalid.
Check works offline and is intended for CI.
It uses recorded resolved commits and does not check whether remote tags have moved.

A successful check means the rule files agree with their inputs.
It does not mean application code follows those rules.

## Conflict review prompt

`code-rules conflicts --prompt` is proposed and is not accepted by the current CLI. Use the manual [conflict-review prompt](/guides/conflicting-guidance/) with an agent today. The agent reviews guidance; the CLI does not assess application compliance or resolve contradictory policies.

## Update the tool

`code-rules update` is not implemented. Replace the executable using the [installation and upgrade procedure](/guides/install/#upgrade-or-roll-back), then refresh the managed project guide with `code-rules init`, regenerate with `code-rules build`, and run `code-rules check`. Use `sync` separately to select library revisions again.

## Explicit configuration

```sh
code-rules check --config path/to/code-rules/config.json
```

Resolve local paths relative to the chosen configuration directory.
Keep replacement files within that directory's `local/` tree.

## Initialize and author a project

Run these commands from your consuming project with `code-rules` installed:

```sh
code-rules init
code-rules local add group practices/testing
code-rules local add rule practices/testing/retry-budget
code-rules add source team --repository https://github.com/example/rules.git --version '>= 1.2.0, < 2.0.0' --groups '*'
```

Use `--config` to select a configuration file outside the default location.
`init` creates an empty source configuration and `local/README.md` under `.code-rules/` by default, preserving existing files.
Create the group before adding a rule. Rule creation fails before prompting for metadata if the group does not exist. No source declaration is needed for local rules.
`add source` validates and records a library declaration; run `sync` separately to fetch it. It preserves existing source exceptions and local files.

See [Set up a project](/guides/set-up-project/) for all explicit flags, the draft completion workflow, and file ownership.
Authoring commands accept `--non-interactive`. Missing inputs fail without prompting when no terminal is available.

## Author a library

These commands run from the library root, independently of a consuming project's configuration.
Use `--directory path` for a different library root. They do not accept `--config`. Follow [Create a rule library](/guides/create-library/) for the complete workflow.

```sh
code-rules library init
code-rules library add group techs/javascript
code-rules library add rule techs/javascript/prefer-for-of
code-rules library check
```

- `library init` creates the format manifest and a README pointing to the canonical authoring guidance. The README defines rules, groups, and libraries and gives agents executable examples for authoring and validation. Existing READMEs are preserved. Supply `--spdx expression --license-file path` and optional `--notice-file path` to copy explicit terms to `LICENSE.md` and `NOTICE.md`. Authors may defer that choice; the manifest then leaves it undeclared.
- `library add group <group-id>` creates `_group.json` under `techs/` or `practices/`, collecting the name, description, and group-level `whenToRead` cues. A group can exist before it has rules.
- `library add rule <rule-id>` creates a Markdown draft from the canonical template in an existing group. If the group is missing, the Go CLI errors with the command to create it first. It does not offer automatic group creation. Required fields need author input; the command does not invent policy.
- `library check` validates the manifest, all group and rule definitions, and declared license and notice assets offline without writing files. It reports file-specific errors and group and rule counts. Marked drafts fail until completed and their `code-rules:draft` marker is removed. Empty groups are valid. Undeclared licenses produce warnings; malformed declarations and missing declared files fail validation.

Scaffolding commands validate paths and detect collisions before writing. They never overwrite existing files or leave partial scaffolds after a failed operation.
They do not create Git repositories, commit, publish, infer licenses, or convert arbitrary upstream material.
Interactive prompts collect missing author input. Noninteractive use must accept equivalent explicit inputs and fail with actionable errors when required input is missing.
Group and rule commands use the same metadata flags as local authoring; run `code-rules --help` for the complete syntax.

Library validation checks the input format, not the quality of the guidance or legal permissions.
Consumer `code-rules check` instead verifies generated output against adopted inputs.
For non-native material, follow [Adapt a third-party rule](/guides/adapt-rules/).

## Errors

Report the affected file or rule ID and the action needed to resolve the problem.
Examples include a missing override target, an unsupported format version, or a vendor snapshot that needs sync.
Exit nonzero on failure and preserve the previous working output if installation fails.

### Group guides

The CLI creates `README.md` alongside `_group.json` for each new local or library group. The guide directs agents to the current metadata and explains how to add, edit, and validate rules. Group READMEs are authoring documentation and are excluded from rule loading and generated guidance. Existing group files are preserved.
