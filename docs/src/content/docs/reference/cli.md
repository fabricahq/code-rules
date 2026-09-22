---
title: "CLI commands"
description: "Project and library commands, output, and error behavior."
---

The Go CLI implements project setup, local and library authoring, source addition, sync, build, and check. [Build or install a candidate executable](/guides/install/) to use it. Public release publication remains separate work. The conflict-review prompt and tool-update commands are not implemented.
See [Sync and recovery](/reference/sync/) for filesystem behavior.

## Choose a scope

Use `project` to configure and manage rules for the software project you are working in.
Use `library` to create and maintain a collection of rules shared across projects.
Rules and groups added through `project` belong to that project; rules and groups added through `library` belong to the shared library.
Command menus list `project` before `library` and initialization before authoring. Add menus list `rule`, `group`, then `library` where available.
Project help separates the main commands (`init`, `add`, and `sync`) from utility commands (`build` and `check`).

```text
code-rules project init
code-rules project add library <alias>
code-rules project add group GROUP_PATH
code-rules project add rule RULE_PATH
code-rules project sync
code-rules project build
code-rules project check

code-rules library init
code-rules library add group GROUP_PATH
code-rules library add rule RULE_PATH
code-rules library check
```

`GROUP_PATH` combines a category and group slug, such as `practices/testing`. `RULE_PATH` adds a rule slug, such as `practices/testing/my-rule`, without `.md`. These paths identify rules and groups within a project or library; the group name and rule title are their readable labels.

`project add library` records configuration without fetching anything. Its alias identifies the library under `sources` in configuration.
Use `project sync` to fetch configured libraries and rebuild guidance. Use `project build` to rebuild from library snapshots already on disk.
`project check` verifies file consistency; it does not review application code for compliance with the rules.

Run all project commands from the project root. They always use `.code-rules/config.json`.

Use `code-rules --version` to print the installed version.
Run `code-rules --help` for a command overview, then narrow the help to the operation you need:

```sh
code-rules project --help
code-rules library --help
code-rules library add rule --help
code-rules project check --help
```

Use `-h` or `--help` on any command to see its accepted options, defaults, examples, and relevant behavior.
Bare command groups such as `code-rules library` also show navigation.
Help never prompts or writes files. Invalid commands and options exit with status 2 and point to the relevant help page.
If an authoring command is missing a required alias or path, the error names the missing argument, explains its purpose, and shows the command syntax and an example.
Authoring help distinguishes fields prompted on a terminal from optional flags; both `project init` and `library init` currently run without prompts.
Every command separates its own options from **Common options**. Group and rule creation use **Group options** and **Rule options**; adding a library or initializing one uses **Library options**. Commands with no specific options show only **Common options**. Common options are reused across commands, not necessarily available everywhere: `--help` is local to each command, `--non-interactive` is local to authoring commands, and `--json` is inherited globally. Library commands also accept `--directory`.

## Command output

The CLI uses human-readable output by default. Add `--json` to any command when a script needs structured results:

```sh
code-rules project init
code-rules project check --json
```

JSON mode prints one response on stdout and never prompts. Success returns `ok: true` with a `value`. Failure returns `ok: false` with an `error` containing `kind` and `message`, plus `location` for validation errors when available. Domain failures also include `error.code`, such as `needs-init`, `missing-group`, or `guide-edited`, so agents can distinguish recovery actions without parsing the message. Authoring results include `value.nextSteps`: ordered instructions with copyable commands, shared with human output. The `value.next` text field remains available and contains the same guidance. A native project check returns `value.status` (`up_to_date` or `out_of_date`) and a `value.problems` list. Each problem identifies its `kind`, `path`, `message`, and repair command in `nextStep`. Problem paths are relative to the configuration directory. Both generated guidance and the project guide must be current for `ok: true`. Build and sync report `added`, `changed`, and `removed` files. When they refresh the managed guide, `value.guide` reports its configuration-relative `path` and whether it was `created`. Help and version return their text in `value.text`.

Exit status is 0 for success, 1 for operation failure or stale output, and 2 for invalid usage. In human mode, failures start with a separated `Error:` label. The label is bold red on a supported terminal and plain text when redirected, when `NO_COLOR` is nonempty, or when `TERM` is empty or `dumb`. Operational errors go to stderr; an out-of-date check prints its error label, status, problems, and next steps on stdout. In JSON mode, errors go in the JSON response; stderr is reserved for failures writing that response.

## Project agent guide

`code-rules project init` creates `.code-rules/README.md` alongside configuration and local rules. This guide defines rules, groups, and libraries and gives agents commands for adding groups, adding rules, adopting libraries, building, and checking results. The project's root `README.md` remains untouched.

Both first-time and repeated initialization show the configuration file path.

