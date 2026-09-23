---
title: "Import rules"
description: "Choose groups from rule libraries and adapt their rules to your project."
---

You can **import rules** written by your team or others instead of writing every rule yourself. These rules come from **libraries**, collections of rules shared across projects.

In this guide, you'll choose **groups** to import, record your choices in `.code-rules/config.yaml`, and add any project-specific rules or exceptions. Then you'll generate and review the files your agent will read.

Start with a project that already has a `.code-rules/` directory. If you haven't created one, follow [Set up your first project](/start-here/set-up-project/).

## Choose groups for your project

Start with the project's technologies and engineering practices.
Inspect its dependencies, architecture, and expected behavior, then read each library's group descriptions and `whenToRead` guidance.

For a TypeScript service that makes requests to other services, useful groups might include:

- `techs/typescript` for language rules.
- `practices/testing` for verifying behavior.
- `practices/observability` for making failures and retries visible.
- `practices/error-handling` for how failures reach callers.

Import rules that cover the project's work, including practices that do not correspond to a package or filename.
Agents choose which installed groups apply to each task using the [rule-loading process](/for-agents/).

## Select groups from each source

In `.code-rules/config.yaml`, each library you import gets a named entry under `sources`, such as `fabrica`. For each library, choose:

- **Where to get it:** `repository` is the library's Git URL.
- **Which version to use:** `ref` selects a specific tag or commit; `version` allows a range of versions. Use one or the other.
- **Which groups to import:** `groups` lists the groups you want, such as `practices/testing`.

You can also use `exclude` to leave out individual rules or `replace` to substitute your own. We'll cover both below.

See the [complete configuration example](/reference/configuration/#complete-example) for how these fields fit together.

To adopt an entire library, set `groups` to `"*"` instead of an array.
Use `"practices/*"` for all practice groups, or `"techs/*"` for all technology groups.
All groups within that scope at the selected revision are included, and exclusions and replacements still apply.
New groups enter when you update the adopted revision. Review them as part of that update.
See [Import every group](/reference/configuration/#import-every-group) for an example and snapshot requirements.

If two sources supply `practices/testing`, their rules combine into one generated testing page, with full rules or summaries and links to individual resolved rules.
Source-prefixed IDs keep matching rule paths distinct.
Neither source automatically overrides the other.

Define local groups with `_group.yaml` under `local/`; a separate configuration declaration is unnecessary.
The [Group concept](/concepts/groups/) explains the distinction between technology and practice groups.

## Adapt the import to your project

Use shared rules as your starting point.
Add project-specific obligations, exclude a rule with a reason, or replace it with a complete local definition.
Keep those decisions in configuration and local files so agents read the resolved result.

### Add a rule

Write a Markdown rule under the matching local group:

```text
.code-rules/local/practices/testing/test-project-contracts.md
```

The group needs `_group.yaml` metadata from a selected library or from `local/<group-id>/`.
The rule joins the inherited rules and receives a `local:`-prefixed ID.
Use the [authoring format](/guides/write-rules/) for its metadata and body.

For a local-only group, include its own `_group.yaml`. It is discovered automatically.
When a library later supplies that group, the local metadata and rules stay in place. Local metadata supplies the project's group description; imported rules join the group.
Imported groups retain source-labeled metadata from every contributing library.

### Exclude a rule

Add its library-relative ID and a reason to `sources.<name>.exclude`:

```yaml
sources:
  acme:
    exclude:
      practices/testing/avoid-snapshot-tests: Contract snapshots follow our separate review policy.
```

These are partial snippets, not complete source definitions.
Merge them into the `acme` and `fabrica` sources from the [configuration example](/reference/configuration/), retaining their repository, revision selection, and groups.
Rule IDs are illustrative and must exist in the selected source groups.
An exclusion removes only the named source's rule, without introducing a replacement.
The same rule path in another source remains active.
The exclusion reason stays in project configuration; the generated files contain only active rules.

### Replace a rule

Write a complete local definition, then reference it from `sources.<name>.replace`:

```yaml
sources:
  fabrica:
    replace:
      techs/typescript/prefer-type-aliases:
        file: local/techs/typescript/prefer-interfaces.md
        reason: Our public extension API relies on declaration merging.
```

The resolved rule uses the complete local definition, including its local ID, title, reading cue, impact, body, attribution, and asset references.
In this example, the generated ID is `local:techs/typescript/prefer-interfaces`.
Configuration and provenance record what it replaced and why; generated rule guidance does not display that history.
Its file stays in the target group and does not also become an additional rule.

Replacing “prefer type aliases” with “prefer interfaces” leaves only the local definition in the effective output.
Include the intended scope and exceptions in that definition.

## Generate and review

When sources, revision selections, or selected groups change, run `code-rules project sync`.
For changes limited to local rules or exceptions, run `code-rules project build` against the existing vendor snapshots.
Review and commit the updated generated files with their inputs.

The importer rejects missing targets, a rule both excluded and replaced, and replacement files reused for multiple rules.
It does not infer overrides from similar wording.
Rules from different sources remain active even when their paths or titles match.
If a local rule or another source contradicts an inherited obligation, explicitly replace or exclude the affected rule inside its owning source.
Source order never resolves the conflict.
Use [Resolve conflicting rules](/guides/conflicting-guidance/) to review the combined rules and resolve competing instructions.


Review `generated/RULES.md`, each relevant group index, and the full resolved definitions. Check exclusions and replacements against configuration and provenance.
Commit configuration, local rules, vendor snapshots, and generated files together.
Use [Update rules](/guides/update/) when you adopt new library versions or change the selected groups later.
