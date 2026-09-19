---
title: "Create your first library"
description: "Write and validate one shared rule, then publish it for projects to import."
---

A **library** is a Git repository that publishes rules for other projects to use. Create one when you want to maintain the same engineering guidance for several codebases. A rule that belongs to only one project can stay [local to that project](/start-here/set-up-project/).

This walkthrough creates a library containing one complete rule. You need [Code Rules installed](/start-here/install/) and Git when you're ready to share it. You don't need to complete the project walkthrough first.

## 1. Create the library folder

Choose an empty folder for this example:

```sh
mkdir engineering-rules
cd engineering-rules
code-rules library init
```

`engineering-rules` is an example name; you can choose another. The command creates `rule-library.json`, which identifies the library format, and a README for authors. It does not create a remote repository.

You may see a reminder that the library has no declared license. We'll address that before sharing it.

## 2. Add a group and a rule

Create an Error handling group:

```sh
code-rules library add group practices/error-handling \
  --name 'Error handling' \
  --description 'Help callers and users understand and recover from failures.' \
  --when-to-read 'When implementing or reviewing error handling and error messages.'
```

Create `practices/error-handling/make-errors-actionable.md` in your editor and paste this rule:

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

Instead of "Invalid configuration", say "The configuration is missing a repository URL. Add a repository value under sources.team."

Do not include secrets, credentials, or private payloads in an error message. When recovery is not possible, explain the limitation rather than suggesting a retry that cannot help.

Check the message against the failure it describes: the explanation should be accurate and the suggested next step should address the cause.
```

The rule's path is its ID: `practices/error-handling/make-errors-actionable`. Keep that path stable after projects start importing it.

Your library now looks like this:

```text
engineering-rules/
  README.md
  rule-library.json
  practices/
    error-handling/
      _group.json
      README.md
      make-errors-actionable.md
```

Library rules live directly under `practices/` or `techs/`. Unlike a consuming project, a library does not need a `.code-rules/` folder or generated agent guidance.

## 3. Check the library

```sh
code-rules library check
```

The result should report **1 group and 1 rule**. The check validates the files and metadata; read the rule yourself to decide whether its advice is clear and useful. Use [Write a rule](/guides/write-rules/) when you want help developing a rule beyond this example.

## 4. Choose terms before sharing

Decide who may use, adapt, and redistribute the library. Public visibility alone does not grant those permissions. Use [License rules](/guides/license-rules/) to choose and record terms, especially when including someone else's material.

For example, if you choose MIT for your own rules, add the complete MIT text with the appropriate copyright notice to `LICENSE.md`, then set `rule-library.json` to:

```json
{
  "formatVersion": 1,
  "license": {
    "spdxExpression": "MIT",
    "file": "LICENSE.md",
    "notices": []
  }
}
```

This records your choice; Code Rules does not supply or infer the license text. Retain any required notices for adapted material and declare them in `notices`.

Run `code-rules library check` again after adding the terms.

## 5. Publish and import it

Create a Git repository and commit the files:

```sh
git init
git add .
git commit -m "Create the first shared rule"
git tag v0.1.0
```

Create an empty repository on your Git host, add it as `origin`, and push the commit and tag through your usual Git workflow. The repository can be public or private; consuming projects need access to it.

Then return to a project you've initialized with `code-rules init`. Replace the example URL below with your library's Git URL:

```sh
code-rules add source team \
  --repository https://github.com/YOUR-ORG/engineering-rules.git \
  --ref v0.1.0 \
  --groups practices/error-handling
code-rules sync
code-rules check
```

Open `.code-rules/generated/RULES.md` in that project and follow its Error handling group to your shared rule. The same library can now serve another project without copying and maintaining the rule by hand.

As your library grows, add focused rules, run `library check`, and publish new version tags. Projects [choose when to adopt updates](/guides/update/). For metadata and supporting files, see [Rule and library format](/reference/rule-library-format/).
