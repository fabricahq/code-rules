# Code Rules

[Fabrica Code Rules](https://code-rules.fabricahq.com) is the package manager for your engineering rules. Author project rules, adopt versioned Git libraries, and generate Markdown that agents can read before they work.

The Go CLI is available. The installable authoring skill remains separate work.

## Installation

See [Install Code Rules](docs/src/content/docs/start-here/install.md) for Homebrew, the standalone installer, and manual downloads, including current availability. These methods use native executables and require no Go, Node.js, or Bun. Maintainers can find channel activation and tests in [installation distribution](_distribution/README.md).

## Build the CLI

Install the Go version declared in [go.mod](go.mod). Git is required to sync remote libraries.

```sh
go build -o ./dist/code-rules ./cmd/code-rules
./dist/code-rules --help
```

The executable runs without Node.js or Bun. From a consuming project's root, run `code-rules project init`, create a local group and rule, then `code-rules project build`. To adopt a library, use `code-rules project add library` with its repository, revision, and groups, then run `code-rules project sync`.

Read [project setup](docs/src/content/docs/start-here/set-up-project.md) and [the CLI reference](docs/src/content/docs/reference/cli.md) for the complete workflow. Human output is the default; `--json` returns structured responses and disables prompts. `project check` reports status and problems without writing files.

## Validate changes

```sh
gofmt -w cmd internal
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
go test -race ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 -test ./...
go build ./cmd/code-rules ./cmd/package-binaries ./cmd/plan-release ./cmd/publish-release
```

Tests exercise parsers, filesystem safety, Git imports, real CLI processes, generated agent instructions, and installation/upgrade/rollback. Parser regression fixtures live beside their Go tests. [Go conventions](_engineering/go-conventions.md) cover error ownership and comments.

[Security practices](_engineering/security-practices.md) define dependency pins, Renovate updates, vulnerability scans, and review requirements.

## Documentation website

The Astro website and its JavaScript tooling are independent of the Go executable. Install the Bun version declared in [package.json](package.json), then run:

```sh
bun install --frozen-lockfile
bun run check
bun run docs:dev
```

`bun run check` validates website/tooling formatting, lint, types, tests, the Astro build, and rendered links. It does not replace Go validation. See [docs/README.md](docs/README.md) for site development and the [CSS and Tailwind guidelines](docs/_internal/css.md) for styling conventions.

TypeScript stays on 6.0.3 because the current Astro checker and ESLint parser require its compiler API. [TypeScript 7 does not yet provide that API](https://devblogs.microsoft.com/typescript/announcing-typescript-7-0/#running-side-by-side-with-typescript-6-0); revisit this pin when both tools support it.

## Package binaries

[Release instructions](_engineering/releasing.md) explain candidate archives and PR download links. Packaging builds committed source and takes an explicit release version. Candidate versions default to a source commit identifier. A release PR supplies editable notes and the version in `releases/v<version>.md`; merging it starts testing, packaging, and publication. Assets are verified on a draft before publication. The first release is `v0.1.0`; the tool uses the [MIT license](LICENSE.md).

## Implementation map

- `internal/rules`: validated configuration, identities, documents, links, and versions.
- `internal/library`: initialize, author, check, and load rule libraries.
- `internal/imports`: import complete libraries with verified Git provenance.
- `internal/build`: generate complete output from validated inputs through `Generate`.
- `internal/project`: initialize and author consuming projects, sync libraries, build offline, and check complete project freshness.
- `internal/filetxn`: bounded filesystem reads, writer ownership, safe publication, and recovery shared by both owners.
- `internal/cli`: collect inputs and present operation results.
- `internal/distribution`: package and verify executable archives.
- `internal/release`: validate release requests and publish verified assets through the GitHub API.
- `internal/test/acceptance`: exercise complete CLI workflows through real processes.
- `internal/test/gitfixture`: supply disposable Git repositories for tests.
- `internal/test/terminalfixture`: exercise interactive CLI prompts in isolated terminals.

`internal/test/` groups these three independent packages; it contains no Go package of its own. Unit tests remain beside the code they test.

Engineering policies belong to independently owned libraries. This repository supplies the formats, tools, and public authoring guidance.
