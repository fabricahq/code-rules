<h1 align="center">Syllabus</h1>

<h3 align="center">The package manager for your engineering&nbsp;rules.</h3>

<p align="center">
  Write your best practices once. Version them, share them across repositories,
  and give every coding agent the same guidance <em>before</em> it writes&nbsp;code.
</p>

<p align="center">
  <a href="https://github.com/fabricahq/syllabus/actions/workflows/go.yml"><img alt="Go checks" src="https://github.com/fabricahq/syllabus/actions/workflows/go.yml/badge.svg"></a>
  <a href="LICENSE.md"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue"></a>
  <img alt="Platform: macOS | Linux" src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey">
</p>

<p align="center">
  <a href="https://code-rules.fabricahq.com">Docs</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="https://code-rules.fabricahq.com/start-here/overview/">Why Syllabus</a> ·
  <a href="https://code-rules.fabricahq.com/reference/cli/">CLI reference</a>
</p>

---

## What is Syllabus?

Syllabus is a **package manager** for engineering practices.

That means you can:

- Capture engineering best practices as individual Markdown files
- Organize and version them in a central Git repo
- Configure each project to import just the rules it needs, so your agents read the right ones when they write or review code

Your agents then follow the same rules whether they're writing code or reviewing it.

## How is it useful?

Agents are capable of writing testable, maintainable, and well-organized code. But they don't do it by default. They only do it when you tell them how.

### Traditional approaches to giving agents guidance

To solve the problem of "how," most teams start by writing guidance into an instruction file like `AGENTS.md` or `CLAUDE.md`. That works for one repository, but the file keeps growing, and when you copy it into the next repo and tweak it, the copies drift apart until nobody knows which one is current.

Many teams also deliver guidance through **skills**. Skills are an excellent way to repeat the same guidance in the same situation, but the skills themselves rarely evolve with you based on the real-world feedback you give to agents. Skills are also coarse-grained, making it hard to use a subset of a skill's guidance on a specific project.

### The Syllabus approach

