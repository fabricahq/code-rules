---
title: "Set up a project"
description: "Start with local rules, then add shared libraries without moving your authored files."
---

Run these commands from the repository where you want to use rules.
The examples assume `code-rules` is installed and available on your PATH. See [CLI commands](/reference/cli/) for availability and development setup.

For an executable you can run outside the checkout, [install the packed release candidate](/guides/install/).

## Start with local rules

```sh
code-rules project init
code-rules project add group practices/testing
code-rules project add rule practices/testing/retry-budget
```

`init` creates `.code-rules/config.json` with no imported sources and `.code-rules/local/README.md`.
No imported library or existing Git repository is required. Repeating `init` preserves valid configuration and local rules and refreshes the managed project README. It refuses to overwrite manually edited guides; keep project notes in a separate file.
Human output starts with a setup confirmation and example commands for project-only rules or a shared library. If setup is already current, it reports that no files changed. Use `--json` for the exact changed file paths.

**Native Go candidate:** `init` also creates a tool-owned agent guide at `.code-rules/README.md`. For a custom configuration outside a directory named `.code-rules`, the guide is `CODE_RULES.md` beside that configuration. Your project's own `README.md` stays unchanged. Repeating `init` refreshes an older generated guide only if you have not edited it. If the guide contains manual edits, init stops and explains how to preserve them before refreshing. Configuration and local rules remain unchanged. Run `check` to verify both the guide and generated guidance.

On a terminal, group creation asks for a name, description, and when-to-read cue. Rule creation asks for its title, when-to-read cue, impact, and consequence.
If the group is missing, rule creation stops before prompting and tells you to create the group first.
A missing group or cancelled prompt writes nothing.

The new Markdown rule is a **draft** from the [canonical template](/reference/rule-authoring/). Complete its obligation, examples, implementation, and validation guidance before building. Remove prompts and sections that add no useful guidance.
A successful format check does not establish that a draft is finished or that its guidance is correct.

```sh
code-rules project build
code-rules project check
```

Open `.code-rules/generated/RULES.md` and review the generated rules. Commit configuration, local rules, and generated files.
Use the [agent integration instructions](/for-agents/) to connect the rules to your existing workflow; setup does not modify `AGENTS.md`.

## Add a library later

```sh
code-rules project add library team \
  --repository https://github.com/example/rules.git \
  --version '>= 1.2.0, < 2.0.0' --groups '*'
code-rules project sync
```

Replace the example repository and version with a library you can access.
`project add library` validates and records the declaration. It does not fetch, verify remote existence, or change generated files. `project sync` performs those steps explicitly.
Use exactly one of `--ref` (an exact tag or full commit) or `--version` (a HashiCorp version constraint).

Use `--groups '*'`, `--groups 'practices/*'`, `--groups 'techs/*'`, or repeat `--groups` for explicit IDs. Quote wildcard values in your shell. Wildcards cannot be combined with other selectors.
Existing sources, exclusions, replacements, and local rule files are preserved. An existing source alias is an error; edit its configuration explicitly to change it.
Local group metadata remains the project's description when imported rules join the same group.

## Use explicit inputs in agents and scripts

Every prompt has an equivalent flag. A non-terminal invocation never prompts. `--non-interactive` also disables prompts when running from a terminal.
Missing required inputs, unknown flags, and repeated single-value flags produce a usage error without authoring files.

```sh
code-rules project add group practices/testing \
  --name Testing \
  --description 'Verify observable project behavior.' \
  --when-to-read 'Before planning, changing, or reviewing project behavior.' \
  --non-interactive

code-rules project add rule practices/testing/retry-budget \
  --title 'Bound retry attempts' \
  --when-to-read 'When implementing or reviewing retry behavior.' \
  --impact HIGH \
  --impact-description 'Unbounded retries can overload an unavailable service.' \
  --non-interactive
```

Pass `--when-to-read` once for either a group or a rule. Combine distinct scope cues into that one string.
Optionally pass `--body-file path/to/guidance.md` to use an already authored Markdown body instead of the draft body. The command creates frontmatter from the explicit metadata flags; the body file should not contain frontmatter.

Create a missing group first with `code-rules project add group GROUP_PATH`, then run `code-rules project add rule`. The Go CLI errors before asking for rule metadata if the group does not exist. A selected group from a verified imported library also counts as an existing group.
Creating a local group establishes the project's description even when a library supplies the same group. Existing local metadata is never overwritten. Sync a configured library before relying on its group metadata.

## File ownership

All commands accept `--config path/to/config.json`; local, vendor, and generated directories live beside that file. Only `init` creates a missing project configuration.

Authoring commands serialize writes with the same project lock used by sync and build. They reject links, unsupported files, and case-colliding target paths. New definitions are never overwritten. Adding a source moves the current configuration aside, verifies its exact bytes, and installs the new file only if the target path remains empty. An editor save is preserved rather than overwritten.
A failed multi-file creation claims newly created files before comparing their bytes for rollback, preserving external replacements.
Cancellation stops further publication and rolls back new files. Once a configuration replacement is published, it remains a complete result.
If the original cannot be restored because another file occupies its path, Code Rules retains the original in the reported `.code-rules-authoring-*` directory. Inspect the saved files and `recovery.json`, keep the intended content, then remove that temporary directory before retrying authoring. Interrupted operations may also leave this directory for inspection. Setup does not fetch libraries, commit files, publish content, or prescribe rule enforcement.
