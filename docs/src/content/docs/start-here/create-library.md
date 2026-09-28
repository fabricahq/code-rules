---
title: "Create your first library"
description: "Write and validate one shared rule, then publish it for projects to import."
---

Let's publish rules that other projects can use. A **library** is a collection of rule groups you maintain independently of the projects that import it. A **rule** is a Markdown file that explains one practice you want agents to follow.

In this walkthrough, you'll:

1. Create a library and write one rule about error messages.
2. Check the rule's format and choose terms for sharing it.
3. Commit the library and release the rule's first version.
4. Import the rule into a project and inspect the guidance its agents will read.

You need [Code Rules installed](/start-here/install/) and Git. You don't need to complete the project walkthrough first. Rules intended for only one codebase can stay [local to that project](/start-here/set-up-project/).

## 1. Create a repository for your library

Give your library its own Git repository so you can release its rules independently of the projects that use them. Start in the directory where you keep your repositories:

```sh
mkdir engineering-rules
cd engineering-rules
git init
code-rules library init
```

`engineering-rules` is an example repository name; you can choose another. `code-rules library init` creates:

- `rule-library.yaml`, which identifies the library format. Its `versioning: rules` line gives each rule its own version.
- A README for authors at the repository root.
- `.github/workflows/code-rules.yml`, which checks pull requests and releases rules once the library is on GitHub.

You'll publish this repository to your Git host in step 5.

Initialize from the repository root. After that, library commands also work from its subdirectories.

You may see a reminder that the library has no declared license. We'll address that before sharing it.

## 2. Add your first rule

A **group** collects related rules and tells agents when to read them. Create a group named "Error handling" to hold rules about how your code handles errors:

```sh
code-rules library add group practices/error-handling \
  --name 'Error handling' \
  --description 'Help callers and users understand and recover from failures.' \
  --when-to-read 'When implementing or reviewing error handling and error messages.'
```

Use the CLI to create a draft rule in that group:

```sh
code-rules library add rule practices/error-handling/make-errors-actionable \
  --title 'Make errors actionable' \
  --when-to-read 'When writing or reviewing an error message a user or caller will receive.' \
  --impact MEDIUM \
  --impact-description 'Helps people recover from a failure without guessing what went wrong.'
```

Open `practices/error-handling/make-errors-actionable.md` in your editor and replace its entire contents with this complete rule:

```md
---
title: Make errors actionable
whenToRead: When writing or reviewing an error message a user or caller will receive.
impact: MEDIUM
impactDescription: Helps people recover from a failure without guessing what went wrong.
tags: errors
---

## Make errors actionable

Explain what failed and what the user or caller can do next. Include relevant context that is safe to disclose.

Instead of "Invalid configuration", say "The configuration is missing a repository URL. Add a repository value under sources.acme-rules."

Do not include secrets, credentials, or private payloads in an error message. When recovery is not possible, explain the limitation rather than suggesting a retry that cannot help.

Check the message against the failure it describes: the explanation should be accurate and the suggested next step should address the cause.
```

The fields at the top describe the rule; the Markdown below tells the agent what to do. Replacing the draft removes its unfinished-draft marker.

The rule's path without `.md` is its ID: `practices/error-handling/make-errors-actionable`. Keep that path stable after projects start importing it.

Each rule has its own version, and every change to a rule needs a **change note** until it's released. The note tells projects what changed. Record that you added this rule:

```sh
code-rules library change practices/error-handling/make-errors-actionable \
  --summary 'Add the rule.'
```

This creates `make-errors-actionable.change.yaml` beside the rule. A new rule's note has no change size, because its first version is always `1.0.0`.

Your library now looks like this:

```text
engineering-rules/
  .github/workflows/code-rules.yml
  README.md
  rule-library.yaml
  practices/
    error-handling/
      _group.yaml
      README.md
      make-errors-actionable.md
      make-errors-actionable.change.yaml
```

Library rules live directly under `practices/` or `techs/`. Publishing a library does not require a Code Rules directory or generated agent guidance. Those belong to projects that consume rules.

