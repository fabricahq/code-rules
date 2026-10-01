# Implement per-rule versioning in Code Rules

You're implementing per-rule versioning in the `code-rules` CLI (`fabricahq/code-rules`), on the branch `claude/rule-versioning-handoff-d5bbf3`. The user-facing docs on this branch are already written and reviewed by Josh, the maintainer. **They are the specification.** This document tells you where to find it, what's settled, how the current code is laid out, and the order to build in.

This file lives on the branch only while the work is in progress. Delete it in the final commit before the branch merges.

## Ground rules

- Read `AGENTS.md`, `CONTRIBUTING.md`, `_engineering/go-conventions.md`, and `_engineering/rules/comment-role-result-and-constraints.md` before writing code. Also read the Fabrica public rules this repository imports in `.code-rules/generated/RULES.md` (added by [fabricahq/code-rules#69](https://github.com/fabricahq/code-rules/pull/69); until it merges, read them on its branch, `claude/dogfood-public-rules`), especially `techs/go`, `practices/testing`, and `practices/code-design`.
- The docs describe the finished behavior. When code and docs disagree, the docs win, unless implementing them exposes a real problem. Then stop and raise it with Josh, and change the docs and code together once he decides. Never quietly implement something different from what the docs say.
- Don't use the em dash character anywhere. Don't add a Co-Authored-By or agent line to commit messages. Make atomic commits that each build and pass tests.
- Write full command names in docs and messages: `code-rules library release`, never `library release` or a bare flag. Say "library release" or "GitHub Release page", never a bare "release".
- Don't tag, publish, or merge. Don't open a pull request until Josh asks. The docs and code merge together, in one pull request, when the whole feature works.
- This is a breaking change to a released CLI (`v0.1.0`). It ships as `v0.2.0`, following `.release-planner/policy.md`. Code Rules has no users yet, so spend no effort on migration: no migration messages or code paths for old configuration fields, `_source.json` format version 1, or libraries without `release/<number>` tags. Old fields fail as unknown fields and old records fail as unsupported formats. The release notes give the manual steps (edit the configuration, delete `vendor/`, and sync) and say that the sync imports every unpinned rule's newest version.
- Prefer quality, simplicity, and long-term maintainability over development cost. Avoid gold plating: build what the docs describe, nothing more.

## The specification

Everything is under `docs/src/content/docs/`. Read these first, in this order:

| Page | What it specifies |
| --- | --- |
| `reference/rule-versions.md` | What a version covers, library releases as `release/<number>` tags, the release record YAML in the tag message, GitHub Release pages, change levels, the change note format in `changes/`, retired rules. |
| `guides/version-rules.md` | The library author's workflow end to end, including the generated GitHub Release page, CI, and recovery. |
| `reference/cli.md` | Every command contract: `project add library`, `project add rule --from` (fork), `project sync`, `project update`, `library init`, `library add rule`, `library change`, `library check`, `library release`. |
| `reference/configuration.md` | Project config: `groups`, `rules`, `pins`, `ref`, `exclude` with `reason`, `replacedBy`, and `basedOn`; rejected fields; how versions are chosen; where each rule's files come from. |
| `guides/update.md` | The update preview, confirmation, pins, scoped updates, retirements, `ref`. |
| `reference/provenance.md` | `_source.json` format version 2 and `generated/provenance.json` fields, and what offline checks verify. |
| `reference/imports.md` | Active-rule resolution, how rule versions are resolved from release tags, validation, limits, error codes. |
| `reference/sync.md`, `reference/files.mdx`, `reference/library-format.mdx` | Sync behavior and records, project file layout, library layout including `changes/`. |
| `guides/select-rules.md`, `for-agents/index.md` | Selection and exceptions; what agents are told about pins and versions. |

To see everything that changed from `main`, run `git diff main -- docs/`.

## Settled design, in brief

This summary is for orientation. The docs are authoritative.

**Library side**

- Every rule has its own semantic version (`major.minor.patch`, no prerelease or build). A version covers the rule's Markdown file and its `assets/<rule-name>/` directory only. Everything else (group metadata, shared `assets/`, `rule-library.yaml`, license and notice files) is a library-wide file: unversioned, needing no change note.
- Change levels are defined by compliance: major means work that complied with the previous version could fail this one; minor means it still complies and the version adds guidance; patch means it still complies and adds no guidance.
- After the first library release, every rule change needs a change note: a uniquely named YAML file in the library-root `changes/` directory with `summary` and a `rules` map from rule ID to `major`, `minor`, `patch`, `new`, `retired`, or `{change: retired, replacedBy: ID}`. Notes are never deleted. Pending notes are the ones added since the latest `release/<number>` tag. Several pending notes on one rule: the largest change wins and every summary is listed.
- A library release is one annotated tag, `release/<number>`, on the current default-branch commit. The message is Markdown release notes, a line containing only `---`, then a YAML release record: `formatVersion`, `release`, `rules` (every current rule's version), `changes`, `retired`, `libraryFiles`. Readers ignore unknown record fields, and `formatVersion` changes only for incompatible changes. Publishing creates no commit and changes no files. The first library release needs no notes and gives every rule `1.0.0`.
- `code-rules library release` fetches, requires the remote default branch with no local or remote difference, requires `code-rules library check` to pass, creates and pushes the tag, then creates the GitHub Release page with `gh`. Rerunning finishes a partial run. `--dry-run` and `--no-github-release` exist. There is no release pull request, no `--pr`, no `--publish`, no `--init`, no `--bump-all`.
- `code-rules library change ID...` writes a new note (`--bump`, `--summary`, `--retire`, `--replaced-by`). `code-rules library check` enforces notes against changes since the latest library release and previews the pending library release. `code-rules library init` also writes a check-only GitHub Actions workflow.
- Retired rule IDs are never reused. A rename is one note that retires the old ID with `replacedBy` and adds the new ID as `new`.

**Project side**

- Per source: `repository`, `groups` (IDs or `"*"`, `"practices/*"`, `"techs/*"`), `rules` (individual rule IDs, unioned with groups), `pins` (rule ID to `{version, reason}`, both required), `ref` (a tag or full commit SHA; not with `pins`), `exclude` (rule ID to `{reason, replacedBy?, basedOn?}`, `basedOn` only with `replacedBy`). `version` and `replace` are rejected as unknown fields.
- `_source.json` (format version 2) is the lockfile: each rule's version, the library release that published it, and that release's commit. Every file is stored at its library path, because a snapshot holds one version of each rule. Library-wide files come from one recorded library release, which `code-rules project update` moves to the newest, or from `ref`.
- `code-rules project sync` restores recorded versions. It chooses versions only for new sources, changed repositories, newly selected rules, added or changed pins, and added, changed, or removed `ref`.
- `code-rules project update [SOURCE | SOURCE:RULE ...]` previews `major`, `minor`, `patch`, `new`, `retired`, `replaced`, and `pinned` rows, then applies after confirmation. In a terminal it asks per major change and retirement whether to keep the current version (writes a pin with a reason) and per new rule whether to add or exclude it (writes an exclusion). Without a terminal, or with `--json`, it only previews unless `--yes`. `--keep`, `--exclude`, and `--reason` do the same non-interactively.
- `code-rules project add rule ID --from LIBRARY@VERSION` forks one rule version into `local/`, copying linked shared assets, and writes an exclusion with the fork as `replacedBy` and the forked version as `basedOn`. The update preview's `replaced` row lists the library's changes after `basedOn`; `--incorporated SOURCE:RULE`, or the terminal answer, advances `basedOn` to the newest version, and nothing advances it automatically.
- Generated guidance shows each imported rule's version. Stale `exclude` or `rules` entries for retired rules warn; unknown IDs fail.

**Rejected, don't build:** library versions or version ranges, per-rule constraints, `hold` or `track` modes, `updates: manual`, an `include` list, branch refs, a manifest file on the default branch, per-rule tags, release pull requests, approval environments, draft releases.

## How the code works today

The Go code still implements the old model: one revision per source, chosen by `ref` or a `version` constraint, plus `exclude` and `replace`. Line numbers are approximate, as of commit `d366227`.

**Configuration** (`internal/rules/`)

- `configuration.go`: `Source` (:20) has `Ref`, `Version`, `Exclude map[string]string`, and `Replace map[string]Replacement`. `ParseConfiguration` (:43) rejects the former `localGroups` field at :48. `parseSource` (:75) lists allowed fields (:84), enforces `ref` XOR `version` (:99), requires both `exclude` and `replace`, and rejects a rule both excluded and replaced (:163).
- `authored_yaml.go` and `json_fields.go` provide the strict YAML handling (single document, no anchors, aliases, tags, or duplicate keys; unknown fields rejected). Reuse them for change notes and release records.
- `configuration_yaml.go`: `AppendConfigurationSource` (:24) appends a source while preserving comments. Nothing edits an existing source, which `code-rules project update --keep` and `--exclude` need.
- `refs.go`: `ParseGitRef` (:38) already accepts a full SHA or an exact tag (`release/4` parses as a tag) and rejects branches and short SHAs. `version_constraint.go` and `version_selection.go` implement semver constraints over `ls-remote --tags`; the constraint path becomes dead, but `tagRecords` (:103), the 20,000-tag limit, and the `^{}` peel checks are reusable for listing `release/*` tags.
- Fixtures: `internal/rules/testdata/<parser>/cases.json`, run by table tests. There are no golden files.

**Fetching** (`internal/imports/`)

- `revision.go` `fetchRevision` (:55): Git 2.30 check, temporary bare repository, `git fetch --depth=1 --no-tags` of exactly one ref (:145), moved-tag check (:159).
- `tree.go`: reads files from Git objects without a checkout (`ls-tree`, `cat-file --batch`), with the limits documented in `reference/imports.md`.
- `libraries.go` `ImportLibraries` (:25): all-or-nothing across sources.
- `process.go`: the hardened, unexported `gitRunner` (hooks disabled, protocol allow-list, output budgets, process-group kill) and `imports.Error{Code, Problem, Cause}`.

**Snapshots and offline checks** (`internal/project/`, `internal/library/snapshot.go`)

- `snapshots.go`: `sourceRecord` (:24) is `_source.json` format version 1. `encodeSnapshots` (:39) and `decodeSnapshots` (:86) write and verify it. `parseSourceRecord` (:128) re-validates identity by synthesizing a config that hard-codes `exclude: {}` and `replace: {}` (:155), which breaks as soon as `replace` is rejected. `matchSnapshotSource` (:197) checks the constraint again.
- `project.go`: `prepareProject` (:180) loads one tree per source (`library.Load`) and `verifyLoadedSnapshot` (:268) compares it with the inventory. Rules from several library releases need a new loading path.

**Resolution, provenance, and rendering** (`internal/build/`)

- `resolve.go` `resolve` (:79): exclusion and replacement targets must exist (:161, :166); replacement file reuse (:192) and same-group (:195) errors; `Upstream` and `Reason` for provenance.
- `output.go`: `libraryReadme` (:73), `provenanceSource` (:108), `provenanceRule` (:130), `provenanceOrigin` (:212). `build.Library{Catalog, Commit, Tag}` (`build.go` :11) assumes one commit per source.
- `render.go` `renderRule` (:57, header lines at :79) and `indexes.go` (rule summary entries at :151) are where rule versions appear in generated guidance.

**CLI** (`internal/cli/`)

- `cli.go` `Run` (:30), persistent `--json`. `project_commands.go` builds sync, build, and check in one loop (:18); `projectLibraryCommand` (:82) has `--repository`, `--ref`, `--groups`. `library_commands.go` has init, check, add group, add rule; there's no `change`, `release`, or `project update`.
- Prompts: `prompts.go` `interactive()` (:18) and `ask` (:27); `authoring_inputs.go` collects flags and prompts. `project_library_selection.go` `collectSource` (:12) requires `ref` today. `project_library_guidance.go` describes version ranges.
- Output: `output.go` `response` (:28), `responseError` (:35), `classifyError` (:101). `classifyError` surfaces `code` only for `*filetxn.Error`; `*imports.Error` codes such as the documented `releases-not-found` and `version-not-found` don't reach JSON yet.

**Library** (`internal/library/`)

- `check.go` `Check` (:29) runs no Git. `validateLibraryInventory` (:71) rejects non-rule files in groups (:110).
- `inventory.go` `readInventory` (:57) walks only `assets/`, `practices/`, and `techs/` (:104), so a root `changes/` directory is invisible today: good for catalogs and snapshots, but `code-rules library check` must now read it.
- `catalog.go` `rulePaths` (:320) and `rules.GroupFromPath` (`internal/rules/groups.go` :42) define what counts as a rule.
- `authoring.go` `Initialize` (:120) writes `rule-library.yaml`, license files, and `README.md` from `library-guide.md`. No workflow yet.
- `rules.ParseLibraryLicense` (`internal/rules/licenses.go` :25) is the only manifest parser; `formatVersion` must be `1`.

**Transactions** (`internal/filetxn/`)

- `writer.go` `WithWriter` (:79) and `Writer.Apply` (:153) replace `vendor/`, `generated/`, and the managed guide atomically, with a journal and recovery. `Apply` doesn't accept `config.yaml`. `publication.go` `Edit` (:396) publishes authored files separately.
- `internal/project/sync.go` `Sync` (:16) never reads the existing `_source.json`; it imports purely from configuration (:35).

**Managed guides**

- `internal/project/project-guide.md` (embedded, digest-stamped, refreshed by init, build, and sync; examples executed by `internal/cli/project_guide_test.go` :63) still describes `ref` and version constraints (:20, :65-83).
- `internal/library/library-guide.md` (written once by `code-rules library init`, never refreshed; examples executed by `internal/cli/library_test.go` :49) says to publish a Git tag.

**Tests**

- `internal/test/gitfixture/fixture.go` builds a real repository with `v1.0.0` and `v1.2.0` tags, served over an SSH shim that runs only `git upload-pack`: fetches work, pushes don't.
- `internal/test/acceptance/` runs end-to-end scenarios through the built binary (`scenarios_test.go` `Run` :41). The consumer scenario uses `--ref ">= 1.0.0, < 2.0.0"` (:165).
- `internal/test/terminalfixture/run.go` drives interactive prompts in a pseudo-terminal.
- `_tools/fake-github-cli.ts` fakes `gh` for the TypeScript tests only.

## Implementation decisions

These follow from the docs and the current code. Items marked **Decided** were settled with Josh; the rest are recommendations to raise with him if implementation shows a problem.

1. **Share one Git runner.** Move the hardened runner out of `internal/imports/process.go` into its own internal package, used by imports and by the new library-history code. Update `_engineering/sync.md`, which says Git stays private to imports (and mentions a `HasLocalRuleGroup` that no longer exists).
2. **Git in the library author's own repository.** Decided: `code-rules library check` and `code-rules library release` run Git in the author's repository with the author's Git configuration, credentials, and hooks, so a `pre-push` hook runs as it would for `git push`. Prompts stay disabled and output budgets still apply. Fetching libraries into projects keeps the hardened, hook-free configuration, because that repository isn't the user's.
3. **Listing and reading library releases.** List `refs/tags/release/*` with `git ls-remote`, applying the existing listing limits and peel checks. Fetch the release tags that are needed with `--depth=1 --filter=blob:none`, read tag messages with `git cat-file tag`, and read only the blobs of imported files. Parse the YAML part of each message with a strict parser in `internal/rules`, with fixtures in `testdata/release-records/`. Servers that ignore the filter still work, just with more data. Enable `uploadpack.allowFilter` in the fixture's shim.
4. **Finding a rule version's files.** A rule's version `V` was published by the library release whose record has a `changes` entry setting it to `V`. In the first library release every rule is `new` at `1.0.0`, so its record lists every rule under `changes`. Read the rule's files at that tag's commit.
5. **Snapshots with several library releases.** `_source.json` format version 2, exactly as in `reference/provenance.md`. Replace the one-tree-per-source loading in `prepareProject` with a loader that assembles a catalog from the main tree plus each rule's files from its own tree. Fix `parseSourceRecord` so it no longer synthesizes a configuration with `replace`.
6. **Sync reads the lockfile.** `Sync` must read the existing `_source.json` and apply the table in `reference/cli.md#project-sync`, fetching only what it needs. Configurations with `version` or `replace` fail before any fetch, as unknown fields.
7. **No migration.** Decided: Code Rules has no users yet. A format 1 `_source.json` fails like any unsupported record, and the fix is to delete `vendor/` and sync. Don't add code to read or convert old records.
8. **Update writes configuration and output together.** Decided: `code-rules project update --keep`, `--exclude`, and the interactive answers write pins and exclusions to `config.yaml` in the same transaction that replaces `vendor/` and `generated/`. Extend the `filetxn` journal so one transaction can also replace `config.yaml`, preserving comments through a new source-editing function beside `AppendConfigurationSource`. Recovery must then restore or finish all three together. Collect every prompt answer before taking the writer lock, per `_engineering/project-authoring.md`.
9. **Library check reads `changes/`.** Keep `changes/` out of catalogs and snapshots, as today. Check reads it separately: pending notes are files under `changes/` that don't exist at the latest `release/<number>` tag reachable from `HEAD`; a note that exists at that tag with different content produces the "edited published note" warning. "Changed since the latest library release" compares the working tree's rule file and asset directory with the tag's tree, so check works before committing. A shallow clone fails with instructions (`git rev-parse --is-shallow-repository`).
10. **Library release.** Implement as documented: fetch the upstream, require the remote default branch with no ahead or behind commits, run check, compute versions, render the notes and record, create the annotated tag, push only that tag, then create the GitHub Release page with `gh release create release/<n> --verify-tag --title ... --notes-file -`. On a rerun, find an existing `release/<n>` tag on `HEAD` and create only what's missing (`gh release view` to detect the page). The release number is one more than the highest `release/<n>` tag on the remote; a rejected tag push means someone else published first, so report it and stop.
11. **Rendering release notes.** A pure function from the pending notes and the release record to Markdown, matching the example in `guides/version-rules.md` exactly, including section order, the opening count line, the major-section sentence, and the collapsed version table. Test it with inline expected output. The same function renders `--dry-run` output.
12. **Error codes in JSON.** Make `classifyError` surface codes from `imports.Error` (and any new library-release error type) so `releases-not-found`, `version-not-found`, and new codes appear in `--json` output as documented.
13. **The check workflow.** `code-rules library init` writes `.github/workflows/code-rules.yml`, matching `guides/version-rules.md#check-changes-in-ci`, with the verified install step from the closed [fabricahq/code-rules#67](https://github.com/fabricahq/code-rules/pull/67) (`gh pr diff 67`): verify the `SHA256SUMS` attestation, check the archive, and install the Code Rules version running `init`. Pin actions by SHA with a version comment, as the docs show.
14. **Exit status of a preview-only update.** Decide and document: recommendation is exit `0` for a preview without `--yes`, with the preview in `value` for `--json`, since "updates are available" isn't an error. Add it to `reference/cli.md` when you implement it.

## Implementation order

Build in slices. Each slice ends with passing validation, atomic commits, and a short summary to Josh, so he can review before the next one starts. Update the managed guides, help text, and `_engineering/` notes in the slice that changes the behavior they describe.

1. **Shared foundations.**
   - Extract the Git runner (decision 1).
   - Surface error codes in JSON (decision 12).
   - Add parsers with fixtures for change notes and release records (strict YAML, all documented validation).
   - Extend `gitfixture` to create `release/<n>` annotated tags with release records, and add a push-capable remote (a local bare repository served over the same shim with `receive-pack`).
   - Add a Go fake for `gh` that records calls.
2. **Library authoring.**
   - `code-rules library change` (decision 9's layout, unique file names).
   - `code-rules library check` with Git history: the full failure list in `reference/cli.md#library-check`, the published-note warning, the shallow-clone failure, and the pending library release preview (`value.pendingRelease`).
   - `code-rules library add rule` next steps.
   - `code-rules library init` writes the check workflow (decision 13). Update `library-guide.md`.
3. **Library releases.** `code-rules library release` with `--dry-run`, `--no-github-release`, the first library release, reruns, and the GitHub Release page (decisions 2, 10, 11).
4. **Project configuration.** New fields (`rules`, `pins`, `exclude` objects with `replacedBy`), removal of `version` and `replace` (unknown fields now), `ref` limited to tags and full SHAs, `pins` and `ref` exclusive, stale-entry warnings versus unknown-ID failures. Rewrite the configuration fixtures. `code-rules project add library` gets `--rules` and an optional `--ref`.
5. **Project resolution.** Release listing and version resolution (decisions 3, 4), `_source.json` format version 2 (decision 5), `code-rules project sync` reading the lockfile (decisions 6, 7), offline verification in build and check, rule versions in generated guidance and provenance, the unreleased-`ref` warning and generated README note, and `replacedBy` in resolution. Make `exclude`, `pins`, and `rules` entries that name a rule the library retired warn instead of fail, while unknown IDs still fail (slice 4's `requireImported` in `internal/build/resolve.go` fails on both), including when a changed `ref` retires an excluded rule. Fix `generatedNotice` in `generated/provenance.json`, which says to run project commands from the project root although they work from any subdirectory. Update `project-guide.md` and `for-agents/index.md` if its text changed.
6. **`code-rules project update`.** Preview rows (`major`, `minor`, `patch`, `new`, `retired`, `replaced`, `pinned`), scoped updates, confirmation and prompts, `--yes`, `--keep`, `--exclude`, `--reason`, `--json`, and the atomic configuration write (decisions 8, 14).
7. **Forking.** `code-rules project add rule ID --from LIBRARY@VERSION`: find the version's library release, copy the rule, its assets, and linked shared assets with rewritten links, add attribution, and write the exclusion with `replacedBy`.
8. **Finish.** Rewrite the acceptance scenarios around the new lifecycle, verify every command example in the docs against the built binary, run an independent review of the docs against the implementation, and fix any drift. Delete this file.

## Breaking changes for the v0.2.0 release notes

Every breaking change from `v0.1.0`, with its manual migration, for the release pull request. Code Rules migrates nothing itself: old fields fail as unknown fields and old records fail as unsupported.

**Project configuration** (`.code-rules/config.yaml`)

- `sources.<name>.version`, the version constraint, is removed and fails as `unknown field version`. Delete it. The source then follows each rule's newest version when the project updates; to import exactly one library release, set `ref: release/<number>`; to hold a rule back, add a pin.
- `sources.<name>.replace` is removed and fails as `unknown field replace`, including an empty `replace: {}`. Move each entry into `exclude`, keeping its reason and naming its file as `replacedBy`: `rule-id: {reason: "…", replacedBy: local/<group>/<rule>.md}`.
- `sources.<name>.exclude` values are objects, not reason strings. A string fails with `expected an object`. Rewrite `rule-id: Reason.` as `rule-id: {reason: Reason.}`. An empty `exclude: {}` is no longer required.
- `ref` accepts only a tag or a full commit SHA, as before, but a version range is no longer accepted anywhere. `ref` is now optional, and can't be combined with `pins`. A `ref` that names a commit or a tag other than `release/<number>`, such as `v1.1.0`, still imports that revision. Its rules whose files match a published version record that version; the rest have no version, and sync and update warn about them, naming them, only when there is at least one.
- New, optional fields: `rules` (individually selected rules, unioned with `groups`, so `groups` is required only when `rules` is empty), `pins`, and an exclusion's `basedOn` (the library version a `replacedBy` rule incorporates, which forks write and sync checks the library published).

**Libraries**

- Projects import rule versions from `release/<number>` tags only. A library without one fails to import with `releases-not-found`, unless the source sets `ref`; older tags such as `v1.0.0` are ignored. Authors publish the first library release, which gives every rule `1.0.0`, with `code-rules library release` from an up-to-date default branch, and never create release tags by hand.
- After the first library release, `code-rules library check` requires a change note in `changes/` for every changed, new, or retired rule. Record them with `code-rules library change`. `changes/` is reserved for change notes.
- `code-rules library check` and `code-rules library add rule` now read Git history and tags, and fail in a shallow clone, even before the first library release, and with `missing-release-tags` in a clone whose commit has change notes but no `release/<number>` tags, such as one made with `git clone --no-tags`. CI must check out with full history, such as `fetch-depth: 0` with `actions/checkout`.
- `code-rules library init` also writes `.github/workflows/code-rules.yml`. Rerunning it in an existing library adds the workflow without touching other files, and never changes an existing workflow: to move a workflow to a newer Code Rules version, delete it and rerun `code-rules library init`. The workflow's install step prints the version it installed to the log and the job summary.
- The library README that `code-rules library init` wrote (from `internal/library/library-guide.md`) is never refreshed and still says to publish a Git tag. Replace its release section with the new guide's, or delete it and rerun `code-rules library init`.
- A declared license or notice file can no longer be a rule's Markdown file or lie in an asset directory inside a technology or practice group, because those files belong to a rule's version. `code-rules library check` rejects such a `rule-library.yaml`, and projects refuse to import a library release whose manifest declares one. Move the file to the library root or the library-root `assets/` directory and update `license.file` or `license.notices`.
- A project now imports only the shared `assets/` files that its selected rules link to, directly or through other shared files, instead of the whole directory.
- Library-wide files (group metadata, shared assets, license and notice files) come from one library release per source, recorded as `release` in `_source.json`, or from its `ref`; before, they came from the one revision the source named. A new source and newly selected groups or rules take them from the newest library release, `code-rules project update` of a whole source moves them to the newest library release even when no rule moves, and plain `code-rules project sync` keeps the recorded one. A `SOURCE:RULE` update leaves them unless the moved rule's version comes from a newer library release. That release is never older than any imported rule version's. The update preview shows a move as `Shared files: release 3 -> 4`, and its JSON as the source's `sharedFiles: {from, to}`, and a shared-files move alone is an update to apply, not "No rule updates are available". When a sync or update writes `local/<group>/_group.yaml` for a local rule's group, the metadata comes from the library release that supplies the source's shared files when it still has the group.

**Vendored snapshots** (`vendor/<source-name>/_source.json`)

- Format version 1 fails with `unsupported source record; delete .code-rules/vendor/ and run code-rules project sync to import it again, which imports unpinned rules at their newest versions`. Format version 2 records a `checksum` of its other fields, `ref`, `release`, `ruleSelection`, `retiredRules` (the retired rules the source would otherwise import: selected, or imported or listed by the previous record, including one it still imports, such as one a pin keeps, which the library README then marks retired; always present, refreshed by update and by a sync that changes the selection, ref, or shared-files release), and each rule's `version`, `release`, and `commit`, and drops `version`, `resolvedTag`, and `resolvedVersion`. It doesn't record pins or exclusions, so adding, rewording, or removing a pin that moves nothing, or an exclusion, by hand or with a fork, never makes it stale, except that pinning or excluding a retired rule it doesn't list needs a sync, which records the retirement; offline checks accept a pin only at the imported version or of a rule `retiredRules` lists, and an exclusion only of an imported rule or one `retiredRules` lists, and otherwise ask for `code-rules project sync`. Files are stored at their library paths, possibly from several library releases. Build, check, and sync all refuse a record whose checksum doesn't match, or that has none, including records from earlier preview builds, such as after a hand edit or a merge resolution; sync never records such a record again. The refusal says to restore a record sync wrote (`git checkout -- .code-rules/vendor/<source-name>/_source.json`, or, in a merge conflict, `git checkout --ours` or `--theirs` for that file) and run `code-rules project sync`, which applies any configuration changes, or to delete `vendor/<source-name>/` and sync, which imports unpinned rules at their newest versions. A record sync can't read says to delete `vendor/<source-name>/` and sync, which imports the source's unpinned rules at their newest versions.
- Migration for every project: edit the configuration as above, delete `.code-rules/vendor/`, run `code-rules project sync`, and commit the configuration, `vendor/`, and `generated/` together. With no record to restore, that sync imports every rule's newest version, except pinned rules and sources that set `ref`, so it can move every unpinned rule past the version the project used; to keep a library revision, set `ref: release/<number>` before syncing, or pin rules.

**Provenance** (`generated/provenance.json`)

- `sources[]` drops `version`, `resolvedTag`, and `resolvedVersion`, and adds `pins`, `release`, and `ruleSelection`.
- `rules[].origin` adds `version` and `release`, `null` for local rules and unreleased imports; `rules[].upstream` adds `version` and `release`; `rules[].basedOn` is new, the version a local replacement incorporates as its exclusion's `basedOn` records it, such as a fork's forked version, or `null`.
- `generatedNotice` says build and sync work from any subdirectory in Git repositories. Tools that read provenance must follow these fields; `code-rules project sync` regenerates the file.

**Generated guidance and the managed project guide**

- Each imported rule shows `Version: X.Y.Z` below its rule ID, in rule files and group pages. RULES.md and group pages ask reviewers to cite a rule's version with its ID. Each `generated/libraries/<source-name>/README.md` shows the library release, a rule version table, and the source's pins. Sync or build regenerates all of it; `code-rules project check` reports it stale until then.
- The managed `.code-rules/README.md` describes pins and updates; it doesn't mention forks or `--incorporated`. Build, sync, and init refresh an unedited older guide on their own. An edited guide stops them with `unrecognized or manually edited project guide`: move your notes to another file, move the guide aside, and rerun the command.

**Commands and JSON output**

- `code-rules project sync` no longer moves rules to newer versions: it restores the versions in `_source.json`, and chooses versions only for new sources, selections, pins, and `ref` changes. Adopt newer versions with the new `code-rules project update`, which previews them and applies them after confirmation, or with `--yes` in scripts and CI.
- `code-rules project add library`: `--ref` is optional and rejects version ranges; `--rules` is new. Scripts that passed a range should drop `--ref`, or pass `--ref release/<number>`.
- New commands: `code-rules project update`, `code-rules project add rule ID --from LIBRARY@VERSION`, `code-rules library change`, and `code-rules library release`.
- Code Rules never shows text from Git, a Git server, or the GitHub CLI; it reads that text privately to choose its own message and code. When the remote refuses a tag push, `code-rules library release` fails with `push-failed` and names the cause it recognizes, such as a repository rule (with GitHub's code, such as `GH013`), a server hook, or denied access, but not the server's lines; the docs show how to read them by pushing a test tag. `fetch-failed`, `github-release-failed`, and import failures (`connection-failed`, `certificate-failed`, `host-key-failed`, `object-fetch-refused`, `not-found-or-no-access`, `git-failed`) quote nothing either. Values Code Rules shows from Git's output, such as commit IDs and counts, are validated first, and an address with an authority but no scheme, such as `//user:password@host/rules`, is hidden. The `not-default-branch` refusal no longer names the default branch the remote advertises. The GitHub Release page URL is built from the repository and tag, not taken from `gh`. [The CLI reference](../docs/src/content/docs/reference/cli.md#text-from-git-servers-and-the-github-cli) states what Code Rules never shows and what it shows as is.
- Every command exits `130` when interrupted, such as with Ctrl-C, including at a prompt, which now says `cancelled; no files were written` instead of `terminal input ended ... context canceled`; an interrupted sync or update says the same, except after it first recovered an interrupted earlier command, when it says it was cancelled and what the recovery did. A failed sync or update that wrote nothing ends its message with `No files were written.` and names its source as `source NAME:` instead of `sync project: import source "NAME" failed (no libraries were returned …)` or `update source "NAME":`. An invalid prompt answer lists the valid answers. A missing required input without a terminal names the flag to pass. Before, an interrupt exited `1`.
- `code-rules library release --dry-run` checks that `gh` is installed and signed in when the library release would create a GitHub Release page, and refuses as the real run would.
- The `code-rules project update` preview labels each summary with its version when a row spans several versions, and its JSON rows add `summaryVersions`.
- A fork of a pinned rule removes the pin in the same configuration write that adds its exclusion, and warns. A fork's exclusion records the forked version as `basedOn`, and the `replaced` update row compares the library's newest version with it rather than with the imported version, also when a pin keeps the imported copy; a replacement without `basedOn` keeps the imported-version comparison. `code-rules project update --incorporated SOURCE:RULE` and the terminal's `incorporated` answer set `basedOn` to the newest version, and the row's JSON adds `basedOn` and the `incorporated` decision.
- A `ref` naming a branch fails with `ref-is-branch` instead of `version-not-found`. Every usage error, such as an unknown flag, a missing argument, or flags a command can't combine, like `--bump` for a new rule in `code-rules library change`, has code `invalid-arguments` and exits `2`; before, usage errors had no code. The commands `code-rules library check` suggests include `--bump` and `--summary` placeholders.
- An out-of-date `code-rules project check` prints only its report on stdout, without a leading `Error:` line. Problems in a source record name its path relative to the Code Rules directory, such as `vendor/team/_source.json` (JSON `error.location`), instead of `team/_source.json`.
- `code-rules project update` says when a retired rule's replacement was retired too, with `replacementRetired` and `currentReplacement` in JSON, instead of recommending it; `code-rules library check` warns when a pending note retires an earlier retirement's replacement.
- JSON output changes for existing commands:
  - `error.code` now appears for Git and import failures (such as `releases-not-found`, `version-not-found`, `shallow-clone`, `missing-release-tags`, and `change-notes`; an unreachable host is `connection-failed`, a server whose TLS certificate or SSH host key Git can't verify is `certificate-failed` or `host-key-failed`, all apart from `not-found-or-no-access`, a failure Code Rules doesn't recognize is `git-failed`, and a server that refuses to send files by object ID is `object-fetch-refused`). Scripts that treated any `code` as a file-transaction error should check its value.
  - `code-rules library check` adds `value.pendingRelease`, with `release`, `libraryFiles` (the library-wide files changed since the latest library release, which the human output lists too), and `rules`, whose rules name versions `from`, `to`, and, for a retired rule, `lastVersion`, with `summaries`, as the release record and `code-rules project update` do; `code-rules library release` reports its `value.rules` the same way.
  - `code-rules project sync` adds `value.warnings`.
  - `code-rules library check` renames its counts `value.groups` and `value.rules` to `value.groupCount` and `value.ruleCount`, since `rules` is a list everywhere else.
  - Authoring results drop the legacy `value.next` text; use `value.nextSteps`. Each `code-rules project check` problem replaces its `nextStep` string with `nextSteps`, a list of the same `{instruction, commands}` objects.
  - Authoring commands (project and library init, add group, add rule including forks, add library, and library change) replace `value.files` with `value.added` and `value.changed`, so created files are told apart from modified ones such as `config.yaml`.
  - Every list field is always present, empty when there's nothing to report, including `value.warnings` of sync, build, and authoring commands and a next step's `commands`, which were left out when empty.
  - Enumerated values are kebab-case: `code-rules project check`'s `value.status` is `up-to-date` or `out-of-date`, its problem kinds are `missing-file`, `stale-contents`, `unexpected-file`, and `outdated-readme`, and a stale check's `error.kind` is `out-of-date`.
- Human output changes, for scripts that read it:
  - Authoring commands (project and library init, add group, add rule including forks, add library, and library change) list files under `Added:` and `Changed:`, relative to the working directory, instead of absolute paths under `Updated files:`.
  - Sync, update, build, and check label paths `Paths relative to DIR:`.
  - Errors no longer start with a blank line.
  - Generated library release notes put each paragraph and list item on one line, list each summary as a nested item, and open the first library release with `Library release 1 publishes N rules.`, without the placeholder summaries.
  - A fork's `attribution.description` and messages say `library release N` or `release/N`, never `library release release/N`.
  - The generated `libraries/<source-name>/README.md` adds a Status column to its Rule versions table: active, excluded, or replaced by a local rule.
  - A duplicate repository names the other source; a pin of an unpublished version lists the published versions, at most ten; sync and update warn about a rules entry whose group `groups` already selects.

**This repository**

- `.code-rules/config.yaml` still uses `replace: {}` and imports `fabricahq/public-rules` at `v1.1.0`, so `v0.2.0` rejects it. Migrate it after public-rules publishes its first library release; see [After the CLI ships](#after-the-cli-ships-fabricahqpublic-rules).

## Final review

When every slice is done and its review fixes have landed, run four independent GPT-6 Astra reviews (Codex CLI, read-only) of the whole change, from `main` to the top of the PR stack. Each reports findings without changing code, and each finding states the insight, its severity, a recommendation, and its blast radius:

1. **Security** concerns across the whole change.
2. **Simplification:** complexity to remove and more elegant abstractions, including ones that alter behavior slightly when that saves considerable complexity.
3. **Testability** of the code.
4. **Correctness** of the code.

Then assemble the findings into one set of recommendations for Josh about what, if anything, to change.

## Testing

- **Parsers:** table fixtures under `internal/rules/testdata/` for configuration, change notes, and release records, covering every documented rejection.
- **Library lifecycle:** check fails without a note, passes with one; the first library release gives every rule `1.0.0`; a later library release computes major, minor, patch, new, and retired correctly, using the largest change per rule and listing every summary; reruns create only what's missing; a behind or ahead branch is refused; a shallow clone fails with instructions; a rejected tag push stops cleanly.
- **Project lifecycle** (acceptance): add a library, sync, and check; publish a new library release in the fixture; preview and apply an update; keep a major change with `--keep`; exclude a new rule; pin a rule down and back up; retire a pinned rule; `ref` to a library release and to an unreleased commit (warning, `null` versions); a scoped `SOURCE:RULE` update; a `replaced` row; forking.
- **Interactive:** `terminalfixture` tests for the update prompts, including cancelling, which writes nothing.
- **Recovery:** interrupting an update that writes `config.yaml` recovers all three targets together.
- **Limits:** a fixture with many `release/<n>` tags stays within the listing limits.
- **Validation:** the commands in `CONTRIBUTING.md`, including `gofmt`, `go vet`, Staticcheck, `go test -race ./...`, govulncheck, the build, and `bun run check` after docs changes.

## After the CLI ships: `fabricahq/public-rules`

Plan this as its own pull request in that repository, after the `v0.2.0` CLI release. Nobody depends on the library yet, so its old `v1.0.0` and `v1.1.0` tags and release-planner setup can go.

1. Replace the release-planner workflow and files with the check workflow from `code-rules library init`, and upgrade the pinned Code Rules version.
2. Publish its first library release with `code-rules library release`, so every rule starts at `1.0.0`. Josh approves the tag.
3. Update its `README.md`, `AGENTS.md`, and any `.claude` or `.agents` instructions that describe releases.
4. Move this repository's own `.code-rules/` ([fabricahq/code-rules#69](https://github.com/fabricahq/code-rules/pull/69) imports public-rules `v1.1.0`) to the new format and public-rules' first library release.

## Out of scope

- Rulemart, Fabrica's catalog of public libraries. It will read the release tags and GitHub Release pages.
- Checking application code for compliance with rules. See the product scope in `AGENTS.md`.
