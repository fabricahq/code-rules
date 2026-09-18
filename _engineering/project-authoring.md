# Project and library authoring

`internal/cli` collects explicit options or terminal input before invoking authoring. Prompts precede writer ownership. Rule creation requires an existing group before asking for rule metadata.

`internal/project` owns project initialization, the managed guide, local rules and groups, and source declarations. `internal/authoring` owns library initialization, library definitions, and complete library checks. Project operations use the selected configuration directory; library operations use an independent library root. Neither owner calls into the other.

Both use `internal/filetxn` for contained reads and protected publication. Its `Edit` operation acquires ownership, prepares changes against current input, publishes them, and reports post-commit cleanup warnings. Shared rule templates and format validation live in `internal/rules`.

Project init preserves valid configuration and local rules while refreshing an unmodified managed project guide. The guide carries a body digest to distinguish older generated text from manual edits; this is an ownership check, not authentication. Custom configurations outside a `.code-rules` directory use `CODE_RULES.md` to preserve the project's general README.

Library and group READMEs explain authoring and remain user-owned after creation. Group-root READMEs are excluded from rule loading. The templates link to the canonical [rule authoring rubric](../docs/src/content/docs/reference/rule-authoring.md).

Test command behavior through `internal/cli`, including real terminals and generated guide examples. Test project status and authoring through their owning package operations. Storage tests exercise publication refusal, rollback, and interrupted-write recovery. See [Go conventions](go-conventions.md) and the [CLI reference](../docs/src/content/docs/reference/cli.md).