The Syllabus philosophy is that the best approach to scaling agent guidance is to carefully consider one unit of guidance at a time. We call those units **[rules](https://code-rules.fabricahq.com/concepts/rule/)**, and they are represented as Markdown files that optionally follow the [Syllabus rule template](https://code-rules.fabricahq.com/reference/rule-authoring/). Rules can give guidance on **technologies** like Go or TypeScript, or on **practices** like testing, observability, or even writing good READMEs.

You can write your own project-specific rules, or pull them from **[libraries](https://code-rules.fabricahq.com/concepts/libraries/)**, which are collections of rules meant for use by many projects. For example, see the [Fabrica Public Rules Library](https://github.com/fabricahq/public-rules).

Pulling many files from versioned libraries into a project, and keeping them up to date, is a familiar problem. Programming languages have solved it for more than 20 years with the humble package manager. Syllabus applies that same approach, not to reusable code, but to agent guidance.

As with packages for code, you can:

- **Version every rule.** Each rule has its own semantic version, so an update shows exactly which rules changed and whether a change could break code that followed them. Projects record the exact version of every rule they import, and accept major changes explicitly.
- **Customize without forking the library.** Exclude an imported rule or replace it with your own, with the decision recorded in config.
- **Build reproducibly.** Every imported rule is vendored at its exact version, and keeps its source and license terms.
- **Catch drift in CI.** `syllabus project check` fails when generated files are out of date.

Agents don't read every rule on every task. Syllabus generates an index with a "when to read" cue for each group of rules, so agents open only the rules that matter for the work at hand. It works with any agent that reads a project instruction file, such as Claude Code, Codex, Cursor, or Gemini CLI.

### Rules can evolve

As you work, you will find that some rules consistently deliver value, while others start to get in the way or no longer represent your preferred way of working. Or you may be repeatedly giving the same guidance to agents, in which case, it may be time to create a rule for it.

Because rules are files in Git, you can ask your agent to review a session and propose a rule change, then review that change like any other code. Your guidance improves in version-controlled steps, and once a library release publishes the change, every project that uses the rule picks it up on its next `syllabus project update`.

## Quick start

### Set up Syllabus

Install the standalone binary (macOS and Linux; no Go, Node.js, or Bun required):

```sh
curl -fsSL https://code-rules.fabricahq.com/install.sh | sh
```

Or install with [Homebrew](https://brew.sh/):

```sh
brew install fabricahq/tap/syllabus
```

Then set up Syllabus from your repository root:

```sh
syllabus project init
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
syllabus project add group practices/testing \
  --name Testing \
  --description 'Tests for the behavior this project provides.' \
  --when-to-read 'When adding or changing behavior, fixing bugs, or reviewing tests.'
syllabus project add rule practices/testing/test-changed-behavior \
  --title 'Test changed behavior' \
  --when-to-read 'When adding or changing externally visible behavior.' \
  --impact HIGH \
  --impact-description 'Prevents behavior changes from silently breaking existing use cases.' \
  --body-file test-changed-behavior.md
```

Leave out `--body-file` to start from a template instead. Either way, the rule lives in `.syllabus/rules/`, where you keep editing it.

### Tell your agents to follow your rule

Build the guidance your agent reads:

```console
$ syllabus project build
Build complete: 6 added, 0 changed, 0 removed.
Paths relative to .syllabus/generated:
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

1. Read `.syllabus/generated/RULES.md` and follow its instructions to
   select relevant groups and read their rules in full, including linked
   files and additional index pages.
2. Follow the applicable rules and their exceptions while doing the work.
3. Before finishing, check your work against those rules and run the
   relevant validation. Briefly report what you verified and any gaps.

If required rule files are unavailable or give conflicting instructions,
report the issue rather than silently skipping them or choosing a policy.
```

Commit `.syllabus/` and you're done. Now ask your agent for a change as you normally would; you don't need to mention Syllabus in each request.

➡️ The full walkthrough is in [Set up your first project](https://code-rules.fabricahq.com/start-here/set-up-project/).

## Share rules across projects

Once a rule proves itself, you may want to move it into a **library**: a Git repository of rule groups that any project can import. A library keeps its rules in `.syllabus/rules/`, the same layout a project uses, with its manifest in the `library` section of `.syllabus/config.yaml`, so one repository can be both.

We recommend publishing your team's default rules in a repository such as `acme/.syllabus`, or starting from an existing library like the [Fabrica Public Rules Library](https://github.com/fabricahq/public-rules):

```sh
syllabus project add source fabrica \
  --repository https://github.com/fabricahq/public-rules.git \
  --rules 'practices/testing/*' \
  --rules 'techs/go/*'
syllabus project sync
```

The dependency lands in `.syllabus/config.yaml`, the file you review and commit:

```yaml
schemaVersion: 2
skillTargets: [claude, agents]
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    rules:
      include:
        - practices/testing/*
        - techs/go/*
```

`syllabus project sync` imports each rule's newest version, snapshots them into `.syllabus/vendor/` with their versions, and rebuilds. Improve a rule in the library and publish it in a library release: each project picks up the improvement when it runs `syllabus project update`, on its own schedule. Update previews every changed, new, and retired rule, and applies them only after you confirm.

➡️ [Create your first library](https://code-rules.fabricahq.com/start-here/create-library/) · [Import rules](https://code-rules.fabricahq.com/guides/select-rules/) · [Update rules](https://code-rules.fabricahq.com/guides/update/)

## What Syllabus doesn't do

- **It doesn't enforce your rules.** Syllabus gives your agents the same rules for writing and reviewing code, but giving an agent a rule doesn't guarantee that it follows it. `syllabus project check` verifies your rule files, not your application code.
- **It doesn't resolve contradictions automatically.** If two rules disagree, you decide which to exclude or replace. [Resolve conflicting rules](https://code-rules.fabricahq.com/guides/conflicting-guidance/) gives you a prompt that has your agent find the conflicts and propose fixes for you to review.
- **It doesn't run on native Windows yet.** The Linux build is expected to work in WSL 2, but hasn't been tested end to end.

## Learn more

- → [**What is Syllabus?**](https://code-rules.fabricahq.com/start-here/overview/) The problem, and how rules, groups, projects, and libraries fit together
- → [**Install**](https://code-rules.fabricahq.com/start-here/install/) Versions, custom directories, checksum verification, and upgrades
- → [**Write a rule**](https://code-rules.fabricahq.com/guides/write-rules/) Write rules agents can actually follow
- → [**Configuration**](https://code-rules.fabricahq.com/reference/configuration/) Every field in `config.yaml`
- → [**CLI reference**](https://code-rules.fabricahq.com/reference/cli/) Every command and flag

## Contributing

Bug reports and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) covers building from source, running the validation suite, and working on the documentation site.

## License

Syllabus is [MIT licensed](LICENSE.md). Rule libraries carry their own license terms, which Syllabus preserves in each project that imports them.
