---
title: "Import rules"
description: "Choose groups from rule libraries and adapt their rules to your project."
---

You can **import rules** written by your team or others instead of writing every rule yourself. These rules come from **libraries**, collections of rules shared across projects.

In this guide, you'll choose **groups** to import, record your choices in `.syllabus/config.yaml`, and add any project-specific rules or exceptions. Then you'll generate and review the files your agent will read.

Start with a project that already has a `.syllabus/` directory. If you haven't created one, follow [Set up your first project](/start-here/set-up-project/).

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

In `.syllabus/config.yaml`, each library you import gets a named entry under `sources`, such as `fabrica`. For each library, choose:

- **Where to get it:** `repository` is the library's Git URL.
- **Which rules to import:** `rules.include` lists selectors. A group's selector, such as `practices/testing/*`, includes every rule in the group, including rules the library adds to it later. A rule ID, such as `techs/go/wrap-errors-with-operation`, imports one rule without the rest of its group; rules added to that group later don't join. See [Select individual rules](/reference/configuration/#select-individual-rules).
- **Which versions to use:** by default, each rule follows its newest version when you update. Add `rules.pins` to keep individual rules at exact versions. See [Choose versions](/reference/configuration/#choose-versions).

You can also use `rules.exclude` to leave out rules, optionally replacing one with your own. We'll cover both below.

See the [complete configuration example](/reference/configuration/#complete-example) for how these fields fit together.

To adopt an entire library, include `"*"`.
Use `practices/*` for all practice groups, or `techs/*` for all technology groups, alone or beside other selectors.
All groups within that scope in the imported library are included, and exclusions and replacements still apply.
When a library adds rules or groups within your selector, `syllabus project update` adds them. Review them as part of that update.
See [Select rules](/reference/configuration/#select-rules) for every selector form and snapshot requirements.

If two sources supply `practices/testing`, their rules combine into one generated testing page, with full rules or summaries and links to individual resolved rules.
Source-prefixed IDs keep matching rule paths distinct.
Neither source automatically overrides the other.

Define your own groups with `_group.yaml` under `rules/`; a separate configuration declaration is unnecessary.
The [Group concept](/concepts/groups/) explains the distinction between technology and practice groups.

## Adapt the import to your project

Use shared rules as your starting point.
Add project-specific obligations, exclude a rule with a reason, or replace it with a complete local definition.
Keep those decisions in configuration and local files so agents read the resolved result.

### Add a rule

Write a Markdown rule under the matching group in `rules/`:

```text
.syllabus/rules/practices/testing/test-project-contracts.md
```

The group needs `_group.yaml` metadata from a selected library or from `rules/<group-id>/`.
If you later deselect or remove the last library supplying the group, sync copies the group's last imported metadata to `rules/<group-id>/_group.yaml` and warns that it did, so the rule keeps its group; see [Keep an authored rule's group](/reference/sync/#keep-an-authored-rules-group).
Put supporting files that several of your rules share in `.syllabus/assets/`, and link to them from the rule as `../../../assets/<file>`.
The rule joins the inherited rules and receives a `local:`-prefixed ID.
Use the [authoring format](/guides/write-rules/) for its metadata and body.

For a local-only group, include its own `_group.yaml`. It is discovered automatically.
When a library later supplies that group, the local metadata and rules stay in place. Local metadata supplies the project's group description; imported rules join the group.
Imported groups retain source-labeled metadata from every contributing library.

### Exclude a rule

Add an entry with its library-relative ID and a reason to `sources.<name>.rules.exclude`:

```yaml
sources:
  acme:
    rules:
      exclude:
        - id: practices/testing/avoid-snapshot-tests
          reason: Contract snapshots follow our separate review policy.
```

These are partial snippets, not complete source definitions.
Merge them into the `acme` and `fabrica` sources from the [configuration example](/reference/configuration/), retaining their repository, `rules.include`, and pins.
Rule IDs are illustrative and must name rules the source imports.
To exclude every rule in a group, or every rule of a type, use a selector as the `id`, such as `practices/observability/*`; rules the library adds there later stay excluded too.
An exclusion removes only the named source's rule, without introducing a replacement.
The same rule path in another source remains active.
The exclusion reason stays in project configuration; the generated files contain only active rules.

### Replace a rule

Write a complete local definition, then exclude the imported rule and name your rule's ID as its `replacedBy`:

```yaml
sources:
  fabrica:
    rules:
      exclude:
        - id: techs/typescript/prefer-type-aliases
          reason: Our public extension API relies on declaration merging.
          replacedBy: techs/typescript/prefer-interfaces
```

Here the local rule is `.syllabus/rules/techs/typescript/prefer-interfaces.md`.

To start from the library's text instead of writing your own, [fork the rule](/reference/cli/#fork-a-library-rule), which copies it into `rules/` and writes the exclusion for you, with the forked version as `basedOn`. With `basedOn`, `syllabus project update` lists the library's changes after that version, so you can decide whether your rule needs them; see [Exclude or replace a rule](/reference/configuration/#exclude-or-replace-a-rule).

The resolved rule uses the complete local definition, including its local ID, title, reading cue, impact, body, attribution, and asset references.
In this example, the generated ID is `local:techs/typescript/prefer-interfaces`.
Configuration and provenance record what it replaced and why; generated rule guidance does not display that history.
Its file stays in the target group and does not also become an additional rule.

Replacing “prefer type aliases” with “prefer interfaces” leaves only the local definition in the effective output.
Include the intended scope and exceptions in that definition.

## Generate and review

When sources, pins, or selected groups or rules change, run `syllabus project sync`.
For changes limited to local rules or exceptions, run `syllabus project build` against the existing vendor snapshots.
Review and commit the updated generated files with their inputs.

The importer rejects exclusions of rules the source doesn't import, missing replacement files, and a local file that replaces more than one rule.
It does not infer overrides from similar wording.
Rules from different sources remain active even when their paths or titles match.
If a local rule or another source contradicts an inherited obligation, explicitly replace or exclude the affected rule inside its owning source.
Source order never resolves the conflict.
Use [Resolve conflicting rules](/guides/conflicting-guidance/) to review the combined rules and resolve competing instructions.


Review `generated/RULES.md`, each relevant group index, and the full resolved definitions. Check exclusions and replacements against configuration and provenance.
Commit configuration, local rules, vendor snapshots, and generated files together.
Use [Update rules](/guides/update/) when you adopt new library releases or change the selected groups later.
