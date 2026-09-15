---
title: "Import rules"
description: "Choose groups from rule libraries and adapt their rules to your project."
---

Import the technology and practice groups your project needs from one or more libraries.
Add local rules and declare exceptions as part of that import configuration.
Code Rules combines those choices into a root group index, group pages, and individual full rule files. Small group pages include complete rules; larger ones contain applicability summaries with explicit reading links.

These commands describe the proposed release.
For a complete setup walkthrough, see [Use rules in a project](/guides/use-rules/).

## Choose groups for your project

Start with the project's technologies and engineering practices.
Inspect its dependencies, architecture, and expected behavior, then read each library's group descriptions and `whenToRead` guidance.

For a TypeScript service that makes requests to other services, useful groups might include:

- `techs/typescript` for language rules.
- `practices/testing` for verifying behavior.
- `practices/observability` for making failures and retries visible.
- `practices/error-handling` for how failures reach callers.

Import rules that cover the project's work, including practices that do not correspond to a package or filename.
Agents will choose which installed groups apply to each task using the [rule-loading process](/for-agents/).

## Select groups from each source

In `.code-rules/config.json`, set `sources.<name>.groups` separately for each library.
Each source has a `repository`, either an exact `ref` or a semantic `version` constraint, and its own `exclude` and `replace` objects.
Use the [complete configuration example](/reference/configuration/#complete-example) as your starting point.

To adopt an entire library, set `groups` to `"*"` instead of an array.
Use `"practices/*"` for all practice groups, or `"techs/*"` for all technology groups.
All groups within that scope at the selected revision are included, and exclusions and replacements still apply.
New groups enter when you update the adopted revision. Review them as part of that update.
See [Import every group](/reference/configuration/#import-every-group) for an example and snapshot requirements.

If two sources supply `practices/testing`, their rules combine into one generated testing page, with full rules or summaries and links to individual resolved rules.
Source-prefixed IDs keep matching rule paths distinct.
Neither source automatically overrides the other.

Keep local-only groups in `localGroups`, with their metadata under `local/`.
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

The group must appear in a source's `groups` or in `localGroups`.
The rule joins the inherited rules and receives a `local:`-prefixed ID.
Use the [authoring format](/guides/write-rules/) for its metadata and body.

For a local-only group, include its own `_group.json` and declare the group in `localGroups`.
Imported groups retain source-labeled metadata from every contributing library.

### Exclude a rule

Add its library-relative ID and a reason to `sources.<name>.exclude`:

```json
{
  "sources": {
    "acme": {
      "exclude": {
        "practices/testing/avoid-snapshot-tests": "Contract snapshots follow our separate review policy."
      }
    }
  }
}
```

These are partial snippets, not complete source definitions.
Merge them into the `acme` and `fabrica` sources from the [configuration example](/reference/configuration/), retaining their repository, revision selection, and groups.
Rule IDs are illustrative and must exist in the selected source groups.
An exclusion removes only the named source's rule, without introducing a replacement.
The same rule path in another source remains active.
The exclusion reason stays in project configuration; the generated files contain only active rules.

### Replace a rule

Write a complete local definition, then reference it from `sources.<name>.replace`:

```json
{
  "sources": {
    "fabrica": {
      "replace": {
        "techs/typescript/prefer-type-aliases": {
          "file": "local/techs/typescript/prefer-interfaces.md",
          "reason": "Our public extension API relies on declaration merging."
        }
      }
    }
  }
}
```

The resolved rule uses the complete local definition, including its local ID, title, reading cue, impact, body, attribution, and asset references.
In this example, the generated ID is `local:techs/typescript/prefer-interfaces`.
Configuration and provenance record what it replaced and why; generated rule guidance does not display that history.
Its file stays in the target group and does not also become an additional rule.

Replacing “prefer type aliases” with “prefer interfaces” leaves only the local definition in the effective output.
Include the intended scope and exceptions in that definition.

## Generate and review

When sources, revision selections, or selected groups change, run the proposed `code-rules sync` command.
For changes limited to local rules or exceptions, run `code-rules build` against the existing vendor snapshots.
Review and commit the updated generated files with their inputs.

The importer rejects missing targets, a rule both excluded and replaced, and replacement files reused for multiple rules.
It does not infer overrides from similar wording.
Rules from different sources remain active even when their paths or titles match.
If a local rule or another source contradicts an inherited obligation, explicitly replace or exclude the affected rule inside its owning source.
Source order never resolves the conflict.
Use [Conflicting guidance](/guides/conflicting-guidance/) to review the combined rules and resolve competing instructions.


Review `generated/RULES.md`, each relevant group index, and the full resolved definitions. Check exclusions and replacements against configuration and provenance.
Commit configuration, local rules, vendor snapshots, and generated files together.
Use [Update rules](/guides/update/) when you adopt new library versions or change the selected groups later.
