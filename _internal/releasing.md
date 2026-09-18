# Releases

Say **“let's release”** to an agent working in this repository. The agent prepares a release PR containing `releases/v<version>.md`. Edit that Markdown file in the PR, save your changes, then **merge the PR to approve publication**.

**The merged Markdown file becomes the GitHub release description verbatim, including your manual edits.** The PR description and review comments are separate review context. Saving an intermediate edit does not publish anything; after merge, the workflow publishes your approved notes once the assets pass verification.

GitHub does not trigger Actions on draft-release saves. A release PR gives notes version history and makes approval explicit. The workflow builds assets first, attaches them to a draft, verifies GitHub's stored SHA-256 digests, and publishes last. [GitHub's release events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#release) and [release management guidance](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository) explain these constraints.

The first release is **v0.1.0**. No release is requested by adding the automation itself. The tool uses the [MIT license](../LICENSE.md), as selected by the owner. Release packaging includes these terms and refuses a missing or empty license file.

## Agent procedure

1. **Establish the range.** Fetch `main` and tags. List published releases, including prereleases, and resolve the most recent version's tag to a commit. Verify it is an ancestor of the intended `main` commit. A draft or a tag without a published release is unfinished work: inspect that attempt before preparing another. With no published releases, use `v0.1.0` and review the repository's full history plus its current capabilities.
2. **Account for all changes.** Read every commit since the previous released tag and its associated PR, including direct commits. Read relevant diffs, tests, and owning documentation. In the PR description, record the base tag, analyzed source SHA, and an inventory mapping every commit/PR to a release-note entry or a reason for omission. Account for reverts and superseded work. Internal refactors, CI changes, and website-only changes may belong only in the inventory; do not present them as new CLI features.
3. **Choose the version.** Apply the policy below and explain the proposed increment in the PR description. Treat command names, flags, configuration, generated file formats, and supported platforms as the public contract. Flag uncertainty instead of inventing compatibility guarantees.
4. **Draft the notes.** Create exactly one `releases/v<version>.md` file on a release branch. Its filename owns the version; its contents become the GitHub release body verbatim. Follow the editorial format below. Include PR/documentation links and actionable migration steps. For the initial release, describe the supported product rather than the sequence of internal migration commits.
5. **Validate and present.** Confirm approved tool terms exist. Run `go run ./cmd/release-plan --base origin/main --head HEAD` after committing the notes. Open a PR titled `Release v<version>` and link directly to the Markdown file's GitHub editor. State that saving edits updates the draft and merging authorizes automated publication. Leave the PR for the maintainer; the release request author does not merge it on the maintainer's behalf.
6. **Preserve edits and refresh the range.** When asked to revise the draft, fetch its branch and retain manual edits. If `main` advanced, account for the new commits, update the inventory and notes, and bring the release branch up to date before approval. Keep feature development outside the release PR. Finish only when every change through the reviewed source is accounted for and CI is green.

For a second agent reviewing the draft, provide the previous tag, analyzed SHA, release-note file, and inventory. Ask it to identify omitted user-visible changes, unsupported claims, missing migration instructions, and an incorrect version increment. It should report evidence and propose edits without replacing the maintainer's wording or publishing.

## Editorial format

Follow the structure demonstrated by Runbooks [beta-v0.9.0](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.9.0), [beta-v0.8.3](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.8.3), and [beta-v0.5.0](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.5.0). [Research notes](release-style-research.md) record more examples.

Use these headings when they have content:

- `## ✨ New Features`: capabilities users could not perform before.
- `## ⬆️ Improvements`: better behavior, performance, or usability of an existing capability.
- `## 🐛 Squashed Bugs`: the trigger, previous incorrect behavior, and corrected result.
- `## ⛓️‍💥 Breaking Changes`: affected users, the old/new contract, and required migration commands or examples. Also call out a breaking change near the start so readers cannot miss it.
- `## What's Changed`: linked PR/commit inventory for readers who want implementation detail.
- `## New Contributors`: only verified first contributions, with credit.

Give significant changes a descriptive `###` heading and a short explanation of why the change matters. Use bullets for small changes, code for commands, and before/after examples when they make an upgrade clearer. Omit empty categories. Group related commits into one coherent entry. Preserve the human's final wording.

End subsequent releases with one **Full Changelog** link using the actual previous and new tags. For `v0.1.0`, link to the tagged source/history and say it is the first release. Check all links; do not copy sample versions or comparison URLs from the reference notes.

## Version policy