## 3. Check the library

You've written a rule. Now check that Code Rules can read and import it:

```sh
code-rules library check
```

The result should report **1 group and 1 rule**, and preview a pending release of `practices/error-handling/make-errors-actionable` at version `1.0.0`. The command checks the library metadata, group definitions, rule format, and change notes without changing your files. Read the rule yourself to decide whether its advice is clear and useful.

There is no library build step. A library publishes source rules; each consuming project builds its own agent guidance after selecting rules and applying its exclusions and replacements. See [how project builds work](/start-here/set-up-project/#3-build-the-guidance-your-agent-will-read).

## 4. Choose terms before sharing

Decide who may use, adapt, and redistribute the library. Public visibility alone does not grant those permissions. Use [License rules](/guides/license-rules/) to choose and record terms, especially when including someone else's material.

For example, if you choose MIT for your own rules, add the complete MIT text with the appropriate copyright notice to `LICENSE.md`, then set `rule-library.yaml` to:

```yaml
formatVersion: 1
versioning: rules
license:
  spdxExpression: MIT
  file: LICENSE.md
  notices: []
```

This records your choice; Code Rules does not supply or infer the license text. Retain any required notices for adapted material and declare them in `notices`.

Check the library again after adding the license file and metadata:

```sh
code-rules library check
```

The check verifies that the declared files exist and that the metadata is valid. It does not decide whether you have permission to publish someone else's material.

## 5. Publish and release the library

Once the library passes its checks and you've reviewed the rule and license, commit the files:

```sh
git add .github README.md rule-library.yaml LICENSE.md practices/
git commit -m "Create the first shared rule"
```

Create an empty repository on your Git host, then replace the example URL below with its Git URL and push:

```sh
git remote add origin https://github.com/YOUR-ORG/engineering-rules.git
git push -u origin HEAD
```

Now release the rule:

```sh
code-rules library release
```

Release turns the change note into a version. It deletes the note, commits `Release 1 rule`, and tags that commit `practices/error-handling/make-errors-actionable@1.0.0`. It pushes the commit and tag, and creates a GitHub Release with the [GitHub CLI](https://cli.github.com/). If your library isn't on GitHub.com, it creates the tag only. Add `--no-github-release` to skip GitHub Releases.

Your library is now available to other projects. The repository can be public or private; consuming projects need access to it.

From now on, change rules through pull requests, each with a change note. The workflow in `.github/workflows/code-rules.yml` opens a "Release rules" pull request, and merging it publishes the new versions. See [Version your rules](/guides/version-rules/) to finish setting it up.

## 6. Try the library in a project

In a [project you've set up](/start-here/set-up-project/), run the following from its root directory. Replace the example URL below with your library's Git URL:

```sh
code-rules project add library acme-rules \
  --repository https://github.com/YOUR-ORG/engineering-rules.git \
  --groups practices/error-handling
```

This adds the library's repository and group to `.code-rules/config.yaml` under the source name `acme-rules`. Without a `ref`, the project follows the library's releases.

Download the rules and build the project's agent guidance:

```sh
code-rules project sync
code-rules project check
```

Open `.code-rules/generated/RULES.md` and follow its "Error handling" group to your shared rule, which shows version `1.0.0`. Sync records that release, so the project keeps the same rules until someone runs `code-rules project update`.

The rules are now in the project, but its agent needs instructions to read them. If you haven't already, [connect the rules to your agent and try a task](/start-here/set-up-project/#5-give-the-rules-to-your-agent). Then commit the project's configuration, imported rules, generated guidance, and agent instructions together.

Other projects can import the same library. You maintain the shared rule in the library, and each project chooses when to adopt your updates.

## Next steps

As your library grows, [write focused rules](/guides/write-rules/), record each change with `code-rules library change`, and [release them](/guides/version-rules/). Projects [choose when to adopt updates](/guides/update/).

For metadata and supporting files, see [Rule and library format](/reference/rule-library-format/).
