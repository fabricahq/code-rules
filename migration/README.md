# Go migration

Build the Go candidate through one reviewed PR at a time. The TypeScript CLI remains available throughout migration.

Use the [Go conventions](../_internal/go-conventions.md) when implementing or reviewing a slice.

## Branches and baseline

- Integration branch: `go-migration`.
- Current slice: `codex/go-rule-documents`, based on `go-migration`. See [rule document boundaries](rule-documents.md) for scope and the interactive lab.
- [Group metadata](group-metadata.md) was human-approved and merged in PR #12 at `5c2c10a23cf556d9533bed316397338213f22fbd`.
- [The identity slice](identities.md) was human-approved and merged in PR #11 at `6bfcaf608bc5ce9c36af4c3c27c02d751a7fdd30`.
- Every migration PR targets `go-migration`. A human merges each slice before the next iteration starts.
- A separate human decision authorizes the eventual `go-migration` to `main` PR, distribution changes, and removal of TypeScript runtime code.

The initial integration branch and TypeScript reference both start at `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`.
That is [PR #9](https://github.com/fabricahq/code-rules/pull/9), stacked on [PR #8](https://github.com/fabricahq/code-rules/pull/8).
It includes library authoring, Node packaging, and contextual command help that `main` lacked when this iteration began.
PR #8 has since merged to `main`; PR #9 was closed without merging in favor of this migration. Its committed behavior remains in the pinned reference.
The user approved merging [the harness PR #10](https://github.com/fabricahq/code-rules/pull/10) into `go-migration`; its merge commit is `5ba1d51c7214c9d22979a882fb6fb46b2120d8fe`. The identity slice starts there.

The product checkout and private rule corpus had unrelated local changes. This work uses a separate clone and committed corpus contents.
The harness slice introduced no Go implementation. The merged identity slice added a native package and development adapter; the current slice adds rule document splitting. Release publication and scheduled work remain unauthorized. The user-approved validation diagnostic improvements are recorded in [the identity slice](identities.md#user-approved-diagnostic-improvement).

## Inventory and evidence

[contracts.json](contracts.json) is the proposed capability inventory. IDs are stable: append new scenario IDs without renumbering existing ones.
Dependencies express migration order. Each capability lists source owners and required happy, unhappy, and boundary cases.
The inventory retains `not-implemented` for full capabilities until acceptance. The identity and group metadata slices record partial implementation evidence separately; every full acceptance scenario remains pending until its required evidence is available and reviewed.

The initial [executable scenarios](../tests/migration/scenarios.ts) exercise representative paths across those capabilities.
The report lists these as `exercisedBy`; it does not count a representative case as full capability completion.
The original fixtures come from [the existing integration fixture](../tests/fixtures/import-fixture.ts) and [installed-package scenarios](../tests/package/installed.test.ts).

Use the harness from this checkout after installing dependencies:

```sh
bun install --frozen-lockfile
bun run build
bun tests/migration/compare.ts \
  --reference "$PWD/dist/cli.js" \
  --candidate "$PWD/dist/cli.js" \
  --reference-revision 7013d3d374a33a5cf65a2a48ff6870e46f9d7209 \
  --report /tmp/code-rules-migration-reference.json
```

To test a candidate, replace `--candidate` with its absolute executable path.
The runner never supplies missing candidate functionality through TypeScript.
Both executables must accept the CLI arguments directly; use an explicitly recorded launcher for an executable needing interpreter arguments.
Executable hashes identify the supplied files. They do not prove which source, template, or external dependencies produced those files.
The person building the reference must record that provenance alongside the report. The reference above uses the pinned sources, lockfile, and packaging script.

Exit 0 means every initial scenario passed parity and independent assertions. Exit 1 means a discrepancy or assertion failed.
Exit 2 means the harness could not establish its inputs or run, including a stale TypeScript baseline.
The report points to retained workspaces and per-scenario JSON containing raw streams, before/after entries, and discrepancies.
Delete the report and its `evidenceRoot` directory when review is complete.

## What comparisons mean

Each side gets separate projects, HOME, scratch space, and identical copies of a real temporary Git library.
Git routing uses a local repository, without modifying user Git configuration. Each invocation has closed stdin, bounded output, and a timeout.
The harness clears inherited application environment variables except PATH and sets locale, time zone, and Git routing explicitly.

Compare:

- Exact exit codes, signals, process errors, and text output.
- JSON stdout by values and field presence. Object key order and formatting are incidental; array order and missing-versus-null remain meaningful.
- Every path, entry kind, permission mode, symlink target, and exact file byte in each controlled root. Generated JSON also compares byte for byte.
- License and notice CRLF bytes, binary assets, source-qualified rule IDs, and versioned provenance.
- Independent expected status, intended values, collision protection, failure preservation, and unchanged trees for read-only steps.

Only equivalent workspace prefixes in stdout/stderr are replaced with `<workspace>`.
File contents receive no normalization. Timestamps, generated versions, commit IDs, and license bytes receive no normalization.
The fixture commits once and copies that Git history to both sides, so both observe identical commit IDs.
[approved-differences.json](approved-differences.json) starts empty; changing comparison policy or acceptance criteria requires explicit independent review.

Filesystem inspection detects persistent changes, including files under controlled HOME and scratch directories.
It is not an OS sandbox or a syscall audit: transient writes, reads, and activity outside those roots require additional safety tests.
The initial suite does not yet measure interactive prompts, cancellation, crash recovery, full npm-range/SPDX/Markdown edge cases, every CLI flag, or native distribution.
Those requirements remain in the inventory and the existing TypeScript suites continue to run.

## Prevent stale evidence

The runner rejects changes from the reference in `src/`, runtime/package metadata, the lockfile, packaging script, and reused fixture/package tests.
That conservative check includes staged, unstaged, committed, and untracked source changes.
Reports record source tree/blob IDs, corpus revision, contract hash, executable hashes, tool versions, harness commit, and whether the checkout was dirty.
Do not silently update the reference to get a passing report.
For a TypeScript change, propose the contract and reference update, add the regression case, and rerun affected comparisons and required repository checks.
For an already migrated capability, update both implementations or explicitly reopen its status.

## Validation and review

```sh
bun test _tools/go-migration.test.ts
bun run check
bun run test:package
```

The sensor tests run the real reference against itself, then reject wrong output, wrong exit status, and writes during help.
They also check array order, missing fields, binary/license bytes, and independent failure when both outputs agree incorrectly.
The CI check uses Node 24 and a complete Git history so it can inspect the pinned reference.
The release validation workflow also fetches that history before running repository checks.
The normal repository test command already includes `_tools/*.test.ts`; its format, lint, and typecheck commands cover the new files.

The harness review covers inventory completeness, baseline choice, comparison normalization, safety limits, and negative controls.
Passing these checks is implementation evidence. Independent validation and human approval remain separate states.
See [feedback.md](feedback.md) and [evidence.md](evidence.md) for the harness handoff.

The merged [identity slice](identities.md) implements group/rule IDs and selector validation.
The merged [group metadata slice](group-metadata.md) adds JSON metadata parsing. The current [document slice](rule-documents.md) separates frontmatter and body; its walkthrough shows only that new operation. The user-approved target uses one nonblank `whenToRead` string for groups and rules; the pinned TypeScript group format still uses arrays.
Review each slice before starting filesystem-writing capabilities.
