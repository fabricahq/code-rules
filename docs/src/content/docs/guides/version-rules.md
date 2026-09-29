---
title: "Version your rules"
description: "How rules change, how rule versions describe those changes, and how to publish new versions from your library."
---

Rules change over time. Some changes are small: an author fixes a typo, rewords a confusing sentence, or adds an example. Others are large: a rule becomes stricter, starts requiring something it previously only recommended, or is retired entirely.

Just as with code, you can track these changes by versioning your rules. This guide explains how rule versions work, then shows how to record changes and publish new versions from your library.

## Rule versions

A **rule version** identifies one state of a rule, such as `1.3.0`. When a rule changes, it gets a new version, along with a summary of what changed.

Each rule has its own version, even though one library repository holds many rules. Code Rules defines a scheme that gives each rule version its own Git tag, named after the rule: `<rule-id>@<version>`, such as `practices/testing/verify-retry-limits@1.3.0`. The tag's message summarizes the change. The rule file itself contains no version.

A version covers everything an agent needs to read the rule: its Markdown file, its own supporting files, and any shared files in the library that it links to. Changing a shared diagram therefore changes every rule that uses it.

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

Sometimes a rule shouldn't change; it should stop. **Retiring** a rule ends its history: the library stops publishing it, and its last version stays its final version. A retirement always says why, and there are two reasons:

- **Superseded:** a better rule replaces it. For example, you fold a narrow rule about retry limits into a broader rule about testing retries. The retirement names the replacement, so projects know what to adopt instead.
- **Withdrawn:** the practice is no longer recommended. For example, you published a rule about how agents should comment code, then concluded from feedback that agents shouldn't add those comments at all. There's no replacement; the summary explains why the advice was withdrawn.

Code Rules records a retirement with a `<rule-id>@retired` tag, whose message gives the reason, the summary, and any replacement. A retired rule's ID is never reused.

Retiring a rule a project uses is as disruptive as a major change, because its agents stop following the rule. So projects accept retirements the same way they accept major changes.

## How projects use rule versions

Projects import rules from your library. Each imported rule's version appears in the project's generated guidance, so agents and reviewers can cite the exact version they followed.

A project keeps the versions it imported until someone runs `code-rules project update`, which lists each rule that changed, with its old and new versions and your summaries. It applies patch and minor changes directly. It stops for major changes and retirements until someone on the project accepts them, because those can change what the project's code must do. See [Update rules](/guides/update/).

Projects can also choose versions rule by rule: hold one rule at its current version, or accept only compatible updates, while the rest moves forward. Projects import only released versions, unless one deliberately imports an exact commit, so changes waiting to be released don't reach them.

## Library releases

New rule versions reach projects through releases. A **library release** publishes all of a library's pending rule changes together: every rule that changed since the last release gets its new version at the same moment.

A release can be as small as a single rule change, published as soon as it's ready. It can also be a larger, meaningful collection of rule updates that you announce together and point users to as a snapshot of the library.

Releases are numbered 1, 2, 3, and so on. They don't use semantic versioning. One release contains many rules, each with its own version, so no single version number could describe the release: the same release might carry a patch to one rule and a major change to another. The release number only identifies the release. Compatibility is described rule by rule, by each rule's own version.

A release produces:

- a new version tag for each changed rule, such as `practices/testing/verify-retry-limits@1.3.0`,
- an update to the library's **release manifest**, `code-rules-release.yaml`, which lists every rule's version as of that release,
- a release tag, such as `release/3`, and
- on GitHub.com, a **GitHub Release page** that announces the release and lists its changes. The page is only an announcement; the release itself is the new versions and tags.

Release whenever you want your pending changes to reach projects. There's no schedule, and no need to wait for a major change: smaller releases are easier for projects to review. Projects don't have to take a whole release, either. Each project chooses versions rule by rule, and can import one specific release when it wants a known snapshot of the library.

## How releases work

You manage versions and releases with two Code Rules commands. You never create tags by hand.

