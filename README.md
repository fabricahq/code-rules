<h1 align="center">Code Rules</h1>

<h3 align="center">The package manager for your engineering&nbsp;rules.</h3>

<p align="center">
  Write your best practices once. Version them, share them across repositories,
  and give every coding agent the same guidance <em>before</em> it writes&nbsp;code.
</p>

<p align="center">
  <a href="https://github.com/fabricahq/code-rules/actions/workflows/go.yml"><img alt="Go checks" src="https://github.com/fabricahq/code-rules/actions/workflows/go.yml/badge.svg"></a>
  <a href="LICENSE.md"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue"></a>
  <img alt="Platform: macOS | Linux" src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey">
</p>

<p align="center">
  <a href="https://code-rules.fabricahq.com">Docs</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="https://code-rules.fabricahq.com/start-here/overview/">Why Code Rules</a> ·
  <a href="https://code-rules.fabricahq.com/reference/cli/">CLI reference</a>
</p>

---

## What is Code Rules?

Code Rules is a **package manager** for engineering practices.

That means you can:

- Capture engineering best practices as individual Markdown files
- Organize and version them in a central Git repo
- Configure your projects to pull down just the right rules automatically when your agents write or validate code

## How is it useful?

Agents are capable of writing tested, maintainable, well-organized code. They just don't do it by default. They do it when you tell them how.

So every time your agent departs from your expectations, you write new guidance into `AGENTS.md`. Then you copy it into the next repo, tweak it, and copy it again. A few months later, every repository has its own slightly different version of your best practices, and nobody knows which one is current.

We solved this problem for code a long time ago with package managers. **Code Rules does the same for agent guidance.**

|                          | Copying `AGENTS.md` by hand            | Code Rules                                                        |
| ------------------------ | -------------------------------------- | ----------------------------------------------------------------- |
| **Sharing**              | Copy, paste, tweak, repeat             | Import versioned rule libraries from any Git repository           |
| **Versioning**           | No idea which copy a repo has          | Each library is pinned to a commit, with version constraints      |
| **Improvements**         | Stay in the repo where they were made  | Release once; projects pick them up on their next `sync`          |
| **Customizing**          | Edit the copy and hope nobody notices  | Exclude or replace individual rules in config, with a reason      |
| **Provenance & license** | Lost the moment you paste              | Source, resolved commit, and license terms travel with every rule |
| **Agent context**        | One ever-growing file                  | A generated index, so agents read only the rule groups they need  |

## Quick start

**1. Install** the standalone binary (macOS and Linux; no Go, Node.js, or Bun required):

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh
```

Or install with [Homebrew](https://brew.sh/):

```sh
brew install fabricahq/tap/code-rules
```

> [!NOTE]
> The first release, `v0.1.0`, is being prepared. Until it's published, install from source with Go 1.27.1 or later: `go install github.com/fabricahq/code-rules/cmd/code-rules@latest`

**2. Write a rule.** A rule is one practice, written in plain Markdown. Save this as `test-changed-behavior.md`:

```md
## Test changed behavior

When a change alters behavior that a caller or user relies on, add or update a test for that outcome.
Assert the observable result, such as an order's new total, rather than which private helper was called.
```

Then, from your repository root, set up Code Rules and add the rule to a group:

```sh
code-rules project init
code-rules project add group practices/testing \
  --name Testing \
  --description 'Tests for the behavior this project provides.' \
  --when-to-read 'When adding or changing behavior, fixing bugs, or reviewing tests.'
code-rules project add rule practices/testing/test-changed-behavior \
  --title 'Test changed behavior' \
  --when-to-read 'When adding or changing externally visible behavior.' \
  --impact HIGH \
  --impact-description 'Prevents behavior changes from silently breaking existing use cases.' \
  --body-file test-changed-behavior.md
```

Leave out `--body-file` to start from a template instead. Either way, the rule lives in `.code-rules/local/`, where you keep editing it.

**3. Build the guidance your agent reads:**

```console
$ code-rules project build
Build complete: 6 added, 0 changed, 0 removed.
Paths relative to .code-rules/generated:
  Add: RULES.md
  Add: groups/README.md
  Add: groups/practices/testing.md
  Add: provenance.json
  Add: rules/README.md
  Add: rules/local/practices/testing/test-changed-behavior.md
```

`RULES.md` is an index that tells agents which groups to open for the task at hand:

```md
### Testing

**Description:** Tests for the behavior this project provides.

**When to read this group:** When adding or changing behavior, fixing bugs, or reviewing tests.

