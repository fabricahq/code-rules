---
title: "CLI commands"
description: "Syntax, arguments, options, and results for every supported Code Rules command."
---

Find each command's syntax and options below. Every command also accepts `--json` and `-h` / `--help`; [shared behavior](#shared-options-and-prompts) and [output formats](#command-output) are documented at the end.

## Project commands

`code-rules project` manages rules for a **project**, the codebase whose rules Code Rules manages. Initialize from the project root. In Git repositories, other project commands also work from subdirectories.

Configuration lives in `.code-rules/config.yaml` at the project root. See [Working directories](#working-directories) for repository discovery and input-file paths.

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
| `--groups GROUP` | Groups to import in full. Repeat for multiple group IDs, or supply one selector: `*`, `practices/*`, or `techs/*`. Quote wildcard values so your shell does not expand them. |
| `--rules RULE` | Individual rules to import without the rest of their group, such as `practices/testing/verify-retry-limits`. Repeat for multiple rules. |
| `--ref REF` | Optional and advanced. Import the library exactly as it was at one revision: a tag, such as the library release tag `release/5`, or a full commit SHA. Recorded as the source's `ref`. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Supply at least one `--groups` or `--rules`. Without `--ref`, the source follows each rule's newest version when the project updates. To pin individual rules, add `pins` to the source in configuration, or use `code-rules project update --keep`; see [Choose versions](/reference/configuration/#choose-versions).

Library addition preserves existing source exceptions and local files. Pass each group ID as a separate option, rather than a comma-separated flag value:

```sh
code-rules project add library team \
  --repository https://github.com/example/rules.git \
  --groups practices/testing \
  --groups techs/typescript \
  --non-interactive
```

The repository address and groups are illustrative; replace them with a library you can access.

### project add group

```sh
code-rules project add group ID [options]
```

Create a local group with `_group.yaml` metadata and an authoring README. `ID` is a group path such as `practices/testing` or `techs/typescript`.

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

Create the group first with `code-rules project add group`. A missing group fails before metadata prompts. Without `--body-file`, complete the [template](/reference/rule-authoring/#markdown-template) and remove its `code-rules:draft` marker before building. Existing rules are not overwritten.

#### Fork a library rule

```sh
code-rules project add rule ID --from LIBRARY@VERSION [options]
```

Copy one version of a library rule into `local/` so the project controls its text. Use a fork to change what a rule says. To import one rule without the rest of its group, [select it individually](/reference/configuration/#select-individual-rules) instead, and to keep an imported rule at an older version, [pin it](/reference/configuration/#pin-a-rule); both keep the rule's identity and update reports. `ID` is the library rule ID, such as `practices/testing/verify-retry-limits`, and the fork keeps it: `local/practices/testing/verify-retry-limits.md`.

| Option | Meaning |
| --- | --- |
| `--from LIBRARY@VERSION` | Required for a fork. `LIBRARY` is a configured source name, such as `team`, or a [repository address](/reference/configuration/#repository-addresses). `VERSION` is one of the rule's versions, such as `1.3.0`. |
| `--reason TEXT` | Why the project replaces the imported rule. Required when the project imports this rule from `LIBRARY`. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Metadata options and `--body-file` don't apply to a fork. The command:

1. Finds the library release that published `<VERSION>` from the release records in the library's `release/<number>` tags, and reads the rule and its asset directory at that tag's commit.
2. Writes them under `local/` and creates the local group from the library's group metadata if it doesn't exist.
3. Adds an `attribution` entry that links to the rule at the tag's commit. Libraries hosted outside GitHub.com and GitLab.com get an attribution entry only when the repository address is an HTTPS URL.
4. When the project imports the rule from `LIBRARY`, adds a `replace` entry for it with your reason, so agents read only the fork.

If the rule links to files in the library's shared `assets/` directory, the fork copies them into its own asset directory and updates the links, because local rules can't depend on library files. Existing local rules are never overwritten. A forked rule has no version; it changes only when you edit it. Run `code-rules project build` afterward. The library's license still applies to the copied text; see [License rules](/guides/license-rules/).

### project sync

```sh
code-rules project sync [options]
```

Import the rule versions recorded for each source, validate their files, and replace `vendor/` and `generated/` in the Code Rules directory with the complete result. Sync also refreshes an older, unedited managed Code Rules guide.

Accepts the [shared options](#shared-options-and-prompts) only.

Sync needs access to every configured repository and uses your existing Git credentials. It never moves a rule to a newer version on its own: it restores the version recorded in `vendor/<source-name>/_source.json`. Sync chooses a rule's version only when configuration asks for something the record doesn't have:

| Situation | Version sync imports |
| --- | --- |
| A new source, a changed repository, or a newly selected group or rule | Each new rule's newest version. |
| A pin added or changed | The pinned version, up or down. |
| A pin removed | The recorded version, unchanged. The next update offers newer versions. |
| `ref` added or changed | The rule as it was at that revision: the version a library release published, or `null` for unreleased changes. |
| `ref` removed | The recorded version when it's a published version; otherwise the newest version. |

These changes come from edits you made to configuration, so sync applies them without a preview. To move rules to newer versions, use [project update](#project-update). If a source fails, the previous complete output is preserved. For recovery behavior, see [Sync and recovery](/reference/sync/).

When a source imports unreleased changes through `ref`, sync prints a warning naming the source and its unreleased rules. When an exclusion, replacement, or individually selected rule names a rule the library retired, sync warns that the entry no longer does anything.

### project update

```sh
code-rules project update [SOURCE | SOURCE:RULE ...] [options]
```

Preview newer rule versions, new rules, and retirements, then apply them after you confirm, and regenerate guidance like sync. With no arguments, every source is updated. `SOURCE`, such as `team`, limits the update to one library. `SOURCE:RULE`, such as `team:techs/react/prefer-server-components`, moves only that rule; nothing else changes, including new rules.

| Option | Meaning |
| --- | --- |
| `--yes` | Apply the previewed changes without asking. Required to apply changes without a terminal, or with `--json`. |
| `--keep SOURCE:RULE` | Pin this rule at its current version before applying the update, so it stays where it is. Repeat for several rules. Requires `--reason`. |
| `--exclude SOURCE:RULE` | Exclude this new rule before applying the update, so it doesn't join. Repeat for several rules. Requires `--reason`. |
| `--reason TEXT` | The reason recorded with each pin that `--keep` writes and each exclusion that `--exclude` writes. |

The preview lists, for each source:

| Change | Meaning |
| --- | --- |
| `major`, `minor`, or `patch` | The rule's old and new versions, and the summary of every version in between. |
| `new` | A rule the library added to a selected group, at its newest version. |
| `retired` | A rule the library retired, with its last version, its summary, and its replacement when there is one. Applying the update drops it. |
| `pinned` | A pinned rule that has a newer version. It doesn't move; the preview shows the pin's reason. |

In a terminal, `code-rules project update` shows the preview and asks for confirmation. For each major change and retirement, you can choose to keep the rule at its current version instead; the command then asks for a reason and writes a pin. For each new rule, you can choose to add it or exclude it; excluding asks for a reason and writes an exclusion. Without a terminal, or with `--json`, it shows the preview and writes nothing unless you pass `--yes`. The update applies exactly the versions the preview showed.

`--keep` and pins in configuration never change which rules are imported. When a rule you exclude, replace, or select individually is retired, the entry no longer does anything; update warns about it so you can delete it, and applies the rest of the update.

Sources that use `ref` don't move; change `ref` and run `code-rules project sync` instead. Group metadata and the library's license files come from the newest library release among the rule versions the project imports.

See [Update rules](/guides/update/) for the workflow.

### project build

```sh
code-rules project build [options]
```

Regenerate `generated/` from project configuration, verified imported files, and local rules. Build works offline and does not change imported revisions. It also creates a missing managed Code Rules guide or refreshes an older, unedited guide alongside generated output. A manually edited guide stops the build before it changes output.

Accepts the [shared options](#shared-options-and-prompts) only.

Run sync first if the imported repository, selection, pins, or `ref` no longer match configuration, or if the stored library files need repair.

### project check

```sh
code-rules project check [options]
```

Check generated guidance and the managed Code Rules guide without writing files or contacting repositories. Reports stale, missing, or unexpected output and invalid inputs.

Accepts the [shared options](#shared-options-and-prompts) only.

Use check in CI to detect files that need regeneration. It uses recorded commits and rule versions, and does not check whether remote tags moved or newer library releases exist. Success means the managed files agree with their inputs; it does not establish that application code follows the rules.

## Library commands

`code-rules library` manages a **library**, an independently maintained collection of rule groups that projects can import. Initialize from the library repository root. Other library commands find that root from subdirectories. Use `--directory PATH` to target another library; see [Working directories](#working-directories).

### library init

```sh
code-rules library init [options]
```

Create `rule-library.yaml`, an authoring README, and a GitHub Actions workflow without overwriting existing authored files. Optionally copy explicitly supplied license terms into the library.

The workflow, `.github/workflows/code-rules.yml`, runs `code-rules library check` on every pull request, using the Code Rules version that created it. See [Check changes in CI](/guides/version-rules/#check-changes-in-ci).

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. Init must target the repository root. |
| `--spdx EXPRESSION` | Library SPDX expression. Must be supplied together with `--license-file`. |
| `--license-file PATH` | UTF-8 license text to copy to `LICENSE.md`. Relative to your working directory, even with `--directory`. |
| `--notice-file PATH` | Optional UTF-8 notice text to copy to `NOTICE.md`. Requires both license options. Relative to your working directory. |
| `--non-interactive` | Accepted; library init runs without prompts even when this flag is omitted. |

If you omit the license options, `rule-library.yaml` leaves the license undeclared. Init does not infer license terms, create a Git repository, commit, or publish the library.

### library add group

```sh
code-rules library add group ID [options]
```

Create a group with `_group.yaml` metadata and an authoring README in the library. `ID` is a group path such as `practices/testing` or `techs/typescript`.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
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
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--title TEXT` | Required. Action-oriented rule title. |
| `--when-to-read TEXT` | Required. When an agent should read the rule. |
| `--impact LEVEL` | Required. One of `CRITICAL`, `HIGH`, `MEDIUM-HIGH`, `MEDIUM`, `LOW-MEDIUM`, or `LOW`. |
| `--impact-description TEXT` | Required. The consequence the rule helps prevent. |
| `--body-file PATH` | Optional UTF-8 Markdown body, without frontmatter. Relative paths start at your working directory. Omit to create an unfinished draft. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Create the group first with `code-rules library add group`; rule creation does not create missing groups. Without `--body-file`, complete the draft and remove its `code-rules:draft` marker before validation. Existing rules are not overwritten. After the first library release, the next steps include adding the new rule's change note with `code-rules library change`.

### library change

```sh
code-rules library change ID... [options]
```

Write a new [change note](/reference/rule-versions/#change-notes) for one or more rules. `ID` is a rule ID, such as `practices/testing/verify-retry-limits`; name several rules to cover related changes in one note. Before the first library release, rules need no notes.

| Option | Meaning |
| --- | --- |
| `--bump LEVEL` | `major`, `minor`, or `patch`. Required for rules that have a version. Not accepted for new or retired rules. |
| `--summary TEXT` | Required. One line describing the change for project maintainers. |
| `--retire` | Record that the rules are [retired](/reference/rule-versions/#retired-rules). Their Markdown files must already be gone. |
| `--replaced-by ID` | Optional with `--retire` and a single `ID`: the rule that replaces the retired one. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

The command writes a new file in `changes/` with a unique name, such as `changes/2026-09-29-verify-retry-limits.yaml`, and never edits or deletes existing notes. A rule without any version is recorded as `new`. It rejects an ID that isn't a rule in the library, unless `--retire` is supplied for a rule that has a version. A rule that was never published can't be retired: delete its Markdown file, and remove it from any pending note.

See [Choose a version change](/reference/rule-versions/#choose-a-version-change) for picking `LEVEL`.

### library check

```sh
code-rules library check [options]
```

Validate `rule-library.yaml`, all groups and rules, change notes, supporting assets, and declared license and notice files. After the first library release, also check that every changed rule has a change note. Does not change files.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--non-interactive` | Accepted; library check does not prompt. |

Reports group and rule counts and file-specific errors. Empty groups are valid. Unfinished marked drafts fail. Undeclared licenses produce warnings; invalid declarations and missing declared files fail validation. Library check validates the format, not writing quality or legal permissions. Use the [authoring rubric](/reference/rule-authoring/#authoring-rubric) to review guidance quality.

After the first library release, check compares each rule's [versioned content](/reference/rule-versions/#what-a-version-covers), meaning its Markdown file and its asset directory, with the latest `release/<number>` tag in the current branch's history. Notes added since that tag are pending. Check fails when:

- A rule changed and no pending note names it. The error names the rule and its latest version, and gives the `code-rules library change` command to run.
- A new rule has no pending note.
- A published rule was deleted without a pending note that retires it.
- A retired rule's `replacedBy` isn't a rule in the library.
- A rule reuses the ID of a retired rule.
- A pending note names a rule that's unchanged since the latest library release, so the note is stale.
- A pending note names a rule that doesn't exist and isn't being retired.
- A note is invalid, such as an unknown change or a blank summary.

Check warns when a note that a library release already published was edited, because the edit has no effect.

This comparison needs the repository's history and tags. Check fails with instructions in a shallow clone; in CI, check out with full history, such as `fetch-depth: 0`. Before the first library release, rules need no notes, and check validates everything else.

When checks pass, the result previews the pending library release: each rule, its change, and its current and next version. JSON output includes this preview in `value.pendingRelease`.

### library release

```sh
code-rules library release [options]
```

Publish the pending change notes, and any changes to library-wide files, as a [library release](/reference/rule-versions/#library-releases). See [Version your rules](/guides/version-rules/#publish-a-library-release) for the workflow.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--dry-run` | Show exactly what would be published, including the release notes, without creating a tag or a GitHub Release page. |
| `--no-github-release` | Skip creating the GitHub Release page. |
| `--non-interactive` | Accepted; library release does not prompt. |

Before publishing, the command fetches from the branch's upstream remote. It refuses unless:

- you're on the remote's default branch,
- the branch matches the remote exactly, so the library release contains no unpushed commits and misses none, and
- `code-rules library check` passes.

It then computes each changed rule's next version from the pending notes, using the largest change when several notes name the same rule. The first library release gives every rule version `1.0.0`. It creates the annotated `release/<number>` tag on the current commit, with the release notes and [release record](/reference/rule-versions/#release-record) as its message, and pushes it. No files change and nothing is committed. Finally, it creates the GitHub Release page.

`--dry-run` shows the repository, branch, commit, release number, each rule's change and versions, and the complete release notes.

If a run stops after pushing the tag, such as when the GitHub Release page can't be created, run it again: it finds the tag on the current commit and creates what's missing. The result reports separately whether the tag and the GitHub Release page were created. With no pending notes and no library-wide changes, the command reports that there is nothing to publish.

The tag uses Git's configured identity as its tagger, including the `GIT_COMMITTER_*` environment variables.

**GitHub Release pages** are created with the [GitHub CLI](https://cli.github.com/), `gh`, for repositories on GitHub.com. Each library release gets one GitHub Release page on its `release/<number>` tag, with the release notes as its body. Before changing anything, `code-rules library release` checks that `gh` is installed and signed in, and refuses if it isn't, unless you pass `--no-github-release`. Repositories hosted elsewhere get tags only.

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

Running `code-rules` or a command group such as `code-rules library` without a subcommand displays help. The `code-rules project`, `code-rules project add`, `code-rules library`, and `code-rules library add` groups organize commands; they do not perform operations themselves. Help never prompts or writes files.

The root `--version` flag prints the tool version. To choose which versions of a library's rules to import, see [Choose versions](/reference/configuration/#choose-versions).

## Working directories

In Git repositories, run `code-rules project init` or `code-rules library init` from the repository root. Initialization from a subdirectory fails without writing files. This also applies to the target of `code-rules library init --directory`.

Other commands find the nearest ancestor containing a `.git` directory or file, including worktrees and submodules. Project commands use that root's `.code-rules/config.yaml`; library commands use its `rule-library.yaml`. They stop at that repository boundary, even if its configuration is missing. A nested `.code-rules/` does not override the root's configuration.

Outside Git, commands use the current directory, or the directory selected by `--directory` for library commands. They do not search parent directories for configuration. Custom project configuration locations are not supported.

Relative input paths such as `--body-file`, `--license-file`, and `--notice-file` resolve from the directory where you run the command. Paths inside project configuration remain relative to `.code-rules/`.

## Shared options and prompts

Every command accepts `--json`, which prints one JSON response and disables prompts. Every command also accepts `-h` / `--help`.

Authoring commands accept `--non-interactive`. Without it, commands can prompt for missing required metadata or source selections when running in a terminal. With `--non-interactive`, `--json`, or no terminal, missing required inputs cause an error. Positional arguments such as `ID` and `ALIAS` must always be supplied.

Invalid interactive answers repeat the same question while retaining earlier answers. Explicit flags are validated without prompting for replacement values. Existing groups, rules, and library aliases fail before prompts.

Both init commands, both check commands, and `code-rules library release` run without prompts. Of these, `code-rules library check` and `code-rules library release` accept `--non-interactive`; `code-rules project check`, `code-rules project sync`, `code-rules project update`, and `code-rules project build` do not need or accept it.

String options accept one value and cannot be repeated, except `--groups`, `--rules`, and `--keep`, which accept repeated values. For a value beginning with `-`, use the equals form, such as `--description='-prefixed text'`.

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

Only sync, update, and build report `added`, `changed`, and `removed` file lists. Update also reports each source's rule changes in `value.sources`. When they refresh the managed Code Rules guide, `value.guide` reports its path relative to the Code Rules directory and whether it was `created`. Help, version, and license return their text in `value.text`.

In human mode, operational errors go to stderr. An out-of-date check prints its status, problems, and next steps on stdout. In JSON mode, errors go in the response; stderr is reserved for failures writing that response. Unreleased preview builds also print a non-production warning with their source commit to stderr before every command, including help, version, and JSON commands. JSON output on stdout is unchanged. See [testing PR preview builds](https://github.com/fabricahq/code-rules/blob/main/_engineering/releasing.md#testing-pr-preview-builds).

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

Library command paths are unchanged. For configuration fields that changed, see [Configuration](/reference/configuration/#fields).
