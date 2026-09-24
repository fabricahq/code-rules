---
title: "Resolve conflicting rules"
description: "Find contradictory rules and make the project's intended policy explicit."
slug: guides/conflicting-guidance
---

Rules **conflict** when they require incompatible actions in the same situation. For example, one rule might require interfaces for object types while another requires type aliases. Both rules stay active in a project until it adopts corrected library guidance or records an exception. Code Rules checks file consistency, but it does not interpret rule prose or choose a source to prioritize.

In this guide, you'll review active rules, decide who owns a fix, record the intended policy, and check the resulting guidance.

Before you begin, [set up a project](/start-here/set-up-project/) with generated rules. Run commands from the project root, the top-level directory of your codebase. In a Git repository, project commands also work from a subdirectory.

## Recognize a conflict

Consider rules from two example libraries, `acme-rules` and `globex-rules`:

| Rule ID | Instruction |
| --- | --- |
| `acme-rules:techs/typescript/prefer-type-aliases` | Use type aliases for object types. |
| `globex-rules:techs/typescript/prefer-interfaces` | Use interfaces for object types. |

If both rules apply to the same object type, an agent cannot satisfy both. Source order gives neither rule priority.

First check each rule's scope and exceptions. A rule requiring interfaces for public extension points can coexist with one requiring type aliases for internal types. A rule that allows interfaces does not contradict a rule that requires type aliases. A recommendation is also different from a requirement. If a compatible reading exists but the scope is unclear, report that ambiguity instead of a confirmed conflict.

Conflicts can occur within one library as well as across libraries. Repeated advice is not automatically a defect: independently reusable groups may state the same expectation so each group makes sense on its own. Consolidate repeated guidance only when it creates ambiguity or maintenance problems.

## Generate a review prompt

1. Choose the review scope. For a project, review its selected rules. To audit a library you maintain, import all of its groups into a disposable project at the revision you want to review. Leave out local rules, exclusions, and replacements so they cannot hide library conflicts. From that project, check that generated files match their inputs:

   ```sh
   code-rules project check
   ```

   Repair any reported problem before reviewing. A passing check confirms file consistency, not that the rules agree with one another.

2. Tell a repository-aware agent which scope you chose, then give it this prompt. The paths are relative to the chosen project root:

   ```text
   Review this project's active Code Rules for conflicting guidance.
   State whether this is a project's selected rules or a maintainer audit
   of all groups in a disposable project. Identify the selected sources,
   revisions, groups, and any gaps in coverage.

   Read .code-rules/generated/RULES.md, every page of each group index,
   and all linked resolved rules. Read instructions, examples, and
   validation guidance. Compare rules within and across groups and
   sources, including local additions and replacements. Use
   .code-rules/generated/provenance.json to identify the reviewed snapshot.
   Excluded rules and replaced upstream text are context, not active rules.

   For each possible conflict, cite both resolved rule IDs and the
   relevant passages. Describe a concrete situation where both rules apply,
   including the relevant technologies, versions, and conditions. Identify
   unknown context rather than guessing. Check scope and exceptions. Distinguish
   what each rule requires, allows, or recommends; permission is not an
   obligation. Confirm a conflict only when both requirements apply and
   cannot be satisfied together in that situation. If a reasonable reading
   allows both but the wording is unclear, report unclear scope instead.
   Classify issues found only in examples or validation separately from
   conflicting instructions.

   Separate confirmed conflicts from unclear scope and repeated guidance.
   Do not treat repetition across independently reusable groups as a
   defect by itself. For each finding, propose a fix at the right owner:
   an authored library change if the maintainer owns the rules, or an
   explicit project exclusion or replacement for a local policy decision
   or workaround. Explain the tradeoff.
   List the files reviewed and any missing or unreadable files. If coverage
   is incomplete, say so. No findings do not prove that no conflicts exist.
   Present proposed changes for human approval. Do not choose unresolved
   policy or edit rule files on the owner's behalf.
   ```

3. Review each finding against the cited full rules and decide the intended project policy. Resolve unclear policy with the people who own the project before editing it.

## Resolve the intended policy

Fix a conflict where the guidance is owned:

- **Library you maintain:** Revise the authored rules or examples to clarify their scope or correct the conflict. Run `code-rules library check`, [publish a new version](/start-here/create-library/#5-commit-and-publish-the-library), then have each project [review and adopt the update](/guides/update/). Editing a project snapshot does not fix the library.
- **Project policy or temporary workaround:** Use [Customize imported rules](/guides/customize/) to exclude a rule under its owning source, or replace it with a complete local definition. State the reason and review any remaining competing rules.
- **Local addition:** Edit its authored file under `local/`. If an imported rule still conflicts, exclude or replace that rule under its source too.

For example, if the project adopts Globex's interface rule, exclude the type-alias rule from `acme-rules`:

```yaml
sources:
  acme-rules:
    exclude:
      techs/typescript/prefer-type-aliases: This project follows Globex's interface rule for object types.
```

The YAML snippet shows only the field to add. Merge `exclude` into the existing `sources.acme-rules` entry, keeping its `repository`, `ref` or `version`, `groups`, and other exceptions. The exclusion affects only `acme-rules`'s rule. Globex's rule stays active. Replace the example IDs with rules that exist in your selected libraries.

Keep project exceptions in configuration and `local/`. Keep shared fixes in the library's authored files. Do not edit `vendor/` or `generated/` directly.

## Rebuild and review again

1. After changing only local rules or exceptions, run:

   ```sh
   code-rules project build
   code-rules project check
   ```

   If you also changed a source, revision, or selected groups, run `code-rules project sync` and then `code-rules project check` instead.

2. Read the affected group indexes and full resolved rules again. Confirm that the intended policy is active and no competing obligation remains. Repeat the agent review if the changed scope could affect other groups.

3. Review and commit configuration, local rules, and generated files together. Include vendor changes if sync refreshed them. A clean check confirms that files agree; your review decides whether the guidance can be followed together.
