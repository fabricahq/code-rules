# Code Rules

Shared engineering standards for the agents building your software.
Code Rules manages which versioned engineering rules a codebase adopts, including shared libraries, local exceptions, and reviewable updates.
It generates rule files for agents and other tools to consume. Your project chooses how to apply, validate, and enforce them through agent prompts or separate tooling. See [product scope](docs/src/content/docs/overview.md#scope-rule-management-and-delivery).

The project is in early implementation.
Imports, offline Builds, sync, and the development CLI are available from this checkout. No package release is published.

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

Run the [manual Builds scenario](tests/manual/builds.ts) to create a temporary workspace with group and rule indexes, individual resolved definitions, local rules, and provenance:

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

## Imports

[`importLibraries`](src/imports/index.ts) accepts raw project configuration and optional cancellation.
It fetches exact Git commits, exact tags, or the highest tag matching an npm version constraint and returns each library's original bytes plus a text snapshot for Builds.
The operation never installs files in a consuming project.
The `sync` function coordinates Imports and Builds and safely applies their combined result.

Imports requires macOS or Linux and Git 2.30 or later.
Private libraries use your configured Git credentials; imports disable terminal prompts and do not print Git stderr.

```ts
import { importLibraries } from './src/imports';
import { buildRules } from './src/builds';

const libraries = await importLibraries(configuration);
const snapshots = Object.fromEntries(
  Object.entries(libraries).map(([name, library]) => [name, library.snapshot]),
);
const generated = buildRules({
  configuration,
  snapshots,
  localFiles: {},
  toolVersion: 'development',
});
```

Run the [manual Imports scenario](tests/manual/imports.ts):

```sh
bun run imports:example
```

The scenario creates two temporary Git libraries, imports them, and writes a temporary workspace for inspection.
Both libraries contain a rule with the same path; the generated testing group identifies each source separately.
Check image links against `vendor/` and license links against `generated/libraries/<source>/licenses/`, then inspect `generated/provenance.json` for resolved commits.
The scenario uses illustrative GitHub and nested GitLab addresses, routed to local repositories only in the test process.
Generated remote source links therefore do not resolve to those local fixture libraries.
The script removes its Git fixtures and leaves the printed workspace for inspection.
This manual API example does not use Sync's persisted records or safe file updates.

### Import limits and failures

Each library has a 120-second deadline, at most 10,000 tree entries, and an 8 MiB tree-listing limit.
Retained files may occupy up to 64 MiB in total, with an 8 MiB limit per file.
A shallow fetch may still download a large tree; retained-file limits do not bound network traffic or Git's temporary disk use.
Imports rejects symlinks, submodules, Git LFS pointers, reserved paths, and case-insensitive NFC-normalized path collisions in retained files.
Required text, declared licenses and notices, and Markdown inspected for dependencies must be UTF-8.

`ImportError.code` distinguishes invalid configuration or libraries, unavailable Git, inaccessible repositories, missing or refused refs, unmatched or ambiguous version constraints, tags changing during fetch, unsupported content, resource limits, cancellation, timeouts, Git failures, and I/O failures.
Failures return no partial library mapping.
The [import reference](docs/src/content/docs/reference/imports.md) describes file selection and preservation behavior.

Imported rule assets follow [two conventional locations](docs/src/content/docs/reference/files.mdx#supporting-assets): an adjacent `assets/<rule-name>/` directory and a shared library-root `assets/` directory.
Imports preserves complete owned directories and adds the shared directory when referenced. Supporting links outside these locations fail validation; declared library licenses keep their manifest-based paths.

Use an exact `ref` or an npm `version` constraint such as `^1.2.0`, never both. Version imports choose the highest matching complete SemVer tag and record its tag, normalized version, and commit.
Offline builds use the existing snapshot. See [version constraints](docs/src/content/docs/reference/configuration.md#semantic-version-constraints) for prerelease, alias, and repeatability behavior.

## Sync a project

Create `.code-rules/config.json` in a consuming project using the [configuration reference](docs/src/content/docs/reference/configuration.md).
Run the development CLI from this checkout, passing the configuration path:

```sh
bun src/cli.ts sync --config /path/to/project/.code-rules/config.json
bun src/cli.ts build --config /path/to/project/.code-rules/config.json
bun src/cli.ts check --config /path/to/project/.code-rules/config.json
```

Sync imports libraries, generates resolved rules, and applies changes safely. Build regenerates offline; check compares without writing.
Each prints added, changed, and removed file paths. Check exits 1 for stale output or invalid input; usage errors exit 2.
The API equivalents are `sync` in `src/sync.ts` and `buildProject` / `checkProject` in `src/project.ts`.
See [sync and recovery](docs/src/content/docs/reference/sync.md) for ownership, snapshot records, locking, and interruption behavior.
