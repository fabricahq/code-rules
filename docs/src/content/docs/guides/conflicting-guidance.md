---
title: "Resolve conflicting rules"
description: "Find contradictory rules and make the project's intended policy explicit."
---

Rules **conflict** when they require incompatible actions in the same situation. For example, one rule might require interfaces where another requires type aliases. An agent cannot follow both, so you need to decide which guidance your project should use.

Code Rules does not detect contradictions in prose or choose which library takes precedence. In this guide, you'll identify conflicting instructions, review them with an agent, and record your decision by excluding or replacing a rule.

## Recognize a conflict

Consider two illustrative rules:

| Rule ID | Instruction |
| --- | --- |
| `fabrica:techs/typescript/prefer-type-aliases` | Use type aliases for object types. |
| `acme:techs/typescript/prefer-interfaces` | Use interfaces for object types. |

If both rules apply to the same object type, an agent cannot satisfy both.
An organization source does not automatically take precedence over another library.

Different wording alone is not a conflict.
One rule could require interfaces only for public extension points, while another requires type aliases only for internal types.
Those rules can coexist when their scopes are explicit and do not overlap.
Repeated guidance may be redundant without being contradictory.

## Generate a review prompt

`code-rules conflicts --prompt` is not implemented.
Run `code-rules project check` and resolve any reported problems before reviewing the adopted rules.
Then give a repository-aware agent this prompt:

```text
Review this project's resolved Code Rules for conflicting guidance.

Read .code-rules/generated/RULES.md, every part of each group index, and all linked resolved rules to assess conflicts across the complete adopted set.
Use .code-rules/generated/provenance.json to identify the reviewed snapshot.
Compare active rules within and across groups, including local additions
and replacements. Excluded rules and replaced upstream text are context,
not active obligations.

For each possible conflict:
- Cite both source-qualified rule IDs and the relevant passages.
- Describe a concrete situation where both rules apply.
- Explain why their required actions cannot both be satisfied.
- Check applicability and exceptions before calling it a conflict.
- Propose a scoped replacement or exclusion, with its tradeoff.

Separate confirmed contradictions, ambiguous scope, and redundant guidance.
Report which files you reviewed and any missing or unreadable files.
If coverage is incomplete, say so; do not claim a complete review.
A review with no findings is not proof that every conflict has been found.

Present proposed changes for review. Do not modify rules or choose an
unresolved engineering policy on the project's behalf.
```

The paths in this prompt are relative to the project root. Project configuration is stored in `.code-rules/config.json`.
Conflict review runs through your agent. The CLI does not provide a `conflicts` command.

## Resolve the intended policy

First decide which instructions should apply in the overlapping situation.
Then record that choice using the [import configuration](/guides/select-rules/#adapt-the-import-to-your-project):

- **Clarify scope:** Replace a rule with a complete definition that states its intended conditions and exceptions.
- **Keep one policy:** Exclude the competing rule under its owning source and explain why.
- **Define a project policy:** Replace the relevant imported rule with a local definition, and resolve any remaining competing obligations explicitly.

For example, if the project adopts Acme's interface rule, exclude Fabrica's type-alias rule:

```json
{
  "sources": {
    "fabrica": {
      "exclude": {
        "techs/typescript/prefer-type-aliases": "This project follows Acme's interface rule for object types."
      }
    }
  }
}
```

This is a partial configuration snippet.
Merge it into the existing source while retaining its repository, revision selection, groups, and other exceptions.
It affects only Fabrica's rule; Acme's rule stays active.

For a conflicting local addition, edit its authored file under `local/`.
Keep edits in source files and configuration, then rebuild the indexes and resolved definitions.
Do not edit `vendor/` or `generated/` directly.

## Rebuild and review again

Run `code-rules project build` after changing local rules or exceptions.
Run `code-rules project sync` instead if you also change sources, revision selections, or imported groups.
Inspect the regenerated indexes and resolved definitions, then repeat the review against that snapshot.

Commit the configuration, local rules, and generated output together; include vendor changes when sync refreshed them.
A clean `code-rules project check` confirms file consistency, while the agent review assesses whether the guidance can be followed together.
