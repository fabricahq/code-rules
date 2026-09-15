# Project setup and authoring

Implements phase four of the launch plan, stacked on the local-group resolution PR.

## Commands

- `init [--config path]`: create an empty sources configuration and local README, preserving existing valid configuration and README. No source or Git repository is required.
- `local add group <group-id>`: collect name, description, and one or more when-to-read cues, then create local metadata.
- `local add rule <rule-id>`: collect selection and impact metadata, then write a draft from the shared template or accept an explicit Markdown body file. Offer missing-group creation interactively; scripts opt in explicitly and provide its metadata.
- `add source <alias>`: record an explicit repository, exactly one ref or version constraint, and group selection. Preserve existing sources, exceptions, and local bytes. Run sync separately to fetch.

Missing inputs prompt only on a terminal. Noninteractive use requires equivalent flags and reports missing inputs without writing. Help and usage errors do not mutate files. Duplicate flags and unknown options fail explicitly.

## Ownership and implementation

`src/authoring/` owns pure template rendering and project authoring operations. File updates use the existing writer lock, reject links and case collisions, validate all proposed inputs before publication, publish new files exclusively, and claim configuration before checking original bytes and publish its replacement exclusively. Retain uncertain claims for manual recovery. Propagate cancellation through every authoring operation. Failed operations roll back newly published files without deleting outside edits. Interactive collection occurs before taking the writer lock; operations validate again under the lock.

The canonical rubric remains the document for authors. Its template and the distributable Markdown template are checked together to prevent drift. Drafts require human or agent completion before adoption; the command does not invent policy. Metadata parsing reuses current format contracts. Library setup, publishing, enforcement, and automatic editing of AGENTS.md remain separate.

## Verification

Drive the real CLI from an empty directory through init, local group/rule creation, offline build/check, source addition and sync against a real fixture Git library. Cover custom config paths, imported groups, missing inputs, collisions, invalid metadata, symlinks, repeat init, preserved exceptions, and failed writes. Exercise interactive prompts and cancellation independently with a pseudo-terminal. Run the full repository check before opening the PR.
