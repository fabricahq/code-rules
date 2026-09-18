# Project and library authoring

`internal/cli` collects explicit options or terminal input before invoking authoring. Prompts precede writer ownership. Rule creation requires an existing group before asking for rule metadata.

`internal/authoring` owns scaffolding, validation, collision checks, and publication. Project operations use the selected configuration directory; library operations use an independent library root. Keep these ownership models separate.

Project init preserves valid configuration and local rules while refreshing an unmodified managed project guide. The guide carries a body digest to distinguish older generated text from manual edits; this is an ownership check, not authentication. Custom configurations outside a `.code-rules` directory use `CODE_RULES.md` to preserve the project's general README.

Library and group READMEs explain authoring and remain user-owned after creation. Group-root READMEs are excluded from rule loading. The templates link to the canonical [rule authoring rubric](../docs/src/content/docs/reference/rule-authoring.md).

Test command behavior through `internal/cli`, including real terminals and generated guide examples. Test publication refusal and rollback through authoring operations. See [Go conventions](go-conventions.md) and the [CLI reference](../docs/src/content/docs/reference/cli.md).
