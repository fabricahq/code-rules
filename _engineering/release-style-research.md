# Release style and workflow research

Research date: 2026-09-18. Recommendations below are design guidance, not a claim that publication is enabled.

## Runbooks release style

Read six releases from `beta-v0.9.0` and earlier through GitHub's release API. The requested page changes as releases accumulate; tag links identify the examples precisely.

The repeated headings are:

- `## ✨ New Features`
- `## ⬆️ Improvements`
- `## 🐛 Squashed Bugs`
- `## ⛓️‍💥 Breaking Changes`
- `## What's Changed`
- `## New Contributors`, when applicable
- A final **Full Changelog** comparison link

The releases omit categories with no entries. Significant changes get a level-three heading, a short explanation of user impact, and supporting PR or documentation links. Small changes can use bullets. Breaking changes include migration instructions, before/after code, or a mapping of old names to new names.

Examples:

- [beta-v0.9.0](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.9.0): two improvements to GitClone outputs; a breaking-change table maps renamed keys and shows the required template update. Includes contributor credit.
- [beta-v0.8.3](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.8.3): a sample runbook under new features; a bug explanation identifies the trigger, incorrect behavior, and corrected behavior.
- [beta-v0.8.0](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.8.0): improvements, a detailed crash fix, optional `Minor Improvements` bullets, and a before/after example for a required property.
- [beta-v0.7.0](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.7.0): new syntax explained with a comparison table and runnable examples; compatibility and migration details appear explicitly.
- [beta-v0.6.0](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.6.0): a major feature walkthrough uses commands, code, and screenshots; smaller improvements and fixes use bullets.
- [beta-v0.5.0](https://github.com/gruntwork-io/runbooks/releases/tag/beta-v0.5.0): features explain the problem they solve, then list capabilities and links. Separate improvements, bugs, and breaking changes follow.

Copy the editorial pattern, but avoid incidental errors in the examples. The `beta-v0.8.0` comparison link ends at a different tag; some older notes repeat the generated changelog. Generate one final comparison link from the actual released tags. Use `v0.1.0` for this project's first tag, not Runbooks' `beta-v` prefix.

## Human editing and publication

GitHub Actions does **not** trigger for draft-release `created`, `edited`, or `deleted` events. Saving a GitHub draft release cannot directly start the build. The `published` event happens after the release is public. [GitHub release-event documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#release)

Build assets first, attach them to a draft, then publish. GitHub recommends this sequence for immutable releases, because publication freezes their assets and tag. Notes remain editable afterward. [GitHub release management](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository)

The selected implementation, [Release Planner](https://release-planner.fabricahq.com), uses a release-note Markdown file: the maintainer edits it in a release PR, and merging approves publication. The release PR tests and builds the assets; the push to the default branch publishes them. This adds a merge step, but gives versioned notes and familiar review history. See [Releases](releasing.md) for how Code Rules uses it.

If Actions creates tags or publishes releases using `GITHUB_TOKEN`, do not expect those events to trigger another workflow. Keep the build/upload/publish sequence in one workflow, or dispatch the next workflow explicitly. [GitHub workflow-trigger rules](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow)