- **`code-rules library change`** records a change to one rule. You say how large the change is (major, minor, or patch) and summarize it. The command saves this in a **change note**, a small file beside the rule.
- **`code-rules library release`** publishes a [library release](#library-releases). It turns every pending change note into a new rule version, updates the release manifest, removes the notes, and creates the tags and the GitHub Release page.

A third command, `code-rules library check`, confirms that every changed rule has a change note, and previews the versions the next release will publish.

You can run `code-rules library release` yourself, or let a GitHub Actions workflow run it for you. With the workflow, a typical cycle looks like this:

1. You edit a rule and run `code-rules library change` in the same pull request. The workflow runs `code-rules library check` on the pull request.
2. After the pull request merges, the workflow runs `code-rules library release --pr`. It opens a "Release rules" pull request, or updates the open one. That pull request updates the release manifest and deletes the notes, so its diff shows exactly what the release will publish.
3. When you merge the release pull request, the workflow runs `code-rules library release --publish`, which creates the tags and the GitHub Release page.

Change notes start with the second release. Your library's [first release](#publish-the-first-release) gives every rule version `1.0.0`, so it needs no notes. For the exact tag and note formats, see [Rule versions](/reference/rule-versions/) and [Change notes](/reference/rule-versions/#change-notes).

## Publish the first release

Before your library's first release, its rules have no versions, so you don't need change notes. When the rules are ready for projects to use, publish the first release the same way as every later one:

- **With the release [GitHub Actions workflow](#automate-releases-with-github-actions),** merge the "Release rules" pull request. Before the first release, the workflow keeps it open, and its only change is creating the release manifest. Leave it unmerged until your rules are ready.
- **Without the workflow,** run `code-rules library release` from your machine. See [Release from your machine](#release-from-your-machine).

The first release gives every rule version `1.0.0`.

From then on, [record every change with a note](#change-rules-after-the-first-release).

## Change rules after the first release

After the first release, every change to a rule needs a change note, recorded with `code-rules library change` in the same pull request as the change. The steps differ slightly for each kind of change.

### Update a rule

Suppose you want to add a Python example to `practices/testing/verify-retry-limits`. First, edit the rule's Markdown file as usual. Then choose the change level, as described in [Semantic versions](#semantic-versions), and record it with `code-rules library change`:

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

Write the summary for someone deciding whether to update: say what changed in the obligation or guidance, not how you edited the file. If the rule already has a pending note, `code-rules library change` keeps the larger change and adds your summary as another line.

### Add a rule

Create the rule as usual:

```sh
code-rules library add rule practices/testing/verify-backoff \
  --title 'Verify retry backoff' \
  --when-to-read 'When adding or changing retry delays.' \
  --impact MEDIUM \
  --impact-description 'Prevents retries from overloading a struggling service.'
```

Then record it. A new rule always starts at version `1.0.0`, so leave out `--bump`:

```sh
code-rules library change practices/testing/verify-backoff \
  --summary 'Add the rule.'
```

### Retire a rule

Delete the rule's Markdown file and its asset directory, then record the retirement with its reason.

When a better rule replaces it, retire it as superseded and name the replacement:

```sh
code-rules library change practices/testing/verify-retry-limits \
  --retire superseded \
  --replaced-by practices/testing/verify-retries \
  --summary 'Covered by the broader rule about testing retries.'
```

The replacement must exist in the library by the time the retirement is released. It can be a new rule in the same release.

When the practice itself is no longer recommended, retire it as withdrawn, and explain why in the summary:

```sh
code-rules library change practices/code-design/comment-intent \
  --retire withdrawn \
  --summary "Withdrawn after feedback that agents shouldn't add explanatory comments."
```

The release creates the rule's `@retired` tag. A rule that was never released can't be retired; delete it and its note together.

### Rename a rule

A rename changes the rule's ID. Move the file, add a new-rule note for the new ID, and retire the old ID as superseded by the new one. Projects see the old rule retired, with the new rule as its replacement.

### Change a shared file

A file in the library's shared `assets/` directory, such as a diagram, is part of every rule that links to it. After editing one, run `code-rules library check`: it names each rule that uses the file and still needs a note. Record a change for each, choosing the change level by its effect on that rule. A clarified diagram is usually a patch.

### Check your changes

Whatever you changed, check the library before opening your pull request:

```sh
code-rules library check
```

`code-rules library check` fails if a changed rule has no note, or if a note no longer matches a change. When it passes, it previews the pending release:

```text
Pending release
  practices/testing/verify-retry-limits  minor  1.2.0 -> 1.3.0
```

Commit each rule and its note together.

## Automate releases with GitHub Actions

`code-rules library init` creates `.github/workflows/code-rules.yml`. It installs the Code Rules version that created it, then:

- **On pull requests,** runs `code-rules library check` with the repository's full history.
- **On pushes to `main`,** runs `code-rules library release --publish`, then `code-rules library release --pr`, then starts the check on the release pull request.

Each release command does nothing when it has nothing to do. `code-rules library release --publish` acts only when `main` contains a release that isn't tagged yet, however the release pull request was merged. `code-rules library release --pr` opens or updates the release pull request while notes are pending, and closes it when none are. The workflow runs one release job at a time, and a job started by an older commit of `main` steps aside for the newer one.

The workflow uses the built-in `GITHUB_TOKEN`; you don't need to create a token. GitHub doesn't start workflows for a pull request that this token opens or updates, so the release job starts the check itself, with a manual run (`workflow_dispatch`) on the release pull request's branch. That run reports its result on the pull request like any other check. The job sets Git's identity variables so releases are committed and tagged as Code Rules Bot; change them to use another name. The release job looks like this, with the install step shortened:

```yaml
release:
  if: github.event_name == 'push'
  runs-on: ubuntu-latest
  permissions:
    actions: write
    contents: write
    pull-requests: write
  env:
    GH_TOKEN: ${{ github.token }}
    # Commit and tag releases as Code Rules Bot.
    GIT_AUTHOR_NAME: Code Rules Bot
    GIT_AUTHOR_EMAIL: code-rules-bot@noreply.invalid
    GIT_COMMITTER_NAME: Code Rules Bot
    GIT_COMMITTER_EMAIL: code-rules-bot@noreply.invalid
  steps:
    - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      with:
        fetch-depth: 0
    # Download Code Rules, verify its attestation and checksum, and add it to PATH.
    - name: Install Code Rules
      run: ...
    - name: Publish a merged release
      run: code-rules library release --publish
    - name: Open or update the release pull request
      run: code-rules library release --pr
    - name: Check the release pull request
      run: |
        if [ "$(gh pr view code-rules/release-pr --json state --jq .state)" = OPEN ]; then
          gh workflow run code-rules.yml --ref code-rules/release-pr
        fi
```

`code-rules library check` needs every tag and the full history to compare rules with their last release, so the workflow checks out with `fetch-depth: 0`.

### Configure the repository

Configure three settings on GitHub. Each step shows the setting's place in the repository's settings, and a [GitHub CLI](https://cli.github.com/) command that applies it. Run the commands from your library's checkout; `gh api` fills in `{owner}` and `{repo}` from it.

1. **Require the check, on an up-to-date branch, before merging to `main`.** Then the release pull request always includes every pending note: if a newer note reaches `main` first, the release pull request must be updated before it can merge. In **Settings > Rules > Rulesets**, add a branch ruleset for the default branch that requires the `check` status check and requires branches to be up to date.

   To do this with the GitHub CLI, run:

   ```sh
   gh api --method POST 'repos/{owner}/{repo}/rulesets' --input - <<'EOF'
   {
     "name": "Code Rules checks",
     "target": "branch",
     "enforcement": "active",
     "conditions": {"ref_name": {"include": ["~DEFAULT_BRANCH"], "exclude": []}},
     "rules": [{
       "type": "required_status_checks",
       "parameters": {
         "strict_required_status_checks_policy": true,
         "required_status_checks": [{"context": "check", "integration_id": 15368}]
       }
     }]
   }
   EOF
   ```

   `check` is the name of the workflow's check job. `15368` is the ID of the GitHub Actions app, so only the workflow can report that check.

2. **Let the workflow open pull requests.** In **Settings > Actions > General**, enable **Allow GitHub Actions to create and approve pull requests**.

   To do this with the GitHub CLI, run:

   ```sh
   gh api --method PUT 'repos/{owner}/{repo}/actions/permissions/workflow' \
     -f default_workflow_permissions=read \
     -F can_approve_pull_request_reviews=true
   ```

   The workflow asks for the write permissions it needs itself, so the default can stay read-only. If your organization disables this setting, an organization owner must allow it first.

3. **Let the workflow push release tags and its branch.** Check that no ruleset covers tags under `techs/`, `practices/`, or `release/`, or branches under `code-rules/`. Review them in **Settings > Rules > Rulesets**.

   To list them with the GitHub CLI, run:

   ```sh
   gh api 'repos/{owner}/{repo}/rulesets' --jq '.[] | {id, name, target}'
   ```

   Show one ruleset's conditions with `gh api 'repos/{owner}/{repo}/rulesets/ID' --jq .conditions`, replacing `ID`. If a ruleset covers those tags or branches, exclude them from it in **Settings > Rules > Rulesets**.

### Review and merge the release pull request

The release pull request, from the branch `code-rules/release-pr`, updates the release manifest and deletes every pending note. The manifest's diff shows every rule's next version. The description lists each rule with its change, its current and next version, its summary, and the pull request that added the note. Review the major changes carefully: every project that uses those rules has to accept them before updating.

Merge it when you want to publish, using whichever merge method your repository allows. The workflow then tags the new rule versions and the release, and creates the release's GitHub Release page. Projects see the new versions the next time they run `code-rules project update`.

## Release from your machine

If maintainers push directly to `main` instead of using pull requests, release locally from an up-to-date checkout of `main`. Preview first:

```sh
code-rules library release --dry-run
```

The preview shows the repository, branch, commit, release number, and each rule's change and versions. Then release:

```sh
code-rules library release
```

The command fetches from the remote first. It refuses unless you're on the remote's default branch, your branch matches the remote exactly, your working tree has no uncommitted changes, and `code-rules library check` passes. It then creates the release commit, pushes it with the tags in one atomic push, and creates the GitHub Release page with the [GitHub CLI](https://cli.github.com/). Because it pushes to `main`, run it as someone allowed to push there.

If the command stops partway, such as on a network error, run it again: it finishes publishing the release it already started. If you don't want a GitHub Release page, or the library isn't hosted on GitHub.com, run `code-rules library release --no-github-release`; repositories hosted elsewhere get tags only.

## Recover from a failed release

| Problem | What to do |
| --- | --- |
| `code-rules library release --publish` stopped partway, such as on a network error. | Rerun the workflow job. The command keeps tags it already created and adds anything missing. |
| `code-rules library release` stopped partway on your machine. | Run it again. It finds the unpublished release commit and finishes publishing it. |
| Publishing refused because the release commit on `main` still contains notes. | A note reached `main` without being included in the release pull request. Revert the release pull request's merge. The workflow then opens a new release pull request with every pending note. To prevent this, require branches to be up to date before merging. |
| A tag points to a different commit. | Someone created or moved a rule tag by hand. Don't move published tags; projects may have imported them. Ask the tag's author, then restore it to its original commit. |
| Check fails in a shallow clone. | Fetch the full history and tags, such as with `git fetch --unshallow --tags`, or `fetch-depth: 0` in CI. |

## Next steps

- [Write a rule](/guides/write-rules/) covers authoring guidance.
- [Update rules](/guides/update/) shows how projects review and adopt your releases.
- The [CLI reference](/reference/cli/#library-release) lists every release option.
