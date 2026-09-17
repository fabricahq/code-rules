# Native migration acceptance candidate

PR39 tests the composed CLI, reconciles implementation evidence, and supplies a
human pilot. It does not approve itself, publish a release, switch npm entrypoints,
merge the integration branch into main, or remove TypeScript.

## Run the candidate

```sh
go build -o /tmp/code-rules ./cmd/code-rules
/tmp/code-rules --help
/tmp/code-rules init
/tmp/code-rules local add group techs/go --name Go \
  --description 'Go guidance.' --when-to-read 'When editing Go.'
/tmp/code-rules local add rule techs/go/errors --title 'Return errors' \
  --impact HIGH --impact-description 'Preserve failures.' \
  --when-to-read 'When calling functions.' --body-file body.md
/tmp/code-rules build
/tmp/code-rules check
```

Run those commands in a disposable project with an authored `body.md`. Missing
metadata prompts only on a terminal. `--non-interactive` requires explicit inputs.
`add source` edits configuration without fetching. `sync` needs Git; `build` and
`check` use verified vendored bytes without Node, Bun, or Git. Exit 0 is success,
1 is an operation failure or stale check, and 2 is invalid usage. Operation JSON
goes to stdout; errors and prompts go to stderr.

## Complete CLI pilots

```sh
go test -race ./internal/acceptance
```

The four pilots execute the compiled command, using original fixture content and
real disposable Git commits/tags. Library authoring happens through the CLI.
Sync sees a PATH containing only Git. Offline commands see an empty PATH.

- **Lifecycle:** create and validate a licensed library, import an annotated release,
  override group guidance locally, build/check offline, repeat without byte changes,
  detect stale generated output without writing, and rebuild it.
- **New release:** add a newer tag, sync updated original rule bytes and provenance,
  then check offline.
- **Manual vendor edit:** alter a retained license; build fails and preserves every
  project file, including the deliberate edit.
- **Failed sync:** select a newer invalid library; the previous vendor and generated
  files remain byte-for-byte unchanged.

The PR39 walkthrough exposes the same commands, statuses, diagnostics, checked
invariants, and final project files. A pilot may pass because a deliberately bad
command correctly returned exit 1. The transcript always shows that error.
The pilot uses a sibling CLI build; PR38 separately tests installed archive binaries.

## Evidence inventory

[contracts.json](contracts.json) keeps the original capability and scenario IDs.
Implementation evidence points to actual Go tests and owning PRs. No scenario is
marked accepted merely because its test file exists. The full Go suite includes
fault injection and boundary cases that the human pilot deliberately does not repeat.

Run from the final stack:

```sh
go test -race ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
bun install --frozen-lockfile
bun run check
bun run test:package
```

The Go workflow also compares shared parser, Git selection, and Markdown fixtures
against the pinned TypeScript implementation. The original broad executable parity
harness and its negative-control tests remain unchanged. Exact broad parity is not
expected across approved format and presentation differences; a passing native
pilot does not replace that discrepancy report or prove TypeScript rollback.

## Candidate validation results

The composed candidate passes the full Go race suite, `go vet`, staticcheck, and
all four native CLI pilots. The original TypeScript `bun run check` also passes.

The unchanged broad executable parity harness ran all 27 steps with both binaries
reporting `0.1.0-rc.1`. It reports differences, not a pass. Its source-add fixture
uses an npm caret constraint, which the approved HashiCorp constraint contract
rejects. The subsequent licensed-sync, replacement-build, and failed-sync checks
therefore lack their required imported source. The native pilots above exercise
those safety boundaries with valid HashiCorp constraints. Other reported differences
include help/error presentation, metadata shapes, generated Markdown, and provenance.
Those differences still need acceptance review; they were not normalized away.

## User-approved migration differences

These decisions were made during walkthrough review and are implemented in their
owning PRs. They are explicit differences, not blanket normalization rules:

- Group and rule `whenToRead` are single nonblank strings; surrounding whitespace
  is trimmed. Unknown metadata fields are errors (#12, #14).
- Version constraints use native HashiCorp syntax rather than npm ranges (#17).
  `TagVersion` returns a descriptive error for invalid complete versions (#16).
- One optional library license replaces an array that held at most one item (#21-22).
- A rule owns one original document; parsed fields coexist with that source (#21).
- Cross-rule links are rejected to avoid links to omitted rules (#22).
- Local group definitions take precedence over imported metadata (#23).
- Small groups retain full rule text. Summary pagination uses 750 lines, natural
  numeric ordering, explicit page counts, navigation, and separated footers (#25).
- Generated JSON keeps literal `<` and `>` for readable version constraints (#26).

The newer SPDX list was accepted during license review. See the owning slice docs
and fixture-specific expectations for details. Global
[approved-differences.json](approved-differences.json) remains empty: the original
broad comparison harness is not weakened to hide mismatches.

## Release gates still open

1. Human review and merge approval for #33-#39, with final automated findings resolved.
2. Review the scenario evidence before marking full capabilities accepted. In
   particular, a TypeScript rollback cannot blindly consume changed Go metadata or
   HashiCorp constraints. Preserve the pre-migration project copy and executable;
   snapshot/journal compatibility is narrower than whole-project rollback.
3. Native runtime evidence on every supported architecture. CI executes its Linux
   and macOS host architectures; the other targets are cross-compiled, not executed.
4. Tool-license and publication decisions. Candidate archives are unpublished and
   the existing package still declares UNLICENSED. Approval of library terms does
   not approve the tool's own license.
5. A separately approved release plan for npm/native activation, main integration,
   and eventual TypeScript removal. Existing npm install/package tests keep the
   production entrypoint available during this review.

There are no additional implementation slices planned in this batch. These gates
are acceptance and release work, not permission for automatic cutover.
