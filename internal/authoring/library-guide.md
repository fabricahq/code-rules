# Rule library

This folder contains a [Fabrica Code Rules](https://code-rules.fabricahq.com) library: a collection of engineering rules that projects can adopt from a Git repository.
Use this README to author and maintain the library.

## Core concepts

- **Rule:** One engineering practice in a Markdown file, with guidance, exceptions, and a cue explaining when to read it.
- **Group:** Related rules for a technology (`techs/go`) or practice (`practices/testing`). Each group has a description and a reading cue.
- **Library:** The groups and rules published together in this repository. Consuming projects choose a revision and groups to import.

Library groups supply shared guidance. A consuming project's local groups belong to that project and can supplement or explicitly override imported guidance.

## Instructions for agents

### Managing rules

1. Run the commands below from the folder containing this README. From elsewhere, pass `--directory` with this library's path.
2. Replace example metadata and guidance with the publisher's intended engineering practices.
3. Before adding or revising a rule, read the [rule authoring rubric](https://github.com/fabricahq/code-rules/blob/main/docs/src/content/docs/reference/rule-authoring.md).
4. After editing, run `code-rules library check`. Fix every error and review licensing warnings before reporting the library ready.

Human-readable output is the default. Add `--json` for structured results, or `--help` to inspect a command's options.

#### Add a group

Choose a technology or practice ID and supply its scope and reading cue:

```sh
code-rules library add group techs/go \
  --name 'Go' \
  --description 'Engineering practices for Go code.' \
  --when-to-read 'When writing or reviewing Go code.'
```

Confirm the metadata in `techs/go/_group.json`. Read `techs/go/README.md` before authoring rules in that group.

#### Add a rule

Create a complete Markdown body, then add its discovery metadata. This example refuses to overwrite an existing body file:

```sh
set -C
cat > return-errors.body.md <<'RULE_BODY'
# Return errors to the caller

Return a descriptive error when an operation fails. Let the caller decide whether to retry, report, or stop.
RULE_BODY
code-rules library add rule techs/go/return-errors \
  --title 'Return errors to the caller' \
  --impact HIGH \
  --impact-description 'Keep failures visible so callers can respond.' \
  --when-to-read 'When calling fallible operations.' \
  --body-file return-errors.body.md
```

Inspect `techs/go/return-errors.md`; make future edits there. The body file is only an authoring input.
If you omit `--body-file`, complete the generated draft and remove unused template prompts before validation.

#### Validate and share

```sh
code-rules library check
```

Exit 0 means the library passes input validation; it does not prove that application code follows its rules.
An undeclared license produces a warning. Confirm the publisher's license terms before sharing.
Use one license declaration for the library in `rule-library.json`, with the actual license text and any notice files.

Review the diff, commit the library, and publish a Git tag using the repository's release process.
Give consumers the repository address, tag or version constraint, and group IDs.
Consumers run `code-rules add source` in their project, then `code-rules sync` to import the selected guidance.

You may customize this README for the library. Re-running `code-rules library init` preserves an existing README.
