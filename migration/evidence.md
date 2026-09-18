# First iteration evidence

Historical harness evidence. PR #10 was subsequently merged into `go-migration` with human approval at `5ba1d51c7214c9d22979a882fb6fb46b2120d8fe`. See [identities.md](identities.md) for the current slice.

## Revisions and scope

- TypeScript reference and integration base: `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`.
- Integration branch: `go-migration`.
- PR head branch: `codex/go-migration-harness`.
- Rule corpus: `e2166f90333157fd3e14c24d3e43287ece858e4b` in `josh-padnick/code-rules`.
- Exact submitted head: the PR head SHA; the retained comparison report records that SHA after commit.

This PR adds the inventory and executable comparison sensor. It implements no Go capability.
The inventory covers formats, resolution, rendering, safe files, snapshots, CLI discovery, project/library authoring, offline commands, imports, sync, and distribution.
Every capability remains pending implementation and independent acceptance.

## Reproduce

Run the commands in [README.md](README.md#validation-and-review).
The default `bun run check` includes sensor tests through `_tools/go-migration.test.ts`.
The comparison command in the README produces a JSON report and retained per-scenario artifacts with the actual executable hashes and environment versions.

Local evidence for the implementation run is retained outside the public repository:

- `/private/tmp/code-rules-migration-evidence/reference-self.json`: exact-head reference self-comparison.
- `/private/tmp/code-rules-migration-evidence/sensor-tests.log`: positive, negative, and incomplete-candidate tests.
- `/private/tmp/code-rules-migration-evidence/check.log`: complete repository validation.
- `/private/tmp/code-rules-migration-evidence/package-tests.log`: installed-package validation.

These paths belong to the originating machine. Other reviewers should regenerate their own evidence; the report links its temporary workspaces.

## Check results

- `bun run check`: passed, including 444 tests, formatting, lint, TypeScript checks, Astro checks/build, and links across 25 pages.
- `npm_config_cache=/private/tmp/code-rules-migration-npm-cache bun run test:package`: passed, all 6 installed-package tests. The temporary cache avoids sandbox restrictions on the user cache.
- The 11 sensor tests pass within that gate. They include reference parity and deliberately incorrect candidates.
- The inventory proposes 14 capabilities and 64 required scenario groups; the initial runner executes 27 steps per executable.
- Native Go distribution, complete capability acceptance, independent validation, and human approval remain pending.
- Oracle advisory review was unavailable because the Oracle CLI was not installed in this environment.

## Review decisions

The original baseline included two then-unmerged TypeScript PRs. PR #8 has since merged to `main`; PR #9 was closed in favor of the Go migration, with its committed behavior retained in the reference.
Review the initial inventory, uncovered required scenarios, normalization policy, and filesystem-observation limits.
No behavior differences are approved. No production behavior or existing acceptance test changed.
The draft assertion corrections are documented in [feedback.md](feedback.md#draft-assertion-corrections).

After human merge and fresh measurement, consider the group/rule identity and selector portion of `formats.identities`.
The next iteration must state its narrower contract IDs, executable adapter, boundaries, and relevant rules before adding Go.
