# Releases

This document owns release policy and the agent procedure. Read the linked implementation for workflow mechanics and command options.

Say **“let's release”** to an agent working in this repository. The agent prepares a release PR containing `releases/v<version>.md`. Edit that Markdown file in the PR, save your changes, then **merge the PR to approve publication**.

**The merged Markdown file becomes the GitHub release description verbatim, including your manual edits.** The PR description and review comments are separate review context. Saving an intermediate edit does not publish anything; after merge, the workflow publishes your approved notes once the assets pass verification.

## Agent procedure

1. **Establish the range.** Fetch `main` and tags. List published releases, including prereleases, and resolve the most recent version's tag to a commit. Verify it is an ancestor of the intended `main` commit. A draft or a tag without a published release is unfinished work: inspect that attempt before preparing another. With no published releases, use `v0.1.0` and review the repository's full history plus its current capabilities.
2. **Account for all changes.** Read every commit since the previous released tag and its associated PR, including direct commits. Read relevant diffs, tests, and owning documentation. In the PR description, record the base tag, analyzed source SHA, and an inventory mapping every commit/PR to a release-note entry or a reason for omission. Account for reverts and superseded work. Internal refactors, CI changes, and website-only changes may belong only in the inventory; do not present them as new CLI features.
3. **Choose the version.** Apply the policy below and explain the proposed increment in the PR description. Treat command names, flags, configuration, generated file formats, and supported platforms as the public contract. Flag uncertainty instead of inventing compatibility guarantees.
4. **Draft the notes.** Create exactly one `releases/v<version>.md` file on a release branch. Its filename owns the version; its contents become the GitHub release body verbatim. Follow the editorial format below. Include PR/documentation links and actionable migration steps. For the initial release, describe the supported product rather than the sequence of internal migration commits.
5. **Validate and present.** Confirm approved tool terms exist. Run `go run ./cmd/release-plan --base origin/main --head HEAD` after committing the notes. Open a PR titled `Release v<version>` and link directly to the Markdown file's GitHub editor. State that saving edits updates the draft and merging authorizes automated publication. Leave the PR for the maintainer; the release request author does not merge it on the maintainer's behalf.
6. **Preserve edits and refresh the range.** When asked to revise the draft, fetch its branch and retain manual edits. If `main` advanced, account for the new commits, update the inventory and notes, and bring the release branch up to date before approval. Keep feature development outside the release PR. Finish only when every change through the reviewed source is accounted for and CI is green.

**After the agent finishes, the maintainer merges the release PR to approve publication.** Automation takes over: it validates the merged commit, builds and verifies the release assets, and publishes the GitHub release with the approved notes verbatim. No separate publish action is needed. If a check or upload fails, the release stays unpublished; follow [Retry a failed release](#retry-a-failed-release).

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

Use [SemVer 2.0.0](https://semver.org/) with Git tags `vMAJOR.MINOR.PATCH`. The CLI must report the same version without `v`. Use a suffix such as `v0.2.0-rc.1` for prereleases and mark them as prereleases on GitHub. Omit build metadata from release tags.

- Start at `v0.1.0`.
- Before `1.0.0`, increment patch for compatible fixes and minor for features or breaking changes. Always document breaking changes; `0.x` is not permission to hide them.
- From `1.0.0`, increment major for incompatible public-contract changes, minor for compatible features, and patch for compatible fixes.
- Each requested version must be newer than existing version tags. Publish one release before requesting the next.
- Published versions and tagged request files are immutable. Correct later behavior in a new release. If only published prose needs correction, edit the GitHub release description deliberately; do not rerun publication to overwrite it.

## Publication policy

- Publish only the source and notes approved by merging the release PR. Do not substitute a newer `main` commit during publication or a retry.
- Keep the GitHub release as a draft until validation passes and all release assets are attached and verified. Never publish an incomplete release.
- Include the approved [MIT license](../LICENSE.md) in release packages.
- Follow [security practices](security-practices.md) for credentials, dependency updates, and review protections. Resolve repository permission restrictions before releasing; do not work around them with a personal token.

For execution details, read the [release workflow](../.github/workflows/release.yml), [release planner and publisher](../internal/release/), and [packager](../internal/distribution/).

## Retry a failed release

Prefer **Re-run all jobs** on the original failed Release run. To start a manual retry, use **Actions → Release → Run workflow**, select `main`, and copy both **Base SHA** and **Approved head SHA** from the original run summary into the corresponding inputs.
Preserve the full approved range. A rebase merge can contain several commits; the commit that first added the notes may omit later approved edits or source changes. Never substitute the latest `main` or guess the base from the head's parent.

Inspect the failed run and any existing draft, tag, or published release before taking corrective action. If an interrupted upload left an invalid asset, a maintainer can remove that asset from the unpublished draft and retry. Never delete or replace published assets/tags as a retry strategy.

If a build failed before creating a tag, correct the source or terms in a separate PR, then edit the untagged notes file in a new release PR. Merging those corrected notes approves the new commit. You can also withdraw an untagged request by deleting its notes file. Wait for the earlier run to finish before approving a replacement. If a tag/draft already exists, inspect and explicitly resolve that unpublished attempt first; do not move its tag.

## Candidate archives

For unpublished review builds, commit intended source changes first:

```sh
go run ./cmd/package-binaries --candidate --output /tmp/code-rules-artifacts
```

Use a new output directory for each attempt. Verify archive checksums against a trusted manifest before extraction. Describe these builds as unpublished review candidates, and claim platform compatibility only where testing supports it.

Read the [packaging command](../cmd/package-binaries/) for options and the [PR packaging workflow](../.github/workflows/package-binaries.yml) for downloadable CI builds.
