---
title: "Group"
description: "The two types of rule groups: technologies and engineering practices."
---

A **rule group** collects related [rules](/concepts/rule/) and explains when an agent should read them.
There are two types:

- **Technology groups**, under `techs/`, cover a named language, framework, tool, platform, or protocol.
- **Practice groups**, under `practices/`, cover engineering practices that apply across technologies, such as testing or observability.

Both types use the same rule format and import behavior.
The distinction helps authors place a rule and agents find the guidance relevant to their work.

## Technologies

A technology is a named language, framework, tool, platform, or protocol.
Examples include TypeScript, React, Go, and Playwright.

Place a rule here when its obligation depends on that technology.
“Use Playwright's web-first assertions” belongs in `techs/playwright`.

## Practices

A practice describes how to engineer software across technologies.
Examples include testing, observability, error handling, and architecture.

“Tests assert observable behavior” belongs in `practices/testing`.
The language used in its example does not determine its home.

For a boundary case, consider infrastructure as code.
Reviewing destructive infrastructure changes is a practice; declaring Terraform provider constraints is technology-specific.

## Group identities

The directory determines the type and identity:

```text
techs/typescript
techs/playwright
practices/testing
practices/observability
```

Do not repeat a group's type in metadata.
A rule has one canonical home; related groups can reference it without copying its text.
References do not automatically install another group in the first release.
When multiple sources supply the same group ID, their rules share one group page. Small groups include full rules; larger groups provide applicability summaries with explicit links to individual resolved rule files.
Each rule keeps its source-qualified ID and each source retains its selection guidance.

## When to read a group

Each group supplies a name, description, and `whenToRead` guidance.
For testing, that guidance should include behavior changes even when no test files change.
Use the canonical [whenToRead authoring guidance](/reference/rule-authoring/#write-whentoread-guidance-that-helps-selection) to describe intended work, add recognizable examples, and check selection against representative tasks.

Group cues describe the group's intended area of work, even when it contains only one rule.
An individual rule's cue identifies situations that warrant reading it; its full text defines the obligation and exceptions.
Reading observability rules does not imply that every function needs a log statement.

See [Import rules](/guides/select-rules/) for selecting and adapting groups, and [Files and formats](/reference/files/#group-metadata) for metadata.
