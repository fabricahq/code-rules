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
| `--ref REF` | Optional. A tag, such as the rule version tag `practices/testing/verify-retry-limits@1.3.0`, or a full Git commit, to pin the source. Omit it to follow the library's releases. See [Select a revision](/reference/configuration/#select-a-revision). |
| `--groups GROUP` | Required. Repeat for multiple group IDs, or supply one selector: `*`, `practices/*`, or `techs/*`. Quote wildcard values so your shell does not expand them. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

The CLI records the value in the configuration's `ref` field. Without `--ref`, the source has no `ref` and follows the library's releases. When prompting, leave the revision blank to follow releases.

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

Copy one version of a library rule into `local/` so the project controls it. Use a fork to stay on an older major version, or to adopt one rule without importing its library. `ID` is the library rule ID, such as `practices/testing/verify-retry-limits`, and the fork keeps it: `local/practices/testing/verify-retry-limits.md`.

| Option | Meaning |
| --- | --- |
| `--from LIBRARY@VERSION` | Required for a fork. `LIBRARY` is a configured source name, such as `team`, or a [repository address](/reference/configuration/#repository-addresses). `VERSION` is one of the rule's versions, such as `1.3.0`. |
| `--reason TEXT` | Why the project replaces the imported rule. Required when the project imports this rule from `LIBRARY`. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

Metadata options and `--body-file` don't apply to a fork. The command:

1. Reads the rule and its asset directory at the tag `<ID>@<VERSION>`.
2. Writes them under `local/` and creates the local group from the library's group metadata if it doesn't exist.
3. Adds an `attribution` entry that links to the rule at the tag's commit. Libraries hosted outside GitHub.com and GitLab.com get an attribution entry only when the repository address is an HTTPS URL.
4. When the project imports the rule from `LIBRARY`, adds a `replace` entry for it with your reason, so agents read only the fork.

The fork fails if the rule links to the library's shared `assets/` directory, because local rules can't depend on library files. Existing local rules are never overwritten. A forked rule has no version; it changes only when you edit it. Run `code-rules project build` afterward. The library's license still applies to the copied text; see [License rules](/guides/license-rules/).

### project sync

```sh
code-rules project sync [options]
```

Import the library revisions recorded for each source, validate their files, and replace `vendor/` and `generated/` in the Code Rules directory with the complete result. Sync also refreshes an older, unedited managed Code Rules guide.

Accepts the [shared options](#shared-options-and-prompts) only.

Sync needs access to every configured repository and uses your existing Git credentials. It doesn't move a source to a newer revision. A source keeps the commit recorded in its `vendor/<source-name>/_source.json` while its repository and `ref` are unchanged. Sync resolves a source again only when it is new, or when you changed its repository or `ref`. A new source without a `ref` imports the library's newest release. To adopt newer revisions, use [project update](#project-update).

Sync also records each imported rule's version. If a source fails, the previous complete output is preserved. For recovery behavior, see [Sync and recovery](/reference/sync/).

### project update

```sh
code-rules project update [SOURCE...] [options]
```

Move each source that follows its library's releases to the newest release, report what changed, then regenerate guidance like sync. `SOURCE` limits the update to the named sources; by default, every source is updated.

| Option | Meaning |
| --- | --- |
| `--accept-major` | Apply the update even when rules the project uses have major changes or were retired. |

A source pinned with `ref` doesn't move; change or remove its `ref` and run `code-rules project sync` instead.

Update reports each rule that changed between the recorded release and the new one: its change (`new`, `major`, `minor`, `patch`, or `retired`), its old and new versions, and each version's summary. A retired rule shows its reason, `superseded` or `withdrawn`, and any replacement. It also lists rules that joined or left the selected groups.

Update refuses to write anything, and exits with status `1`, when a rule the project uses has a major change or was retired. Excluded and replaced rules don't need consent; update lists their changes so you can review your exceptions. Review the reported changes, then rerun with `--accept-major`. To stay on a rule's older major version instead, [fork it](#fork-a-library-rule) first.

A retired rule that the configuration still excludes or replaces makes the configuration invalid at the new revision. Update names the entry to delete and changes nothing.

See [Update rules](/guides/update/) for the workflow.

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

Use check in CI to detect files that need regeneration. It uses recorded commits and rule versions, and does not check whether remote tags moved or newer releases exist. Success means the managed files agree with their inputs; it does not establish that application code follows the rules.

## Library commands

`code-rules library` manages a **library**, an independently maintained collection of rule groups that projects can import. Initialize from the library repository root. Other library commands find that root from subdirectories. Use `--directory PATH` to target another library; see [Working directories](#working-directories).

### library init

```sh
code-rules library init [options]
```

Create `rule-library.yaml`, an authoring README, and a GitHub Actions workflow without overwriting existing authored files. Optionally copy explicitly supplied license terms into the library.

The workflow, `.github/workflows/code-rules.yml`, checks pull requests and releases rules from `main`, and installs the Code Rules version that created it. See [Version your rules](/guides/version-rules/#automate-releases-with-github-actions).

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

Create a group with `_group.yaml` metadata and an authoring README in the library. `ID` is a group path such as `practices/testing` or `techs/typescript`.

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

Create the group first with `code-rules library add group`; rule creation does not create missing groups. Without `--body-file`, complete the draft and remove its `code-rules:draft` marker before validation. Existing rules are not overwritten. After the library's first release, the next steps include adding the new rule's change note with `code-rules library change`.

### library change

```sh
code-rules library change ID [options]
```

Create or update the [change note](/reference/rule-library-format/#change-notes) for one rule. `ID` is the rule ID, such as `practices/testing/verify-retry-limits`. Before a library's first release, rules need no notes.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--bump LEVEL` | `major`, `minor`, or `patch`. Required for a rule that has a version. Not accepted for a new or retired rule. |
| `--summary TEXT` | Required. One line describing the change. |
| `--retire REASON` | Record that the rule is [retired](/reference/rule-library-format/#retired-rules): `superseded` or `withdrawn`. The rule's Markdown file must already be gone. |
| `--replaced-by ID` | The rule that replaces a `superseded` rule. Required with `--retire superseded`, and rejected otherwise. |
| `--non-interactive` | Never prompt. Supply all required inputs as flags. |

A rule without any version is new, and its note omits `bump`. When a note already exists, the command keeps the larger of the two bumps and appends the new summary line. It rejects an ID that isn't a rule in the library, unless `--retire` is supplied for a rule that has a version. A rule that was never released can't be retired: delete its Markdown file and its note together.

See [Choose a version change](/reference/rule-library-format/#choose-a-version-change) for picking `LEVEL`.

### library check

```sh
code-rules library check [options]
```

Validate the library manifest, all groups and rules, supporting assets, and declared license and notice files. After the library's first release, also check change notes against the last release. Does not change files.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. Init must target the repository root. |
| `--non-interactive` | Accepted; library check does not prompt. |

Reports group and rule counts and file-specific errors. Empty groups are valid. Unfinished marked drafts fail. Undeclared licenses produce warnings; invalid declarations and missing declared files fail validation. Library check validates the format, not writing quality or legal permissions. Use the [authoring rubric](/reference/rule-authoring/#authoring-rubric) to review guidance quality.

After the first release, check compares each rule's Markdown file and asset directory with the most recent release commit in the current branch's history:

| Situation | Result |
| --- | --- |
| A rule changed and has a valid note. | Passes. |
| A rule changed and has no note. | Error naming the rule and its latest version, with the `code-rules library change` command to run. |
| A new rule has no note. | Error. |
| A released rule was deleted without a `retired` note. | Error. |
| A superseded rule's `replacedBy` isn't a rule in the library. | Error. |
| A rule reuses the ID of a retired rule. | Error. |
| A note's rule is unchanged since the last release. | Error: the note is stale. |
| A note has no matching rule and isn't a retirement. | Error: the note is orphaned. |
| A note is invalid, such as an unknown `bump` or a blank summary. | Error. |
| A tag is named like a group or folder that contains rules. | Error. |

This comparison needs the repository's history and tags. Check fails with instructions in a shallow clone; in CI, check out with full history, such as `fetch-depth: 0`. Before the first release, every rule is new.

When checks pass, the result previews the pending release: each rule, its change, and its current and next version. JSON output includes this preview in `value.pendingRelease`.

### library release

```sh
code-rules library release [options]
```

Publish rule versions from the pending change notes. Requires a Git remote. See [Version your rules](/guides/version-rules/) for the workflow.

| Option | Meaning |
| --- | --- |
| `--directory PATH` | Directory to operate in. Defaults to your working directory; repository discovery applies. |
| `--dry-run` | Report what the release would do without changing files, commits, tags, or GitHub. |
| `--pr` | Open or update the release pull request instead of releasing. For CI. |
| `--publish` | Tag and publish a merged release pull request. For CI. |
| `--bump-all major` | Add a major change to every rule, then release. Requires `--summary`. |
| `--summary TEXT` | The change summary used with `--bump-all`. |
| `--no-github-release` | Skip creating GitHub Releases. |
| `--non-interactive` | Accepted; library release does not prompt. |

`--pr` and `--publish` can't be combined with each other or with `--bump-all`.

**Without a mode**, release runs locally, for maintainers who push directly to the default branch. It refuses when the working tree has uncommitted changes, the branch is behind its upstream, or `code-rules library check` fails. Then it:

1. Computes each rule's next version from its note, or its retirement.
2. Deletes every note and commits the result as `Release N rules`.
3. Creates an annotated tag for each new version, and a `<rule-id>@retired` tag for each retirement, on that commit.
4. Pushes the commit, the tags, and the `code-rules/released` branch in one atomic push, so either all of them arrive or none do.
5. Creates a GitHub Release for each tag.

With no pending notes, release reports that there is nothing to release.

**The first release** gives every rule version `1.0.0`, with the message `new: Initial version.` It runs locally, even in a library that uses the release workflow: until then, `--pr` does nothing and reports that the first release is pending. If notes exist, the first release deletes them and commits as usual; otherwise it tags the current commit.

**`--pr`** keeps one release pull request up to date. It force-updates the branch `code-rules/release-pr` with a single commit on top of the current commit that deletes every pending note, and opens or updates the pull request "Release rules". The description lists each rule, its change, its current and next version, its summary, and the pull requests that added its note when GitHub can find them. When nothing is pending, it closes an open release pull request and exits successfully.

**`--publish`** runs on the commit created by merging the release pull request, and does nothing on any other commit. It reads the deleted notes from the commit's parent, computes versions, creates the tags on the merge commit, pushes them and the `code-rules/released` branch atomically, and creates GitHub Releases. It refuses when any note remains at that commit, or when `code-rules/released` isn't an ancestor of it. Rerunning it is safe: tags that already point to the commit are kept, missing GitHub Releases are created, and a tag that points elsewhere stops the command.

**`--bump-all major`** marks a clean break: it adds a major change with your summary to every rule's note, then releases locally as above.

**GitHub Releases** are created with the [GitHub CLI](https://cli.github.com/), `gh`, for repositories on GitHub.com. Each release is named after its tag and uses the change summary as its body. Before changing anything, release checks that `gh` is installed and signed in, and refuses if it isn't, unless you pass `--no-github-release`. Repositories hosted elsewhere get tags only.

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

Running `code-rules` or a command group such as `code-rules library` without a subcommand displays help. The `project`, `code-rules project add`, `library`, and `code-rules library add` groups organize commands; they do not perform operations themselves. Help never prompts or writes files.

The root `--version` flag prints the tool version. To select a library revision, use `code-rules project add library --ref` instead.

## Working directories

In Git repositories, run `code-rules project init` or `code-rules library init` from the repository root. Initialization from a subdirectory fails without writing files. This also applies to the target of `code-rules library init --directory`.

Other commands find the nearest ancestor containing a `.git` directory or file, including worktrees and submodules. Project commands use that root's `.code-rules/config.yaml`; library commands use its `rule-library.yaml`. They stop at that repository boundary, even if its configuration is missing. A nested `.code-rules/` does not override the root's configuration.

Outside Git, commands use the current directory, or the directory selected by `--directory` for library commands. They do not search parent directories for configuration. Custom project configuration locations are not supported.

Relative input paths such as `--body-file`, `--license-file`, and `--notice-file` resolve from the directory where you run the command. Paths inside project configuration remain relative to `.code-rules/`.

## Shared options and prompts

Every command accepts `--json`, which prints one JSON response and disables prompts. Every command also accepts `-h` / `--help`.

Authoring commands accept `--non-interactive`. Without it, commands can prompt for missing required metadata or source selections when running in a terminal. With `--non-interactive`, `--json`, or no terminal, missing required inputs cause an error. Positional arguments such as `ID` and `ALIAS` must always be supplied.

Invalid interactive answers repeat the same question while retaining earlier answers. Explicit flags are validated without prompting for replacement values. Existing groups, rules, and library aliases fail before prompts.

Both init commands, both check commands, and `code-rules library release` run without prompts. Of these, `code-rules library check` and `code-rules library release` accept `--non-interactive`; project `check`, `sync`, `update`, and `build` do not need or accept it.

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

Library commands and the on-disk configuration and directory layout are unchanged.
