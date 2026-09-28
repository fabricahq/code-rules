---
title: "Version your rules"
description: "How rules change, how rule versions describe those changes, and how to publish new versions from your library."
---

Rules change over time. Some changes are small: an author fixes a typo, rewords a confusing sentence, or adds an example. Others are large: a rule becomes stricter, starts requiring something it only recommended, or is retired entirely.

Just as with code, you can track these changes by versioning your rules. This guide explains how rule versions work, then shows how to record changes and publish new versions from your library.

## Rule versions

A **rule version** identifies one state of a rule, such as `1.3.0`. When a rule changes, it gets a new version, along with a summary of what changed.

Each rule has its own version, even though one library repository holds many rules. Code Rules defines a scheme that gives each rule version its own Git tag, named after the rule: `<rule-id>@<version>`, such as `practices/testing/verify-retry-limits@1.3.0`. The tag's message summarizes the change, and on GitHub each tag also gets a GitHub Release. The rule file itself contains no version.

## Semantic versions

Rule versions are [semantic versions](https://semver.org/): three numbers, `MAJOR.MINOR.PATCH`. Which number increases tells you how large the change was. Software uses these numbers to describe changes to an API. For rules, they describe changes to the rule's **obligation**: what work must do to follow it.

| Change | What it means | Example |
| --- | --- | --- |
| **Major**, such as `1.3.0` to `2.0.0` | A breaking change: the obligation became stricter or different, so work that followed the previous version could fail this one. | Lower the required retry limit, or require a test the rule previously only recommended. |
| **Minor**, such as `1.3.0` to `1.4.0` | New guidance that no previously compliant work can fail. | Add a Python example of the same test. |
| **Patch**, such as `1.3.0` to `1.3.1` | Clearer wording or fixed examples, with the same obligation. | Fix a misleading sentence or a typo in an example. |

To decide, ask one question: could work that followed the previous version fail this one? If yes, the change is major. Watch for changes that only add text but widen where the rule applies: code in the newly covered situation may not follow it, which makes the change major. When unsure, choose the larger change.

A few more conventions:

- A new rule starts at `1.0.0`.
- Removing a rule ends its history at its last version.
- Renaming or moving a rule changes its ID, so it counts as removing the old rule and adding a new one.

## How projects use rule versions

Projects import rules from your library. Each imported rule's version appears in the project's generated guidance, so agents and reviewers can cite the exact version they followed.

A project keeps the versions it imported until someone runs `code-rules project update`. Update lists each rule that changed, with its old and new versions and your summaries. It applies patch and minor changes directly. It stops for major changes and removals until someone on the project accepts them, because those can require changes to the project's code. See [Update rules](/guides/update/).

## How releases work

You don't create version tags by hand. Instead, you describe each change in a **change note** beside the rule, and a **release** turns the pending notes into new versions, all tagged on one commit. For the exact formats, see [Rule versions](/reference/rule-library-format/#rule-versions) and [Change notes](/reference/rule-library-format/#change-notes).

A library opts in to rule versions with `versioning: rules` in `rule-library.yaml`. Libraries created with `code-rules library init` do. For an older library, see [Adopt rule versions in an existing library](#adopt-rule-versions-in-an-existing-library).

A typical release goes like this:

1. You change a rule and add a change note beside it, in the same pull request.
2. `code-rules library check` confirms that every changed rule has a note.
3. After the pull request merges, notes wait on `main`. The release workflow keeps one "Release rules" pull request up to date. It deletes every pending note and lists the versions the release will publish.
4. Merging the release pull request approves the release. The workflow tags a new version for each rule on that merge commit and creates a GitHub Release for each tag.

The merge commit is a **release commit**. It contains no pending notes, so each rule's content matches its newest version. Projects import only release commits, so unreleased changes on `main` never reach them.

## Record a change

Edit the rule as usual. Then choose its change level, as described in [Semantic versions](#semantic-versions), and record it with `library change`:

```sh
code-rules library change practices/testing/verify-retry-limits \
  --bump minor \
  --summary 'Add a Python example of the retry-limit test.'
```

This writes `practices/testing/verify-retry-limits.change.yaml`:

```yaml
bump: minor
summary: Add a Python example of the retry-limit test.
```

Write the summary for someone deciding whether to update: say what changed in the obligation or guidance, not how you edited the file. If the rule already has a pending note, `library change` keeps the larger change and adds your summary as another line.

Check the library before opening your pull request:

```sh
code-rules library check
```

Check fails if a changed rule has no note, or if a note no longer matches a change. When it passes, it previews the pending release:

```text
Pending release
  practices/testing/verify-retry-limits  minor  1.2.0 -> 1.3.0
```

Commit the rule and its note together.

### Add a rule

A new rule needs a note without a `bump`, because its first version is always `1.0.0`:

```sh
code-rules library add rule practices/testing/verify-backoff \
  --title 'Verify retry backoff' \
  --when-to-read 'When adding or changing retry delays.' \
  --impact MEDIUM \
  --impact-description 'Prevents retries from overloading a struggling service.'
code-rules library change practices/testing/verify-backoff \
  --summary 'Add the rule.'
```

### Remove or rename a rule

Delete the rule's Markdown file and its asset directory, then record the removal:

```sh
code-rules library change practices/testing/verify-retry-limits \
  --removed \
  --summary 'Replaced by practices/testing/verify-retries.'
```

Removals create no tag; the rule's last version stays its final version. Projects that update past the release see the rule reported as removed, and must accept the removal like a major change.

A rename changes the rule's ID. Record it as a removal of the old ID and a new rule at the new ID. A rule that was never released needs no removal note; delete it and its note together.

## Automate releases with GitHub Actions

`code-rules library init` creates `.github/workflows/code-rules.yml` in a library that versions rules. It installs the Code Rules version that created it, then:

- **On pull requests,** runs `code-rules library check` with the repository's full history.
- **On pushes to `main`,** runs `code-rules library release --publish`, then `code-rules library release --pr`.

Each release command does nothing when it has nothing to do. `--publish` acts only on the merge of the release pull request. `--pr` opens or updates the release pull request while notes are pending, and closes it when none are. The release job looks like this, with the install step shortened:

```yaml
release:
  if: github.event_name == 'push'
  runs-on: ubuntu-latest
  permissions:
    contents: write
    pull-requests: write
  env:
    GH_TOKEN: ${{ secrets.CODE_RULES_RELEASE_TOKEN || github.token }}
  steps:
    - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      with:
        fetch-depth: 0
        token: ${{ secrets.CODE_RULES_RELEASE_TOKEN || github.token }}
    # Download Code Rules, verify its attestation and checksum, and add it to PATH.
    - name: Install Code Rules
      run: ...
    - name: Set the commit author
      run: |
        git config user.name 'github-actions[bot]'
        git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
    - name: Publish a merged release
      run: code-rules library release --publish
    - name: Open or update the release pull request
      run: code-rules library release --pr
```

`library check` needs every tag and the full history to compare rules with their last release, so the workflow checks out with `fetch-depth: 0`.

### Configure the repository

In the repository's settings on GitHub:

1. **Require branches to be up to date before merging** on `main`. Then the release pull request always includes every pending note. If a newer note reaches `main` first, the release pull request must be updated before it can merge.
2. **Let the workflow open pull requests.** Either enable **Allow GitHub Actions to create and approve pull requests**, or add a token as described next.
3. **Let the workflow push tags and the `code-rules/released` branch.** Exempt the workflow from any rulesets that protect tags under `techs/` and `practices/`, or that branch.

Pull requests opened with the workflow's default token don't start other workflows, so required checks never run on the release pull request. If `main` requires status checks, create a [GitHub App](https://docs.github.com/en/apps/creating-github-apps) or a fine-grained personal access token with write access to contents and pull requests. Save it as the repository secret `CODE_RULES_RELEASE_TOKEN`; the workflow uses it when present.

### Review and merge the release pull request

The release pull request, from the branch `code-rules/release-pr`, deletes every pending note. Its description lists each rule with its change, its current and next version, its summary, and the pull request that added the note. Review the major changes carefully: every project that uses those rules has to accept them before updating.

Merge it when you want to publish. The workflow then tags the new versions on the merge commit, moves `code-rules/released` to it, and creates the GitHub Releases. Projects see the new versions the next time they run `code-rules project update`.

## Release from your machine

If maintainers push directly to `main` instead of using pull requests, release locally. Preview first:

```sh
code-rules library release --dry-run
```

Then release:

```sh
code-rules library release
```

Release refuses when your working tree has uncommitted changes, your branch is behind its upstream, or `library check` fails. It deletes the notes, commits `Release N rules`, tags the new versions, pushes the commit, tags, and `code-rules/released` in one atomic push, and creates GitHub Releases with the [GitHub CLI](https://cli.github.com/). If you don't want GitHub Releases, or the library isn't hosted on GitHub.com, pass `--no-github-release`; repositories hosted elsewhere get tags only.

## Make a clean break

To rework a library so thoroughly that no rule's previous version still applies, give every rule a major change in one release:

```sh
code-rules library release --bump-all major \
  --summary 'Rewrite the library for the new service architecture.'
```

This merges a major change into every rule's note, then releases from your machine. Every project that updates must accept the release.

## Adopt rule versions in an existing library

A library that publishes tags such as `v1.2.0` for the whole library can switch to rule versions once. From an up-to-date checkout of `main` with no change notes:

```sh
code-rules library release --init --dry-run
code-rules library release --init
```

`--init` adds `versioning: rules` to `rule-library.yaml`, commits it, tags every rule at `1.0.0`, and pushes the commit, tags, and `code-rules/released` together. Because it pushes to `main`, run it as someone allowed to push there. It creates no GitHub Releases.

Then add the release workflow:

```sh
code-rules library init
```

Init keeps your existing files and adds the workflow if it's missing. Commit it.

Existing tags such as `v1.2.0` stay in place, and projects pinned to them keep working. To follow rule versions, a project removes the source's `ref` from `.code-rules/config.yaml` and runs `code-rules project sync`. Tell your consumers when you switch.

## Recover from a failed release

| Problem | What to do |
| --- | --- |
| Publishing stopped partway, such as on a network error. | Rerun the workflow job. Publishing keeps tags it already created and adds anything missing. |
| Publishing refused because the merge commit still contains notes. | A note reached `main` without being included in the release pull request. Revert the release pull request's merge. The workflow then opens a new release pull request with every pending note. To prevent this, require branches to be up to date before merging. |
| A tag points to a different commit. | Someone created or moved a rule tag by hand. Don't move published tags; projects may have imported them. Ask the tag's author, then restore it to its original commit. |
| Check fails in a shallow clone. | Fetch the full history and tags, such as with `git fetch --unshallow --tags`, or `fetch-depth: 0` in CI. |

## Next steps

- [Write a rule](/guides/write-rules/) covers authoring guidance.
- [Update rules](/guides/update/) shows how projects review and adopt your releases.
- The [CLI reference](/reference/cli/#library-release) lists every release option.
