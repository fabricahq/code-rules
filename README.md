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

Your agents then follow the same rules whether they're writing code or reviewing it.

## How is it useful?

Agents are capable of writing testable, maintainable, and well-organized code. But they don't do it by default. They only do it when you tell them how.

### Traditional approaches to giving agents guidance

To solve the problem of "how," most teams start by writing guidance into an instruction file like `AGENTS.md` or `CLAUDE.md`. That works for one repository, but the file keeps growing, and when you copy it into the next repo and tweak it, the copies drift apart until nobody knows which one is current.

Many teams also deliver guidance through **skills**. Skills are an excellent way to repeat the same guidance in the same situation, but the skills themselves rarely evolve with you based on the real-world feedback you give to agents. Skills are also coarse-grained, making it hard to use a subset of a skill's guidance on a specific project.

### The Code Rules approach

The Code Rules philosophy is that the best approach to scaling agent guidance is to carefully consider one unit of guidance at a time. We call those units **[rules](https://code-rules.fabricahq.com/concepts/rule/)**, and they are represented as Markdown files that optionally follow the [Code Rules rule template](https://code-rules.fabricahq.com/reference/rule-authoring/). Rules can give guidance on **technologies** like Go or TypeScript, or on **practices** like testing, observability, or even writing good READMEs.

You can write your own project-specific rules, or pull them from **[libraries](https://code-rules.fabricahq.com/concepts/libraries/)**, which are collections of rules meant for use by many projects. For example, see the [Fabrica Public Rules Library](https://github.com/fabricahq/public-rules).

That's where the package manager comes in. As with packages for code, you can:

- **Pin versions.** Pin each library to an exact tag or commit, or accept compatible releases with ranges like `>= 1.0.0, < 2.0.0`.
- **Customize without forking.** Exclude an imported rule or replace it with your own, with the decision recorded in config.
- **Build reproducibly.** Libraries are vendored at an exact commit, and every imported rule keeps its source and license terms.
- **Catch drift in CI.** `code-rules project check` fails when generated files are out of date.

Agents don't read every rule on every task. Code Rules generates an index with a "when to read" cue for each group of rules, so agents open only the rules that matter for the work at hand. It works with any agent that reads a project instruction file, such as Claude Code, Codex, Cursor, or Gemini CLI.

### Rules can evolve

As you work, you will find that some rules consistently deliver value, while others start to get in the way or no longer represent your preferred way of working. Or you may be repeatedly giving the same guidance to agents, in which case, it may be time to create a rule for it.

Because rules are files in Git, you can ask your agent to review a session and propose a rule change, then review that change like any other code. Your guidance improves in version-controlled steps, and every project that uses the rule picks up the improvement on its next sync.

## Quick start

### Set up Code Rules

Install the standalone binary (macOS and Linux; no Go, Node.js, or Bun required):

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh
```

Or install with [Homebrew](https://brew.sh/):

```sh
brew install fabricahq/tap/code-rules
```

Then set up Code Rules from your repository root:

```sh
code-rules project init
```

### Write your first rule

A rule is one practice, written in plain Markdown. Save this abbreviated rule as `test-changed-behavior.md`:

```md
## Test changed behavior

When a change alters behavior that a caller or user relies on, add or update a test for that outcome.
Assert the observable result, such as an order's new total, rather than which private helper was called.
```

Then add the rule to a group:

```sh
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

### Tell your agents to follow your rule

Build the guidance your agent reads:

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

Then point your agent at it. Add this to `AGENTS.md`, `CLAUDE.md`, or whichever instruction file your agent reads:

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

Once a rule proves itself, you may want to move it into a **library**: a Git repository of rule groups that any project can import.

We recommend publishing your team's default rules in a repository such as `acme/.code-rules`, or starting from an existing library like the [Fabrica Public Rules Library](https://github.com/fabricahq/public-rules):

```sh
code-rules project add library fabrica \
  --repository https://github.com/fabricahq/public-rules.git \
  --ref '>= 1.0.0, < 2.0.0' \
  --groups practices/testing \
  --groups techs/go
code-rules project sync
```

The dependency lands in `.code-rules/config.yaml`, the file you review and commit:

```yaml
schemaVersion: 1
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    version: '>= 1.0.0, < 2.0.0'
    groups:
      - practices/testing
      - techs/go
    exclude: {}
    replace: {}
```

`sync` picks the newest release that satisfies the constraint, snapshots it into `.code-rules/vendor/`, and rebuilds. Improve a rule in the library and tag a release: every project whose range allows it picks up the improvement on its next sync, on its own schedule.

➡️ [Create your first library](https://code-rules.fabricahq.com/start-here/create-library/) · [Import rules](https://code-rules.fabricahq.com/guides/select-rules/) · [Update rules](https://code-rules.fabricahq.com/guides/update/)

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
