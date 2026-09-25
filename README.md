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

Agents are capable of writing testable, maintainable, and well-organized code. But they don't do it by default. They only do it when you tell them how.

### Traditional approaches to giving agents guidance

To solve the problem of "how," many teams use static methods to deliver guidance to their agents like **skills.** Skills are an excellent way to repeat the same guidance in the same situation, but the skills themselves rarely evolve with you based on the real-world feedback you give to agents. Skills are also coarsely-grained, making it hard to extract a subset of guidance from a skill to be used on a specific project.

### The Code Rules approach

The Code Rules philosophy is that the best approach to scaling agent guidance is to carefully consider one unit of guidance at a time. We call those units **[rules](https://code-rules.fabricahq.com/concepts/rule/)** and they are represented as Markdown files that optionally follow the [Code Rules rule template](https://code-rules.fabricahq.com/reference/rule-authoring/).

You can write your own project-specific rules, or pull them from **[libraries](https://code-rules.fabricahq.com/concepts/libraries/),** which are collections of rules meant for use by many projects. For example, see the [Fabrica Public Rules Library](https://github.com/fabricahq/public-rules).

Code Rules is agent- and harness-agnostic, so it will work with Claude Code, Codex, Grok, and everything else.

### Rules can cover anything

Rules can give guidance on **technologies** like Go or Typescript, or on **practices** like testing, observability, or even writing good READMEs. 

### Rules can evolve

As you work, you will find that some rules consistently deliver value, while others start to get in the way or no longer represent your preferred way of working. Or you may be repeatedly giving the same guidance to agents, in which case, it may be time to create a rule for it.

Ideally, you can ask your agent to automatically introspect your sessions and modify rules as needed, opening up the possibility for version-controlled, evolving guidance.

### Key use cases

Ultimately, Code Rules is useful for:

- **Implementation.** Your agents will write code according to your team's guidance.
- **Validation.** Your agents will review code against your team's guidance.

## Quick start

### Set up Code Rules

**1. Install** the standalone binary (macOS and Linux; no Go, Node.js, or Bun required):

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh
```

Or install with [Homebrew](https://brew.sh/):

```sh
brew install fabricahq/tap/code-rules
```

### Write your first rule

**2. Write a rule.** A rule is one practice, written in plain Markdown. Save this abbreviated rule as `test-changed-behavior.md`:

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

### Tell your agents to follow your rule

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

Once a rule proves itself, you may want move it into a **library**: a Git repository of rule groups that any project can import. 

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
