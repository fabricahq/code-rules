---
title: "Set up a project"
description: "Start with local rules, then add shared libraries without moving your authored files."
---

Run these commands from the repository where you want to use rules.
The examples assume `code-rules` is installed and available on your PATH. See [CLI commands](/reference/cli/) for availability and development setup.

## Start with local rules

```sh
code-rules init
code-rules local add group practices/testing
code-rules local add rule practices/testing/retry-budget
```

`init` creates `.code-rules/config.json` with no imported sources and `.code-rules/local/README.md`.
No imported library or existing Git repository is required. Repeating `init` preserves existing valid configuration, README text, and rules.

On a terminal, group creation asks for a name, description, and when-to-read cue. Rule creation asks for its title, when-to-read cue, impact, and consequence.
If the group is missing, rule creation offers to create it and collects its metadata before writing either file.
Declining the offer or cancelling a prompt writes nothing.

The new Markdown rule is a **draft** from the [canonical template](/reference/rule-authoring/). Complete its obligation, examples, implementation, and validation guidance before building. Remove prompts and sections that add no useful guidance.
A successful format check does not establish that a draft is finished or that its guidance is correct.

```sh
code-rules build
code-rules check
```

Open `.code-rules/generated/RULES.md` and review the generated rules. Commit configuration, local rules, and generated files.
Use the [agent integration instructions](/for-agents/) to connect the rules to your existing workflow; setup does not modify `AGENTS.md`.

## Add a library later

```sh
code-rules add source team \
  --repository https://github.com/example/rules.git \
  --version '^1.2.0' --groups '*'
code-rules sync
```

Replace the example repository and version with a library you can access.
`add source` validates and records the declaration. It does not fetch, verify remote existence, or change generated files. `sync` performs those steps explicitly.
Use exactly one of `--ref` (an exact tag or full commit) or `--version` (an npm version constraint).

Use `--groups '*'`, `--groups 'practices/*'`, `--groups 'techs/*'`, or repeat `--groups` for explicit IDs. Quote wildcard values in your shell. Wildcards cannot be combined with other selectors.
Existing sources, exclusions, replacements, and local rule files are preserved. An existing source alias is an error; edit its configuration explicitly to change it.
Local group metadata remains the project's description when imported rules join the same group.

## Use explicit inputs in agents and scripts

Every prompt has an equivalent flag. A non-terminal invocation never prompts. `--non-interactive` also disables prompts when running from a terminal.
Missing required inputs, unknown flags, and repeated single-value flags produce a usage error without authoring files.

```sh
code-rules local add group practices/testing \
  --name Testing \
  --description 'Verify observable project behavior.' \
  --when-to-read 'Before planning, changing, or reviewing project behavior.' \
  --non-interactive

code-rules local add rule practices/testing/retry-budget \
  --title 'Bound retry attempts' \
  --when-to-read 'When implementing or reviewing retry behavior.' \
  --impact HIGH \
  --impact-description 'Unbounded retries can overload an unavailable service.' \
  --non-interactive
```

Repeat group `--when-to-read` for distinct scope cues. Rule `--when-to-read` is one string.
Optionally pass `--body-file path/to/guidance.md` to use an already authored Markdown body instead of the draft body. The command creates frontmatter from the explicit metadata flags; the body file should not contain frontmatter.

To create a missing local group with a rule, supply `--create-group`, `--group-name`, `--group-description`, and one or more `--group-when-to-read` flags.
This explicitly creates local metadata, so it also establishes the project's description when a library supplies the group. Existing local metadata is never overwritten.
Without these flags, an existing local or verified imported group is sufficient. Sync a configured library before relying on its group metadata.

## File ownership

All commands accept `--config path/to/config.json`; local, vendor, and generated directories live beside that file. Only `init` creates a missing project configuration.

Authoring commands serialize writes with the same project lock used by sync and build. They reject links, unsupported files, and case-colliding target paths. New definitions are never overwritten. Adding a source moves the current configuration aside, verifies its exact bytes, and installs the new file only if the target path remains empty. An editor save is preserved rather than overwritten.
A failed multi-file creation claims newly created files before comparing their bytes for rollback, preserving external replacements.
Cancellation stops further publication and rolls back new files. Once a configuration replacement is published, it remains a complete result.
If the original cannot be restored because another file occupies its path, Code Rules retains the original in the reported `.code-rules-authoring-*` directory. Inspect the saved files and `recovery.json`, keep the intended content, then remove that temporary directory before retrying authoring. Interrupted operations may also leave this directory for inspection. Setup does not fetch libraries, commit files, publish content, or prescribe rule enforcement.