Build and sync automatically refresh an older generated guide using the template bundled in the selected CLI version. Repeating init also refreshes it. These commands preserve configuration and local rules, and refuse to overwrite manual edits to the guide. Changes to the managed guide format are [breaking changes](/guides/install/#versioning).

Run `code-rules project check` in CI to verify both generated guidance and the project guide without changing files. A missing or outdated guide fails the check. Run `code-rules project build` to refresh the guide and regenerate guidance, or run `code-rules project init` to refresh only the guide and setup files. The guide is embedded in each native binary, and Code Rules CI executes its shell examples against the real CLI.

## Sync

```sh
code-rules project sync
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
code-rules project build
```

Generate resolved rules from verified vendor content, local rules, and configuration.
Build works offline and does not change imported revisions. It also creates a missing managed project guide or refreshes an older, unedited guide. Guide and generated-output updates share one recoverable transaction. A manually edited guide stops the build before it changes output.

If any source repository, requested revision selection, or imported groups differ from the vendor snapshot, sync before building.

## Check

```sh
code-rules project check
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

`code-rules update` is not implemented. Replace the executable using the [installation and upgrade procedure](/guides/install/#upgrade-or-roll-back), then refresh the managed guide and regenerate guidance with `code-rules project build`, and run `code-rules project check`. Use `project sync` separately to select library revisions again.

## Project location

Run project commands from the project root. Configuration lives in `.code-rules/config.json`; custom configuration locations are not supported. Commands do not search parent directories or discover the Git root.
Keep replacement files within `.code-rules/local/`.

## Initialize and author a project

Run these commands from your consuming project with `code-rules` installed:

```sh
code-rules project init
code-rules project add group practices/testing
code-rules project add rule practices/testing/retry-budget
code-rules project add library team --repository https://github.com/example/rules.git --ref '>= 1.2.0, < 2.0.0' --groups '*'
```

`project init` creates an empty source configuration and `local/README.md` under `.code-rules/`, preserving existing files.
Adding an existing group, rule, or library alias fails before any prompts. Edit the existing item or choose a different path or alias.
Create the group before adding a rule. Rule creation fails before prompting for metadata if the group does not exist. No source declaration is needed for local rules.
`project add library` validates and records a library declaration; run `project sync` separately to fetch it. It preserves existing source exceptions and local files.
`--ref` accepts an exact tag, full commit SHA, or version range. Bare versions are literal tags; ranges use operators such as `>=` or `~>`. The CLI saves the value in the appropriate `ref` or `version` configuration field.
Before prompting, it explains the project-local alias, the single ref input, and group selectors with an example. Find group paths in the library documentation or its `practices/` and `techs/` directories at the selected revision. Choose explicit paths or one wildcard. The same guidance is available with `--help`, including an example with all required flags for agents and scripts.

See [Set up a project](/guides/set-up-project/) for all explicit flags, the draft completion workflow, and file ownership.
Before prompting, group and rule commands show the selected path and an example of all the fields together. A group has a readable name, description, and reading cue. A rule has a readable title, reading cue, and impact details; the full instructions belong in its Markdown body. Examples are guidance only and are never saved as your content.

After creating a group, the CLI shows a copyable add-rule command using that group's path and, for library commands, the selected library directory. Replace the example `my-rule` slug with your own.

After creating a rule, the CLI prints its Markdown file path and explains what to edit below the metadata block. Finish the instructions, rationale, correct and incorrect examples, and validation steps; replace template placeholders and remove unused sections. The final commands preserve your library `--directory` selection. Supplying `--body-file` creates a rule with existing text and shows review instructions instead of draft-completion steps.

Authoring commands accept `--non-interactive`. Missing inputs fail without prompting when no terminal is available.

## Author a library

These commands run from the library root, independently of a consuming project's configuration.
Use `--directory path` for a different library root. Follow [Create a rule library](/guides/create-library/) for the complete workflow.

```sh
code-rules library init
code-rules library add group techs/javascript
code-rules library add rule techs/javascript/prefer-for-of
code-rules library check
```

- `library init` creates the format manifest and a README pointing to the canonical authoring guidance. The README defines rules, groups, and libraries and gives agents executable examples for authoring and validation. Existing READMEs are preserved. Supply `--spdx expression --license-file path` and optional `--notice-file path` to copy explicit terms to `LICENSE.md` and `NOTICE.md`. Authors may defer that choice; the manifest then leaves it undeclared.
- `library add group GROUP_PATH` creates `_group.json` under `techs/` or `practices/`, collecting the name, description, and group-level `whenToRead` cues. A group can exist before it has rules.
- `library add rule RULE_PATH` creates a Markdown draft from the canonical template in an existing group. If the group is missing, the Go CLI errors with the command to create it first. It does not offer automatic group creation. Required fields need author input; the command does not invent policy.
- `library check` validates the manifest, all group and rule definitions, and declared license and notice assets offline without writing files. It reports file-specific errors and group and rule counts. Marked drafts fail until completed and their `code-rules:draft` marker is removed. Empty groups are valid. Undeclared licenses produce warnings; malformed declarations and missing declared files fail validation.

Scaffolding commands validate paths and detect collisions before writing. They never overwrite existing files or leave partial scaffolds after a failed operation.
They do not create Git repositories, commit, publish, infer licenses, or convert arbitrary upstream material.
Interactive prompts collect missing author input. A blank required answer or an invalid impact, repository, revision choice, revision value, or group selection shows an error and repeats the same question, keeping earlier answers. Press Ctrl-C to cancel. Explicit flags are validated without replacing them with prompts. Noninteractive use must accept equivalent explicit inputs and fail with actionable errors when required input is missing.
Group and rule commands use the same metadata flags as local authoring; run `code-rules --help` for the complete syntax.

Library validation checks the input format, not the quality of the guidance or legal permissions.
Consumer `code-rules project check` instead verifies generated output against adopted inputs.
For non-native material, follow [Adapt a third-party rule](/guides/adapt-rules/).

## Errors

Report the affected file or rule ID and the action needed to resolve the problem.
Examples include a missing override target, an unsupported format version, or a vendor snapshot that needs sync.
Exit nonzero on failure and preserve the previous working output if installation fails.

### Group guides

The CLI creates `README.md` alongside `_group.json` for each new local or library group. The guide directs agents to the current metadata and explains how to add, edit, and validate rules. Group READMEs are authoring documentation and are excluded from rule loading and generated guidance. Existing group files are preserved.

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