**Open group:** [Testing](groups/practices/testing.md)
```

**4. Point your agent at it.** Add this to `AGENTS.md`, `CLAUDE.md`, or whichever instruction file your agent reads:

```markdown
## Engineering rules

Before planning, implementing, reviewing, testing, or debugging a change:

1. Read `.code-rules/generated/RULES.md` and follow its instructions to
   select relevant groups and read their rules in full, including linked
   files and additional index pages.
2. Follow the applicable rules and their exceptions while doing the work.
3. Before finishing, check your work against those rules and run the
   relevant validation. Briefly report what you verified and any gaps.

If required rule files are unavailable or give conflicting instructions,
report the issue rather than silently skipping them or choosing a policy.
```

Commit `.code-rules/` and you're done. Now ask your agent for a change as you normally would; you don't need to mention Code Rules in each request.

➡️ The full walkthrough is in [Set up your first project](https://code-rules.fabricahq.com/start-here/set-up-project/).

## Share rules across projects

Once a rule proves itself, move it into a **library**: a Git repository of rule groups that any project can import. Publish your team's defaults in a repository such as `acme/.code-rules`, then adopt them in each project:

```sh
code-rules project add library acme \
  --repository https://github.com/acme/.code-rules.git \
  --ref '>= 1.0.0, < 2.0.0' \
  --groups practices/testing \
  --groups techs/go
code-rules project sync
```

The dependency lands in `.code-rules/config.yaml`, the file you review and commit:

```yaml
schemaVersion: 1
sources:
  acme:
    repository: https://github.com/acme/.code-rules.git
    version: '>= 1.0.0, < 2.0.0'
    groups:
      - practices/testing
      - techs/go
    exclude: {}
    replace: {}
```

`sync` picks the newest release that satisfies the constraint, snapshots it into `.code-rules/vendor/`, and rebuilds. Improve a rule in the library and tag a release: every project whose range allows it picks up the improvement on its next sync, on its own schedule.

➡️ [Create your first library](https://code-rules.fabricahq.com/start-here/create-library/) · [Import rules](https://code-rules.fabricahq.com/guides/select-rules/) · [Update rules](https://code-rules.fabricahq.com/guides/update/)

## Features

- **Local rules and shared libraries.** Start with project-only rules; import libraries from any Git host, public or private, when you're ready.
- **Semantic version constraints.** Pin an exact tag or commit, or accept compatible releases with ranges like `>= 1.0.0, < 2.0.0`.
- **Customize without forking.** Exclude an imported rule or replace it with your own local definition, with the decision recorded in config.
- **Offline, reproducible builds.** Libraries are vendored at an exact commit, so `build` needs no network and every checkout sees the same rules.
- **Provenance and licenses included.** Every imported rule keeps its source, resolved commit, and declared license terms.
- **Guidance sized for agent context.** Groups carry "when to read" cues, so agents open only the rules that matter for the task.
- **CI-ready.** `code-rules project check` exits non-zero when generated files are stale, and `--json` gives structured output with no prompts.
- **Works with any agent.** Output is plain Markdown in your repository. If your agent reads a project instruction file, it can use Code Rules.

## What Code Rules doesn't do

- **It doesn't enforce your rules.** Code Rules delivers the same resolved rules to implementation and review, but giving an agent a rule doesn't guarantee that it follows it. `project check` verifies your rule files, not your application code. For a suggested plan, write, and review loop, see [For agents](https://code-rules.fabricahq.com/for-agents/).
- **It doesn't resolve contradictions for you.** If two rules disagree, you decide which to exclude or replace. See [Resolve conflicting rules](https://code-rules.fabricahq.com/guides/conflicting-guidance/).
- **It doesn't run on native Windows yet.** The Linux build is expected to work in WSL 2, but hasn't been tested end to end.

## Learn more

- → [**What is Code Rules?**](https://code-rules.fabricahq.com/start-here/overview/) The problem, and how rules, groups, projects, and libraries fit together
- → [**Install**](https://code-rules.fabricahq.com/start-here/install/) Versions, custom directories, checksum verification, and upgrades
- → [**Write a rule**](https://code-rules.fabricahq.com/guides/write-rules/) Write rules agents can actually follow
- → [**Configuration**](https://code-rules.fabricahq.com/reference/configuration/) Every field in `config.yaml`
- → [**CLI reference**](https://code-rules.fabricahq.com/reference/cli/) Every command and flag

## Contributing

Bug reports and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) covers building from source, running the validation suite, and working on the documentation site.

## License

Code Rules is [MIT licensed](LICENSE.md). Rule libraries carry their own license terms, which Code Rules preserves in each project that imports them.
