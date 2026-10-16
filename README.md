<h1 align="center">Code Rules</h1>

<h3 align="center">The package manager for your agent skills and&nbsp;rules.</h3>

<p align="center">
  Track, update, and share the agent skills you already use, across your
  repositories and agents. Add the granular rules your skills&nbsp;apply.
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

Code Rules is a **package manager** for your agent skills and engineering rules.

> [!NOTE]
> Skill support isn't in a released version of Code Rules yet. The released CLI manages rules; this README describes how skills will work.

That means you can:

- Install skills from libraries and other Git repositories, with a record of where each came from and which version you have
- Update skills and rules when you choose, after previewing what changed
- Share skills and rules across your organization, and require the skills every project needs
- Install each skill for Claude Code, Codex, Cursor, and other agents at once
- Capture engineering best practices as individual rules, so your skills and agents apply them consistently

## How is it useful?

### Skills need a package manager

Most people already guide their agents with skills, the AI skills we all know, officially folders that follow the open [Agent Skills](https://agentskills.io) standard. They install skills from a teammate's repository, a vendor's collection, or the last project they worked on. Before long, it's hard to know where each skill came from, keep it up to date, share it across an organization, or use it with more than one agent.

Pulling many files from versioned sources into a project, and keeping them up to date, is a familiar problem. Programming languages have solved it for more than 20 years with the humble package manager. Code Rules applies that same approach to skills. Each project declares its sources in one configuration file, and Code Rules records each skill's source and exact revision, updates it only when you ask, and writes it into the folders your agents read. Your organization can publish shared skills in its own `.code-rules` repository and require or recommend them for every project.

To publish skills, you put them in a **[library](https://code-rules.fabricahq.com/concepts/libraries/)**: a Git repository whose manifest lists the skills it offers, with a version for each skill and a change note for each change, so tools and projects can read exactly what you publish. For example, see the [Fabrica Public Rules Library](https://github.com/fabricahq/public-rules).

### Skills aren't enough

Skills are coarse-grained. They're good for playbooks, techniques, and coordinating a whole task, such as building a feature or reviewing a pull request. But a skill that bundles many practices makes it hard to use a subset of its guidance on a specific project.

So Code Rules adds **[rules](https://code-rules.fabricahq.com/concepts/rule/)**: one practice per Markdown file, optionally following the [Code Rules rule template](https://code-rules.fabricahq.com/reference/rule-authoring/). Rules give guidance on **technologies** like Go or TypeScript, or on **practices** like testing, observability, or even writing good READMEs, and live in **rule groups**. **[Skills](https://code-rules.fabricahq.com/concepts/skill/)** cite the rules each step applies, so a project can exclude or replace one rule without forking the skill.

Code Rules manages rules and skills together, for libraries that publish them, organizations that standardize on them, and projects that use them. You can write your own project-specific rules and skills, or take them from libraries.

As with packages for code, you can:

- **Version every rule and skill.** Each has its own semantic version, so an update shows exactly what changed and whether a change could break code that followed it. Projects record the exact version of everything they import, and accept major changes explicitly.
- **Customize without forking the library.** Exclude an imported rule or replace it with your own, with the decision recorded in config.
- **Build reproducibly.** Every imported rule is vendored at its exact version, and keeps its source and license terms.
- **Catch drift in CI.** `code-rules project check` fails when generated files are out of date.

Agents don't read every rule on every task. Code Rules generates an index with a "when to read" cue for each rule group, so agents open only the rules that matter for the work at hand. It works with any agent that reads a project instruction file, such as Claude Code, Codex, Cursor, or Gemini CLI.

### Rules can evolve

As you work, you will find that some rules consistently deliver value, while others start to get in the way or no longer represent your preferred way of working. Or you may be repeatedly giving the same guidance to agents, in which case, it may be time to create a rule for it.

Because rules are files in Git, you can ask your agent to review a session and propose a rule change, then review that change like any other code. Your guidance improves in version-controlled steps, and once a library release publishes the change, every project that uses the rule picks it up on its next `code-rules project update`.

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

Then add the rule to a rule group:

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

Leave out `--body-file` to start from a template instead. Either way, the rule lives in `.code-rules/rules/`, where you keep editing it.

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

`RULES.md` is an index that tells agents which rule groups to open for the task at hand:

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
2. If `.code-rules/generated/SKILLS.md` exists, read it and use the skills
   relevant to the task. Read each skill's instructions and the rules it
   cites before performing the steps that apply them.
3. Follow the applicable rules and their exceptions while doing the work.
4. Before finishing, check your work against those rules and run the
   relevant validation. Briefly report what you verified and any gaps.

If required rule files are unavailable or give conflicting instructions,
report the issue rather than silently skipping them or choosing a policy.
```

Commit `.code-rules/` and you're done. Now ask your agent for a change as you normally would; you don't need to mention Code Rules in each request.

➡️ The full walkthrough is in [Set up your first project](https://code-rules.fabricahq.com/start-here/set-up-project/).

## Share rules across projects

Once a rule proves itself, you may want to move it into a **library**: a Git repository of rule groups that any project can import. A library keeps its rules in `.code-rules/rules/`, the same layout a project uses, with its manifest in the `library` section of `.code-rules/config.yaml`, so one repository can be both.

We recommend publishing your team's default rules in a repository such as `acme/.code-rules`, or starting from an existing library like the [Fabrica Public Rules Library](https://github.com/fabricahq/public-rules):

```sh
code-rules project add source fabrica \
  --repository https://github.com/fabricahq/public-rules.git \
  --rules 'practices/testing/*' \
  --rules 'techs/go/*'
code-rules project sync
```

The dependency lands in `.code-rules/config.yaml`, the file you review and commit:

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

`code-rules project sync` imports each rule's newest version, snapshots them into `.code-rules/vendor/` with their versions, and rebuilds. Improve a rule in the library and publish it in a library release: each project picks up the improvement when it runs `code-rules project update`, on its own schedule. Update previews every changed, new, and retired rule, and applies them only after you confirm.

➡️ [Create your first library](https://code-rules.fabricahq.com/start-here/create-library/) · [Include rules](https://code-rules.fabricahq.com/guides/select-rules/) · [Update rules and skills](https://code-rules.fabricahq.com/guides/update/)

## What Code Rules doesn't do

- **It doesn't enforce your rules.** Code Rules gives your agents the same rules for writing and reviewing code, but giving an agent a rule doesn't guarantee that it follows it. `code-rules project check` verifies your rule files, not your application code.
- **It doesn't resolve contradictions automatically.** If two rules disagree, you decide which to exclude or replace. [Resolve conflicting rules](https://code-rules.fabricahq.com/guides/conflicting-guidance/) gives you a prompt that has your agent find the conflicts and propose fixes for you to review.
- **It doesn't manage personal skills.** Code Rules installs skills into your repositories, not your home directory. It warns when a personal skill has the same name as one your project installs.
- **It doesn't run on native Windows yet.** The Linux build is expected to work in WSL 2, but hasn't been tested end to end.

## Learn more

- → [**What is Code Rules?**](https://code-rules.fabricahq.com/start-here/overview/) The problem, and how skills, rules, rule groups, projects, and libraries fit together
- → [**Install**](https://code-rules.fabricahq.com/start-here/install/) Versions, custom directories, checksum verification, and upgrades
- → [**Write a rule**](https://code-rules.fabricahq.com/guides/write-rules/) Write rules agents can actually follow
- → [**Configuration**](https://code-rules.fabricahq.com/reference/configuration/) Every field in `config.yaml`
- → [**CLI reference**](https://code-rules.fabricahq.com/reference/cli/) Every command and flag

## Contributing

Bug reports and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) covers building from source, running the validation suite, and working on the documentation site.

## License

Code Rules is [MIT licensed](LICENSE.md). Rule libraries carry their own license terms, which Code Rules preserves in each project that imports them.
