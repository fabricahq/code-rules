---
title: "Version your rules"
description: "How rules change, how rule versions describe those changes, and how to publish new versions from your library."
---

Rules change over time. Some changes are small: an author fixes a typo, rewords a confusing sentence, or adds an example. Others are large: a rule becomes stricter, starts requiring something it previously only recommended, or is retired entirely.

Just as with code, you can track these changes by versioning your rules. This guide explains how rule versions work, then shows how to record changes and publish new versions from your library.

## Rule versions

A **rule version** identifies one state of a rule, such as `1.3.0`. When a rule changes, it gets a new version, along with a summary of what changed.

Each rule has its own version, even though one library repository holds many rules. Each library release records every rule's version, and the summary of each change, in its Git tag, so the tags together hold each rule's full history. The rule file itself contains no version.

A version covers the rule's Markdown file and its own supporting files. Library-wide files, such as group descriptions and shared diagrams, aren't part of any rule version.

## Semantic versions

Rule versions are [semantic versions](https://semver.org/): three numbers, `MAJOR.MINOR.PATCH`. Which number increases tells you how large the change was. Software uses these numbers to describe changes to an API. For rules, they describe changes to the rule's **obligation**: what work must do to comply with it.

| Change | Definition | Example |
| --- | --- | --- |
| **Major**, such as `1.3.0` to `2.0.0` | Work that complied with the previous version could fail this one. | Lower the required retry limit, or require a test the rule previously only recommended. |
| **Minor**, such as `1.3.0` to `1.4.0` | Work that complied with the previous version still complies, and this version adds new guidance. | Add a Python example of the same test. |
| **Patch**, such as `1.3.0` to `1.3.1` | Work that complied with the previous version still complies, and this version adds no new guidance. | Fix a misleading sentence or a typo in an example. |

To decide, ask two questions in order. First, could work that complied with the previous version fail this one? If yes, the change is major. If no, does this version add new guidance, such as a new example or advice for a new situation? If yes, the change is minor; if no, it's a patch. Watch for changes that only add text but widen where the rule applies: code in the newly covered situation may not comply, which makes the change major. When unsure, choose the larger change.

A few more conventions:

- A new rule starts at `1.0.0`.
- Renaming or moving a rule changes its ID, so the old ID is retired and the new ID starts at `1.0.0`. See [Retired rules](#retired-rules).

## Retired rules

Sometimes a rule shouldn't change; it should stop. **Retiring** a rule ends its history: the library stops publishing it, and its last version stays its final version. A retirement always explains why, and it usually happens for one of two reasons:

- **A better rule replaces it.** For example, you fold a narrow rule about retry limits into a broader rule about testing retries. The retirement names the replacement, so projects know what to adopt instead.
- **The practice is no longer recommended.** For example, you published a rule about how agents should comment code, then concluded from feedback that agents shouldn't add those comments at all. There's no replacement; the summary explains why the advice was withdrawn.

The library release that publishes a retirement records it with its summary and any replacement. A retired rule's ID is never reused.

Retiring a rule a project uses is as disruptive as a major change, because its agents stop following the rule. Projects see retirements in their update preview, next to major changes.

## How projects use rule versions

Projects import rules from your library. Each imported rule's version appears in the project's generated guidance, so agents and reviewers can cite the exact version they followed.

A project keeps the versions it imported until someone runs `code-rules project update`, which previews each rule that changed, with its old and new versions and your summaries, plus new and retired rules. Nothing applies until someone on the project confirms, so major changes, which can change what the project's code must do, get reviewed first. See [Update rules](/guides/update/).

A project can also pin a rule to a version, with a reason, while the rest moves forward. That's how projects hold back one major change, often while they give you feedback on it. Projects import only published versions, unless one deliberately imports an exact commit, so changes waiting for a library release don't reach them.

## Library releases

New rule versions reach projects through library releases. A **library release** publishes all of a library's pending changes together: every rule that changed since the last library release gets its new version at the same moment, and changes to library-wide files, such as group descriptions and shared diagrams, go out with it.

A library release can be as small as a single rule change, published as soon as it's ready. It can also be a larger, meaningful collection of rule updates that you announce together and point users to as a snapshot of the library.

Library releases are numbered 1, 2, 3, and so on. They don't use semantic versioning. One library release contains many rules, each with its own version, so no single version number could describe the library release: the same library release might carry a patch to one rule and a major change to another. The release number only identifies the library release. Compatibility is described rule by rule, by each rule's own version.

A library release produces:

- a release tag, such as `release/4`, whose message records every rule's version and the release notes, and
- on GitHub.com, a **GitHub Release page** that announces the library release with the same release notes. The page is only an announcement; the library release itself is the tag.

Publish a library release whenever you want your pending changes to reach projects. There's no schedule, and no need to wait for a major change: smaller library releases are easier for projects to review. Projects don't have to take a whole library release, either. Each project chooses versions rule by rule, and can import one specific library release when it wants a known snapshot of the library.

## How library releases work

You manage versions and library releases with three Code Rules commands. You never create tags by hand.

- **`code-rules library change`** records a change. You say how large the change is (major, minor, or patch) and summarize it. The command saves this in a **change note**, a small file in the library's `changes/` directory.
- **`code-rules library check`** confirms that every changed rule has a change note, and previews the versions the next library release will publish.
- **`code-rules library release`** publishes a [library release](#library-releases).

Change notes pile up on `main` as you merge changes, and they're never deleted. Publishing a library release doesn't change any files: `code-rules library release` computes each rule's next version from the notes added since the previous library release, and records the result in a new `release/<number>` tag. The next library release starts from that tag.

Change notes start with the second library release. Your library's [first library release](#publish-the-first-library-release) gives every rule version `1.0.0`, so it needs no notes. For the exact formats, see [Rule versions](/reference/rule-versions/) and [Change notes](/reference/rule-versions/#change-notes).

## Publish the first library release

Before your first library release, its rules have no versions, so you don't need change notes. When the rules are ready for projects to use, publish the first library release the same way as every later one; see [Publish a library release](#publish-a-library-release). It gives every rule version `1.0.0`.

From then on, [record every change with a note](#change-rules-after-the-first-library-release).

## Change rules after the first library release

After the first library release, every change to a rule needs a change note, recorded with `code-rules library change` in the same pull request as the change. The steps differ slightly for each kind of change.

### Update a rule

Suppose you want to add a Python example to `practices/testing/verify-retry-limits`. First, edit the rule's Markdown file as usual. Then choose the change level, as described in [Semantic versions](#semantic-versions), and record it with `code-rules library change`:

```sh
code-rules library change practices/testing/verify-retry-limits \
  --bump minor \
  --summary 'Add a Python example of the retry-limit test.'
```

This writes a new note, such as `changes/2026-09-29-verify-retry-limits-7f3a9c.yaml`:

```yaml
summary: Add a Python example of the retry-limit test.
rules:
  practices/testing/verify-retry-limits: minor
```

Write the summary for someone deciding whether to update: say what changed in the obligation or guidance, not how you edited the file. Every note gets its own file, so two pull requests that change the same rule never conflict. When several notes name the same rule, the library release uses the largest change and lists each summary.

### Add a rule

Create the rule as usual, and complete the draft it writes:

```sh
code-rules library add rule practices/testing/verify-backoff \
  --title 'Verify retry backoff' \
  --when-to-read 'When adding or changing retry delays.' \
  --impact MEDIUM \
  --impact-description 'Prevents retries from overloading a service.'
```

Then record it. A new rule always starts at version `1.0.0`, so leave out `--bump`:

```sh
code-rules library change practices/testing/verify-backoff \
  --summary 'Add a rule about testing retry backoff.'
```

### Retire a rule

Delete the rule's Markdown file and its asset directory, then record the retirement.

When a better rule replaces it, name the replacement:

```sh
code-rules library change practices/testing/check-retry-backoff \
  --retire \
  --replaced-by practices/testing/verify-retries \
  --summary 'Covered by the broader rule about testing retries.'
```

The replacement must exist in the library by the time the retirement is published. It can be a new rule in the same library release.

When the practice itself is no longer recommended, leave out `--replaced-by`, and explain why in the summary:

```sh
code-rules library change practices/code-design/comment-intent \
  --retire \
  --summary "Agents shouldn't add explanatory comments."
```

A rule that was never published can't be retired; just delete it.

### Rename a rule

A rename changes the rule's ID. Move the file, then record one note that retires the old ID, replaced by the new one, and adds the new ID:

```yaml
summary: Rename to describe what the rule checks.
rules:
  practices/testing/check-retry-backoff:
    change: retired
    replacedBy: practices/testing/verify-retry-backoff
  practices/testing/verify-retry-backoff: new
```

Projects see the old rule retired, with the new rule as its replacement.

### Change a shared file or group description

Files in the library's shared `assets/` directory, such as a diagram, and group descriptions in `_group.yaml` are library-wide files. They aren't part of any rule version, so changing them needs no change note. `code-rules library check` lists the ones that changed, and the next library release publishes them, even if no rule changed.

A project receives library-wide files from one library release at a time. `code-rules project update` moves them to the newest library release, even when no rule moves, so a library release that changes only shared files reaches projects with their next update. A project that starts importing your library, or selects more of it, also gets them from the newest library release.

Keep everything that defines a rule's obligation in the rule itself. A rule pinned to an older version can link to newer shared files, from the library release that supplies the project's library-wide files, so shared files should only explain and illustrate.

### Check your changes

Whatever you changed, check the library before opening your pull request:

```sh
code-rules library check
```

`code-rules library check` fails if a changed rule has no note, or if a note doesn't match a change. When it passes, it previews the pending library release:

```text
Library is valid: 3 group(s), 6 rule(s).

Pending library release 3
  practices/testing/verify-retry-limits  minor  1.2.0 -> 1.3.0
```

When you changed library-wide files, such as a group description, the preview lists them under `Library-wide files changed since release/2:`, so you know the next library release publishes them even if no rule changed.

Commit each rule and its note together. Reviewers can then review the rule and the wording of its note in the same pull request.

## Publish a library release

Publish from an up-to-date checkout of `main`. This works on any Git host.

### Review the release notes

Preview exactly what will be published:

```sh
code-rules library release --dry-run
```

The preview shows the repository, branch, commit, release number, each rule's change and versions, and the complete release notes. The release notes are generated from the change notes, so to change their wording, edit the notes in `changes/` in a normal pull request, then preview again.

### Publish

```sh
code-rules library release
```

The command fetches from the remote first. It refuses unless you're on the remote's default branch, your branch matches the remote exactly, and `code-rules library check` passes. It then creates the annotated `release/<number>` tag on the current commit, pushes it, and creates the GitHub Release page with the [GitHub CLI](https://cli.github.com/). No files change and nothing is committed. Because it pushes a tag, run it as someone allowed to push to the repository.

If you don't want a GitHub Release page, or the library isn't hosted on GitHub.com, run `code-rules library release --no-github-release`; repositories hosted elsewhere get the tag only.

### What the GitHub Release page looks like

For a library release that changes four rules and a group description, the generated page, titled `release/4`, looks like this:

```md
Library release 4 changes 4 rules: 1 major, 1 minor, 1 new, and 1 retired.

## Major changes

Code that complied with the previous rule version could fail the new one, so review these before updating.

- **practices/testing/verify-retry-limits** `1.3.0` → `2.0.0`
  - Require a test at the limit for every retry policy.

## Minor changes

- **techs/react/test-hooks-in-isolation** `2.1.0` → `2.2.0`
  - Add an example for custom hooks.

## New rules

- **practices/testing/verify-retries** `1.0.0`
  - Add a broader rule about testing retries.

## Retired rules

- **practices/testing/check-retry-backoff**, last version `1.2.0`, replaced by **practices/testing/verify-retries**
  - Covered by the broader rule about testing retries.

This library release also updates shared files, such as group descriptions or shared assets.

<details>
<summary>All rule versions in this library release</summary>

| Rule | Version |
| --- | --- |
| practices/code-design/organize-code-by-feature | 1.1.0 |
| practices/testing/verify-backoff | 1.3.0 |
| practices/testing/verify-retries | 1.0.0 |
| practices/testing/verify-retry-limits | 2.0.0 |
| techs/react/prefer-server-components | 1.4.0 |
| techs/react/test-hooks-in-isolation | 2.2.0 |

</details>
```

Code Rules generates all of it from the change notes and the library's changes: the counts, the sections, which appear only when they have entries, the versions, a sentence saying the library release also updates shared files when it does, except in the first library release, which adds them all, and the table. The notes never list shared files by name; the tag's release record does. A library release that changes no rules opens by saying it updates shared files instead. Each summary is copied from its note, as an item under its rule. Every paragraph and list item is one line, because GitHub shows each line break in a GitHub Release page as a new line. The only judgment in it is yours, recorded in the notes before publishing: each change level, summary, and replacement.

To add a general message, such as an introduction to what this library release is about, edit the GitHub Release page on GitHub. The page is only the announcement. The tag message keeps the generated text, and it's what projects read when they update.

### After publishing

The tag is the permanent record, so leave it as it is. Fix a typo on the GitHub Release page directly. To correct something substantive, such as a misstated change, say so in a note in the next library release.

### Publish from CI

To publish from GitHub Actions instead of your machine, add a workflow that someone starts by hand:

```yaml
name: Library release
on: workflow_dispatch
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    env:
      GH_TOKEN: ${{ github.token }}
      GIT_COMMITTER_NAME: Code Rules Bot
      GIT_COMMITTER_EMAIL: code-rules-bot@noreply.invalid
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7
        with:
          fetch-depth: 0
      # Download and verify Code Rules, then add it to PATH.
      - name: Install Code Rules
        run: ...
      - run: code-rules library release
```

The job needs only permission to push tags. It tags as Code Rules Bot, set through Git's identity variables.

## Check changes in CI

`code-rules library init` creates `.github/workflows/code-rules.yml`, which runs `code-rules library check` on every pull request, using the Code Rules version that created it:

```yaml
name: Code Rules
on: pull_request
permissions:
  contents: read
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7
        with:
          fetch-depth: 0
          persist-credentials: false
      # Download and verify Code Rules, then add it to PATH.
      - name: Install Code Rules
        run: ...
      - run: code-rules library check
```

The install step downloads that version's Linux archive and `SHA256SUMS` from the Code Rules GitHub Release page, verifies that Code Rules' release workflow built `SHA256SUMS`, and installs the archive only when `SHA256SUMS` lists that exact archive with its checksum. A release whose files were replaced, even with another genuine release's, fails the step instead of installing the wrong version. The step prints the version it installed, such as `Installed Code Rules 0.2.0.`, in the job's log and summary.

The workflow keeps using that version until you change it; rerunning `code-rules library init` never changes an existing workflow. To check with a newer Code Rules, such as one that checks a feature your library started using, [upgrade Code Rules](/start-here/install/#upgrade), delete `.github/workflows/code-rules.yml`, and run `code-rules library init` again, which writes the workflow for the version you run. Review the difference and commit it.

`code-rules library check` needs every tag and the full history to compare rules with the latest library release, so the workflow checks out with `fetch-depth: 0`.

To make the check required before merging, add a branch ruleset in **Settings > Rules > Rulesets** that requires the `check` status check.

## Recover from a failed library release

| Problem | What to do |
| --- | --- |
| `code-rules library release` stopped after pushing the tag, such as when GitHub was unavailable. | Run it again. It finds the tag on the current commit and creates the GitHub Release page. |
| The remote refused the tag push, such as a server hook or a GitHub tag ruleset. | The error shows the server's reason: the lines its hook printed and Git's `[remote rejected]` line, with credentials hidden. Fix what it asks for, such as getting permission to create `release/<number>` tags, then run `code-rules library release` again. The command deleted the tag it created, so the rerun starts over. |
| A release tag points to a different commit. | Someone created or moved a `release/<number>` tag by hand. Don't move published tags; projects may have imported them. Ask the tag's author, then restore it to its original commit. |
| Check fails in a shallow clone. | Fetch the full history and tags, such as with `git fetch --unshallow --tags`, or `fetch-depth: 0` in CI. |
| Check fails because the clone has change notes but no release tags, such as a clone made with `git clone --no-tags`. | Fetch the tags with `git fetch --tags`, or check out with `fetch-depth: 0` in CI. Without them, the library would look as if it had never published a library release. |

## Next steps

- [Write a rule](/guides/write-rules/) covers authoring guidance.
- [Update rules](/guides/update/) shows how projects review and adopt your library releases.
- The [CLI reference](/reference/cli/#library-release) lists every option.
