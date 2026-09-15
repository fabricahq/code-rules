---
title: "CLI commands"
description: "The proposed rule import, validation, conflict-review prompt, and tool update commands."
---

**These commands are proposed and are not available in a release yet.**
No installation command is published on this site.

For `sync`, `build`, `check`, and `conflicts --prompt`, run from the consuming project's root by default.
Use `--config` to identify a configuration file elsewhere.

## Sync

This command is proposed. The Imports API implements fetching; the CLI and safe project file updates belong to a dependent PR.

```sh
code-rules sync
```

Resolve each source's exact ref or highest matching version tag to a full commit SHA, validate all libraries, and generate resolved rules together.
Record the requested ref or version constraint, selected version tag when applicable, and resolved commit in the vendored provenance.
Each sync resolves tags again and reports changed commit targets, including when the configured tag name stays the same.
Use the caller's existing Git credentials for private repositories.
Accept the explicit [Git addresses](/reference/configuration/#repository-addresses) in configuration, independently of the hosting provider.
Compare updates against the previous vendor snapshot before installing the replacement.

Sync requires access to all configured source repositories.
If any source fails, preserve the previous complete output.

## Build

```sh
code-rules build
```

Generate resolved rules from committed vendor content, local rules, and configuration.
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

```sh
code-rules conflicts --prompt
```

`--prompt` is required.
Running `code-rules conflicts` without it exits with a usage error that shows the required flag.

Print a Markdown prompt for an agent to review the project's active rules for contradictory guidance.
The prompt includes the effective root index, group index, and rule file paths, source refs and resolved commits, and a digest identifying the reviewed inputs and output.
Generate it from the configured project paths so the agent can read the exact snapshot.
The prompt is for an agent with access to those repository files; it does not embed the complete rule corpus.

Validate the inputs and generated output using the same consistency checks as `check` before emitting a prompt.
If validation fails, exit nonzero and explain the required rebuild or sync on stderr; do not emit a review prompt.
Send the successful Markdown prompt to stdout and operational diagnostics to stderr.
Generation works offline and leaves repository files unchanged.

The prompt instructs the agent to compare all effective groups and cite source-qualified IDs, conflicting passages, overlapping applicability, and proposed resolutions.
It includes local additions and replacements, while treating excluded rules and replaced upstream definitions as inactive context.
It asks the agent to distinguish contradictions from ambiguous scope and redundant guidance, and to report incomplete coverage.

The CLI generates the review instructions; an agent performs the semantic review.
A zero exit status confirms prompt generation, not the absence of contradictions.
The command does not choose policies, apply exceptions, or launch an agent.

See [Conflicting guidance](/guides/conflicting-guidance/) for a prompt you can use today and examples of resolving findings.

## Update the tool

```sh
code-rules update
```

Upgrade the Code Rules CLI to the latest stable release.
The command reports the current version and the installed version, or confirms that the tool is already up to date.

Use the existing installation method when applying the upgrade.
For installations managed by a package manager, preserve that manager's ownership of the installation.
If an installation cannot be updated automatically, provide the specific command needed to upgrade it.
A failed upgrade must preserve a working installation.

This command updates the tool itself.
Library refs, vendored rules, and generated rule files remain unchanged.
Use `code-rules sync` to download rule libraries and rebuild their indexes and resolved definitions.

After upgrading, run `code-rules check` in a consuming project to check its generated files against the new tool version.
If regeneration is needed, run `code-rules build` and review the output before committing it.

## Explicit configuration

```sh
code-rules check --config path/to/code-rules/config.json
```

Resolve local paths relative to the chosen configuration directory.
Keep replacement files within that directory's `local/` tree.

## Author a library

These proposed commands run from the library root, independently of a consuming project's configuration.
They form the final implementation phase. Follow [Create a rule library](/guides/create-library/) for the complete workflow.

```sh
code-rules library init
code-rules library add group techs/javascript
code-rules library add rule techs/javascript/prefer-for-of
code-rules library check
```

- `library init` creates the format manifest and a README pointing to the canonical authoring guidance. It records an explicit license choice and retains the corresponding text and notices. Authors may defer that choice; the manifest then leaves it undeclared.
- `library add group <group-id>` creates `_group.json` under `techs/` or `practices/`, collecting the name, description, and group-level `whenToRead` cues. A group can exist before it has rules.
- `library add rule <rule-id>` creates a Markdown draft from the canonical template in an existing group. A missing group produces an error with the command to create it. Required fields need author input; the command does not invent policy.
- `library check` validates the manifest, all group and rule definitions, and declared license and notice assets offline without writing files. It reports file-specific errors and group and rule counts. Empty groups are valid. Undeclared licenses produce warnings; malformed declarations and missing declared files fail validation.

Scaffolding commands validate paths and detect collisions before writing. They never overwrite existing files or leave partial scaffolds after a failed operation.
They do not create Git repositories, commit, publish, infer licenses, or convert arbitrary upstream material.
Interactive prompts collect missing author input. Noninteractive use must accept equivalent explicit inputs and fail with actionable errors when required input is missing.
Exact input flags will be specified with implementation.

Library validation checks the input format, not the quality of the guidance or legal permissions.
Consumer `code-rules check` instead verifies generated output against adopted inputs.
For non-native material, follow [Adapt a third-party rule](/guides/adapt-rules/).

## Errors

Report the affected file or rule ID and the action needed to resolve the problem.
Examples include a missing override target, an unsupported format version, or a vendor snapshot that needs sync.
Exit nonzero on failure and preserve the previous working output if installation fails.
