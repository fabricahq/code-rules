# Code Rules

Shared engineering standards for the agents building your software.
Code Rules defines a convention and importer for versioned rule libraries, project exceptions, and generated rules used during planning, implementation, and review.

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

Run the original example to create a temporary workspace with an index, an aggregate, local rules, and provenance:

```sh
bun run builds:example
```

Inspect the printed `generated/RULES.md` path, then its linked testing group.
The example replaces an upstream rule while retaining its ID and adds a separate local rule.
Source repository names and commits are illustrative.
