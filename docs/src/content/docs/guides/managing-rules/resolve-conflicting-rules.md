---
title: "Resolve conflicting rules"
description: "Find contradictory rules and make the project's intended policy explicit."
slug: guides/conflicting-guidance
---

Rules **conflict** when they require incompatible actions in the same situation. For example, one rule might require interfaces for object types while another requires type aliases. Both rules stay active until your project explicitly resolves the overlap. Code Rules checks file consistency, but it does not interpret rule prose or choose a source to prioritize.

In this guide, you'll review your project's active rules, decide which instruction should apply, record that choice, and check the resulting guidance.

Before you begin, [set up a project](/start-here/set-up-project/) with generated rules. Run commands from the project root, the top-level directory of your codebase. In a Git repository, project commands also work from a subdirectory.

## Recognize a conflict

Consider rules from two example libraries, `acme-rules` and `globex-rules`:

| Rule ID | Instruction |
| --- | --- |
| `acme-rules:techs/typescript/prefer-type-aliases` | Use type aliases for object types. |
| `globex-rules:techs/typescript/prefer-interfaces` | Use interfaces for object types. |

If both rules apply to the same object type, an agent cannot satisfy both. Source order gives neither rule priority.

First check each rule's scope and exceptions. A rule requiring interfaces for public extension points can coexist with one requiring type aliases for internal types. Repeated advice may be redundant without being contradictory.

## Generate a review prompt

1. Check that the project's generated files match its inputs:

   ```sh
   code-rules project check
   ```

   Repair any reported problem before reviewing. A passing check confirms file consistency, not that the rules agree with one another.

2. Give a repository-aware agent this prompt. The paths are relative to the project root:

   ```text
   Review this project's active Code Rules for conflicting guidance.

   Read .code-rules/generated/RULES.md, every page of each group index,
   and all linked resolved rules. Compare rules within and across groups,
   including local additions and replacements. Use
   .code-rules/generated/provenance.json to identify the reviewed snapshot.
   Excluded rules and replaced upstream text are context, not active rules.

   For each possible conflict, cite both source-qualified rule IDs and the
   relevant passages. Give a concrete situation where both rules apply.
   Check each rule's scope and exceptions, then explain why the actions
   cannot both be satisfied. Suggest a scoped replacement or exclusion and
   its tradeoff.

   Separate confirmed conflicts from unclear scope and repeated guidance.
   List the files reviewed and any missing or unreadable files. If coverage
   is incomplete, say so. No findings do not prove that no conflicts exist.
   Present proposed changes for review. Do not choose unresolved project
   policy or edit rule files on the owner's behalf.
   ```

3. Review each finding against the cited full rules and decide the intended project policy. Resolve unclear policy with the people who own the project before editing it.

## Resolve the intended policy

Record the decision in [Customize imported rules](/guides/customize/) or a local rule:

- **Clarify scope:** Replace an imported rule with a complete local definition whose conditions and exceptions remove the overlap.
- **Keep one imported policy:** Exclude the competing rule under its owning source and explain why.
- **Change a local addition:** Edit its authored file under `local/`. If another imported rule still conflicts, exclude or replace that rule under its source too.

For example, if the project adopts Globex's interface rule, exclude the type-alias rule from `acme-rules`:

```yaml
sources:
  acme-rules:
    exclude:
      techs/typescript/prefer-type-aliases: This project follows Globex's interface rule for object types.
```

The YAML snippet shows only the field to add. Merge `exclude` into the existing `sources.acme-rules` entry, keeping its `repository`, `ref` or `version`, `groups`, and other exceptions. The exclusion affects only `acme-rules`'s rule. Globex's rule stays active. Replace the example IDs with rules that exist in your selected libraries.

Keep changes in configuration and `local/`. Do not edit `vendor/` or `generated/` directly.

## Rebuild and review again

1. After changing only local rules or exceptions, run:

   ```sh
   code-rules project build
   code-rules project check
   ```

   If you also changed a source, revision, or selected groups, run `code-rules project sync` and then `code-rules project check` instead.

2. Read the affected group indexes and full resolved rules again. Confirm that the intended policy is active and no competing obligation remains. Repeat the agent review if the changed scope could affect other groups.

3. Review and commit configuration, local rules, and generated files together. Include vendor changes if sync refreshed them. A clean check confirms that files agree; your review decides whether the guidance can be followed together.
