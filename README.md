# Code Rules

Shared engineering standards for the agents building your software.
Code Rules defines a convention and importer for versioned rule libraries, project exceptions, and generated rules used during planning, implementation, and review.

The project is in design.
This repository contains a working documentation site describing the proposed first release; the CLI is not implemented.

## Documentation

The [documentation site](docs/README.md) uses Astro and Starlight.
Start with [What is Code Rules?](docs/src/content/docs/overview.md) or [Project status](docs/src/content/docs/status.md).

```sh
bun install --frozen-lockfile
bun run docs:dev
```

To validate the site:

```sh
bun run check
```

The public tool is independent of any particular rule library.
All examples in these docs are illustrative; this repository does not contain Fabrica's private rule corpus.
