# Code Rules

This guide explains how to manage this project's [Fabrica Code Rules](https://code-rules.fabricahq.com) configuration and the engineering rules that guide agent work.

## Core concepts

- **Rule:** One engineering practice expressed in a Markdown file. It tells agents what to do, when it applies, and which exceptions to respect.
- **Group:** Related rules for a technology, such as `techs/go`, or a practice, such as `practices/testing`. Its description and reading cue help agents select relevant rules.
- **Library:** A versioned collection of groups in a Git repository. Projects import selected groups from libraries and can also define their own local groups and rules.

## Instructions for agents

### Managing rules

Use this guide to **manage rules**. Use `.code-rules/generated/RULES.md` to **read and apply adopted rules** when working on the project.

1. Identify the requested change: add local guidance, adopt a library, change an existing definition, or verify generated output.
2. Run the corresponding commands below. Run `code-rules project init` **from the Git repository root**. Other project commands can run from any subdirectory and use the root's `.code-rules/config.yaml`. Outside Git, run commands from the project root, the parent of `.code-rules/`.
3. Supply the project's intended metadata and guidance. The examples below illustrate command syntax; replace their values before using them in a real project.
4. Build after local edits. Sync after changing a library's repository, ref, version constraint, or group selection. Inspect the resulting diff and resolve errors before reporting completion.
5. Run check. Exit 0 confirms that generated files match their inputs and this guide matches the installed CLI; it does not verify application code against the rules.

Human-readable output is the default. Add `--json` to any command for a structured response. JSON mode never prompts: supply all required flags. Inspect `ok`, `value`, and `error` and the process exit status. Exit 1 means operation failure or stale output; exit 2 means invalid usage. Use `--help` on a command for all options.

#### Add a group

Choose a technology group path such as `techs/go`, or a practice group path such as `practices/testing`. Use `description` to describe its scope and `when-to-read` to tell agents when to open it. Create a group once, before adding its rules.

```sh
code-rules project add group techs/go \
  --name 'Go' \
  --description 'Go conventions for this project.' \
  --when-to-read 'When writing or reviewing Go code.'
```

Read `.code-rules/local/techs/go/README.md` for group authoring instructions. Confirm that `.code-rules/local/techs/go/_group.json` contains the intended metadata. Local group metadata takes precedence over imported metadata for the same group.

#### Add a rule

Choose a rule path within an existing group. If the group is missing, create it first with `code-rules project add group`; rule creation returns an error without creating the group. Supply a complete Markdown body and discovery metadata. This example creates a new body file without overwriting an existing one:

```sh
set -C
cat > return-errors.body.md <<'RULE_BODY'
# Return errors to the caller

Return a descriptive error when an operation fails. Let the caller decide whether to retry, report, or stop.
RULE_BODY
code-rules project add rule techs/go/return-errors \
  --title 'Return errors to the caller' \
  --impact HIGH \
  --impact-description 'Keep failures visible so callers can respond.' \
  --when-to-read 'When calling fallible operations.' \
  --body-file return-errors.body.md
code-rules project build
code-rules project check
```

Confirm that `.code-rules/local/techs/go/return-errors.md` contains the complete rule and that the generated Go group includes it. The body file is an authoring input; future edits belong in the local rule file. If you omit `--body-file`, open the created Markdown file in your editor. Keep its metadata between the `---` lines; below it, write the instructions, rationale, correct and incorrect examples, and validation steps. Replace template placeholders and remove unused sections before building.

Follow the [rule authoring rubric](https://github.com/fabricahq/code-rules/blob/main/docs/src/content/docs/reference/rule-authoring.md). Use explicit exclusions or replacements in the configuration when overriding imported rules; adding a local rule does not automatically replace an imported rule.

#### Add a third-party library

Obtain the publisher's Git repository address, a released tag or full commit, and the group paths you intend to adopt. Inspect the library's guidance and license terms before adopting it. Replace this illustrative repository and selection:

```sh
code-rules project add library team \
  --repository 'https://github.com/example/engineering-rules' \
  --ref v1.0.0 \
  --groups techs/go
code-rules project sync
code-rules project check
```

`code-rules project add library` records the declaration without fetching. `code-rules project sync` fetches configured sources, validates them, and installs vendor and generated files together. If any source fails, the previous complete output stays in place. Inspect the selected revision and retained terms in `.code-rules/generated/provenance.json` and `.code-rules/vendor/team/`.

Use `--ref` for an exact tag, full commit SHA, or version constraint. For a version constraint, replace `--ref v1.0.0` with `--ref '>= 1.0.0, < 2.0.0'`. Repeat `--groups` for multiple groups. For a wildcard, quote it: `--groups 'techs/*'` or `--groups '*'`.

#### Maintain and verify the project

- After editing local rules, run `code-rules project build`, then `code-rules project check`. In Git repositories, both work from any subdirectory.
- After changing source selection, run `code-rules project sync`, then `code-rules project check`. Sync resolves tags again, so inspect revision changes before committing them.
- Review configuration, local rules, vendor snapshots, and generated output together. Keep project-specific notes in a separate file.
- Use `code-rules library --help` when authoring a separately published library. Project-local authoring and publisher authoring use different command trees.

#### Keep this guide current

Code Rules owns this guide. After upgrading the CLI, refresh it from the project root:

```sh
code-rules project build
code-rules project check
```

`code-rules project build` and `code-rules project sync` refresh this guide alongside generated guidance. `code-rules project init` also refreshes it when creating or checking project setup. These commands preserve valid configuration and local definitions. If this guide has manual edits or an unrecognized format, they stop and ask you to move those notes to a separate file before regenerating it.

Include `code-rules project check` in CI after installing the pinned CLI version. It verifies generated guidance and this guide without writing files. It fails if generated output is stale or this guide is missing or differs from the guide shipped with that version. Run `code-rules project build` to refresh both generated output and this guide, or `code-rules project init` to refresh only this guide and setup files. Code Rules' own tests execute the command examples above against the real CLI on every release change.
