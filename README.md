# Code Rules

Shared engineering standards for the agents building your software.
Code Rules manages which versioned engineering rules a codebase adopts, including shared libraries, local exceptions, and reviewable updates.
It generates rule files for agents and other tools to consume. Your project chooses how to apply, validate, and enforce them through agent prompts or separate tooling. See [product scope](docs/src/content/docs/overview.md#scope-rule-management-and-delivery).

The project is in early implementation.
The offline Builds module and documentation site are available for development; the CLI is not implemented.

## Documentation

The [documentation site](docs/README.md) uses Astro and Starlight.
Start with [What is Code Rules?](docs/src/content/docs/overview.md) or [Project status](docs/src/content/docs/status.md).

```sh
bun install --frozen-lockfile
bun run docs:dev
```

To validate the implementation and documentation:

```sh
bun run check
```

The check includes comment-presence linting for authored TypeScript, JavaScript, and Astro files.
Use `@fileoverview` headers separated from declarations by a blank line; export TSDoc belongs immediately above its declaration.
For Astro components, the frontmatter overview describes the component's role and rendered result.
Private-helper documentation stays optional, and review checks comment accuracy and usefulness.

The public tool is independent of any particular rule library.
All examples in these docs are illustrative; this repository does not contain Fabrica's private rule corpus.

## Builds

[`buildRules`](src/builds/index.ts) accepts configuration, in-memory library snapshots, local rule files, and a tool version.
It returns the complete generated file set without fetching libraries or writing project files.
The [Builds scope note](_internal/builds.md) explains the interface, subsystem responsibilities, and verification.

Run the [manual Builds scenario](tests/manual/builds.ts) to create a temporary workspace with group and rule indexes, individual effective definitions, local rules, and provenance:

```sh
bun run builds:example
```

The script creates files for inspection and makes no automated assertions.
After running it:

1. Open the printed `generated/RULES.md` path and follow its testing-group link.
2. Confirm the group page includes exactly two complete active rules: the project retry budget and stopping retries after success.
3. Open `generated/groups/techs/typescript.md` to see the larger group as an index with explicit **Read full rule** links. Follow the testing rule links and confirm the retry-budget replacement retains ID `example:practices/testing/verify-retries` and links to its local definition.
4. Use the printed `scenarios.md` to inspect pre-implementation selection and a review with no test-file edits. The TypeScript scenario should lead to testing and code-design rules, while Go is unrelated.
5. Open `generated/libraries/example/README.md` for the source identity and revision. This unlicensed fixture has no generated license directory.
6. Inspect `generated/provenance.json`: the replacement should have a local origin and an imported upstream origin; the additional local rule should have no upstream origin.

Source repository names and commits are illustrative, so upstream GitHub links do not point to a real fixture library.
The temporary workspace remains available after the script exits.
Automated behavior checks live beside the implementation in `src/builds/build*.test.ts`, grouped by core behavior, selection, input validation, indexes, group delivery, Markdown, and licensing. Each suite tests through the public `buildRules` interface; shared fixture factories live in `build-test-fixtures.ts`.