Use [SemVer 2.0.0](https://semver.org/) with Git tags `vMAJOR.MINOR.PATCH`. The CLI reports the same version without `v`. Prereleases use a suffix such as `v0.2.0-rc.1` and are marked as prereleases on GitHub. Build metadata is omitted from release tags because `manifest.json` records the exact source commit.

- Start at `v0.1.0`.
- Before `1.0.0`, increment patch for compatible fixes and minor for features or breaking changes. Always document breaking changes; `0.x` is not permission to hide them.
- From `1.0.0`, increment major for incompatible public-contract changes, minor for compatible features, and patch for compatible fixes.
- Each requested version must be newer than existing version tags. Publish one release before requesting the next.
- Published versions and tagged request files are immutable. Correct later behavior in a new release. If only published prose needs correction, edit the GitHub release description deliberately; do not rerun publication to overwrite it.

## What approval runs

A `main` push adding `releases/v<version>.md` starts [Release](../.github/workflows/release.yml). Release PRs validate the request but do not publish. The merge/push commit owns both the notes and the source; advancing `main` during a build cannot change the selected source.

CI validates the version, checks formatting, runs vet, Staticcheck and race-enabled Go tests on Linux and macOS, and builds macOS/Linux archives for amd64/arm64. The distribution tests run extracted host binaries, exercise project commands, install two versions, and verify upgrade/rollback and corrupted-archive refusal. Cross-compilation does not establish execution compatibility on every architecture.

Publication is serialized across all versions. A changed tag inventory invalidates an older plan before it can publish out of order. Build jobs have read-only repository credentials. The read-only Linux build job compiles `cmd/publish-release`. Only the final publication job has `contents: write`; it downloads that executable and the verified bundle, then supplies `GITHUB_TOKEN` to the publication command. That job does not check out source or compile dependencies. After all builds/tests pass, that job:

1. Verifies the approved commit, version, license presence, four archives, manifest, sizes, and checksums.
2. Verifies the predecessor is published and creates an unmoved version tag at the exact tested commit.
3. Rechecks that the version tags still match the validated plan, then creates a draft release with the approved notes and uploads the four archives, `manifest.json`, and `SHA256SUMS`.
4. Checks the complete remote asset set and GitHub's stored digests, rechecks the tag and notes, then publishes.

A failed build or upload cannot publish an incomplete release. If upload already started, the draft and tag may remain. The workflow neither signs nor notarizes binaries. Repository rules may restrict tag creation or `GITHUB_TOKEN` writes; resolve those policies before the first release rather than adding a personal token. Keep `main` protected against unreviewed release requests and workflow changes.

## Retry a failed release

Use **Actions → Release → Run workflow**, select `main`, and supply the full commit SHA that added the notes. This retries the original approved source, not the latest `main`. You can also rerun the original failed workflow.

Retries accept an existing tag only at the same commit. A matching draft resumes missing uploads; matching published releases require no writes. Changed notes, unexpected assets, bad digests, tag conflicts, or a newer version cause refusal. Inspect the failed run before taking corrective action. If an interrupted upload left an invalid asset, a maintainer can remove that asset from the unpublished draft and retry. Never delete or replace published assets/tags as a retry strategy.

If a build failed before creating a tag, correct the source or terms in a separate PR, then edit the untagged notes file in a new release PR. Merging those corrected notes approves the new commit. You can also withdraw an untagged request by deleting its notes file. Wait for the earlier run to finish before approving a replacement. If a tag/draft already exists, inspect and explicitly resolve that unpublished attempt first; the automation will not move its tag.

## Candidate archives

For unpublished review builds, commit intended source changes first:

```sh
go run ./cmd/package-binaries --candidate --output /tmp/code-rules-artifacts
```

Packaging uses an isolated copy of committed HEAD; uncommitted edits are excluded. Candidates default to `0.0.0-dev.g<commit>` and accept an explicit `--version`. Release packaging requires an explicit unprefixed `--version`, clean source, and nonempty `LICENSE.md`. The packager itself never publishes.

The output directory must be new. A failed build retains an `INCOMPLETE` marker; retry into a new directory. Verify archive SHA-256 values against a trusted manifest before extraction. Checksums detect corruption, not substitution of both an archive and its manifest.

The [PR packaging workflow](../.github/workflows/package-binaries.yml) uploads candidate bundles for seven days. Its separate notification workflow posts a download link for the exact PR commit. It does not execute PR code or read artifact contents. GitHub sign-in is required for candidate downloads.
