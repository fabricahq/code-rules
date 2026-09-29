# Project and library authoring

`internal/cli` collects explicit options or terminal input before invoking authoring. Prompts precede writer ownership. Rule creation requires an existing group before asking for rule metadata.

`internal/project` owns project initialization, the managed guide, local rules and groups, and source declarations. `internal/library` owns library initialization, library definitions, complete library checks, library releases, and catalog loading. Project operations use the selected configuration directory; library operations use an independent library root. Project operations load adopted catalogs through `library`; library operations are independent of consuming projects.

`library.Initialize`, `AddGroup`, `AddRule`, and `Check` operate on the library root. `Load` and `LoadSource` validate selected catalogs for consumers; full-library checks also validate unused content. Inventory capture and its filesystem adapter remain private.

`Check` also compares the working tree with the latest `release/<number>` tag reachable from `HEAD`. It reads the library's own repository through `internal/gitexec`'s owned mode, which honors the author's Git configuration, and compares content as Git would store it, so line-ending conversion is not a change. Change notes in `changes/` are read separately from the inventory, so catalogs and snapshots never include them. A library whose root has no `.git` has no library releases.

`PlanChange` checks a change note request against the working tree and that library release before prompts collect a missing change level or summary; its `Commit` revalidates under writer ownership and creates a new, uniquely named note, never editing an existing one.

`Release` changes no files in the library. It checks the GitHub CLI and the tagger identity, then fetches the upstream branch and missing release tags, and refuses when the branch isn't the remote's default branch or differs from it, when a release tag differs from the remote's, when library check fails, or when the files check read differ from `HEAD`, so the tag publishes exactly what check validated. The notes and the tag message's record are rendered by pure functions from the release record, and the message must parse back unchanged before tagging. It creates the annotated tag with `git tag`, honoring the author's identity and signing configuration, and pushes only that tag; a failed push deletes the local tag, so the clone never keeps a release tag the remote lacks. A release tag already on `HEAD` means an earlier run stopped, so it creates only what's missing. GitHub Release pages go through `gh`; tests use `internal/test/ghfixture` and the pushable `gitfixture` remote.

Both use `internal/filetxn` for contained reads and protected publication. Its `Edit` operation acquires ownership, prepares changes against current input, publishes them, and reports post-commit cleanup warnings. Shared rule templates and format validation live in `internal/rules`.

Project init preserves valid configuration and local rules while refreshing an unmodified managed project guide. The guide carries a body digest to distinguish older generated text from manual edits; this is an ownership check, not authentication. Project commands use `.code-rules/config.yaml` from the project root. The managed guide lives at `.code-rules/README.md`; the project's root README is outside its ownership.

`library.Initialize` also writes the check workflow, `.github/workflows/code-rules.yml`, from the embedded `check-workflow.yml`, pinned to the running Code Rules version, or to the latest release for a development build. Renovate updates its action pins. A test keeps it equal to the workflow in the version rules guide outside its install step, so update both together; another runs the install step against fake release tools to prove it installs only verified releases.

Library and group READMEs explain authoring and remain user-owned after creation. Group-root READMEs are excluded from rule loading. The templates link to the canonical [rule authoring rubric](../docs/src/content/docs/reference/rule-authoring.md).

Test command behavior through `internal/cli`, including real terminals and generated guide examples. Test project status and authoring through their owning package operations. Storage tests exercise publication refusal, rollback, and interrupted-write recovery. See [Go conventions](go-conventions.md) and the [CLI reference](../docs/src/content/docs/reference/cli.md).
