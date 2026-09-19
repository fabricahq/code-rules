---
title: "Use rules in a project"
description: "Walk through the import workflow, from choosing libraries to directing agents."
---

[Install the CLI](/start-here/install/) before following these commands.

## Start locally

Start with [project setup](/start-here/set-up-project/) to create a local group and rule, then run `code-rules project build`. Open `.code-rules/generated/RULES.md` to inspect the output. This local-only workflow fetches no libraries.

## 1. Choose libraries and revisions

Choose one or more canonical libraries whose rules your project is authorized to use.
For example, combine a shared community library with your organization's rules.
Give each source a stable name. Select an exact commit or tag with `ref`, or a HashiCorp version constraint with `version`. Specify exactly one.
An upstream edit should arrive through an explicit update.

## 2. Select technologies and practices

Start with [project setup](/start-here/set-up-project/), or create `.code-rules/config.json` manually:

```json
{
  "schemaVersion": 1,
  "sources": {
    "fabrica": {
      "repository": "https://github.com/fabricahq/.code-rules-example.git",
      "ref": "v1.0.0",
      "groups": [
        "techs/typescript",
        "practices/testing"
      ],
      "exclude": {},
      "replace": {}
    },
    "acme": {
      "repository": "https://github.com/acme/.code-rules.git",
      "ref": "<full Git commit SHA>",
      "groups": [
        "techs/react",
        "practices/testing",
        "practices/observability"
      ],
      "exclude": {},
      "replace": {}
    }
  }
}
```

Replace the illustrative repositories, refs, and groups with libraries and rules your project can access.
Read each source's group descriptions before selecting its groups.
The example imports testing rules from both sources; both contribute to the generated testing file.
Generated rule IDs include the source name, so matching paths do not collide or imply an override.
Exclusions and replacements live inside their owning source and use that library's rule IDs.

Keep project-only rules under `.code-rules/local/`.
For a group supplied only locally, create `.code-rules/local/<group-id>/_group.json`. No configuration entry is needed.
Keep that metadata when importing a library with the same group: it remains the project's chosen description, and imported rules join the group.
The complete configuration contract is in [Configuration](/reference/configuration/).

## 3. Import the rules

Run:

```sh
code-rules project sync
```

Sync resolves each ref to a full commit SHA and records it as `resolvedCommit` in `.code-rules/vendor/<source-name>/_source.json`.
It copies the selected groups from that commit into `.code-rules/vendor/<source-name>/`.
Sync passes snapshots and project inputs to the builder, which resolves them and produces the root group index, a page per group with full rules or applicability summaries, and individual full rule files under `.code-rules/generated/rules/`.
Project exclusions and replacements are already applied.
For example, `.code-rules/generated/groups/practices/testing.md` includes complete testing rules when the page fits the inline limit. Larger pages provide summaries and reading links.
`.code-rules/generated/RULES.md` helps agents choose groups; oversized indexes link to complete numbered parts.
The import installs all sources together after validation succeeds.
A later sync can pick up a moved tag; review resolved-commit changes along with the rule changes.
[Adapt rules](/guides/select-rules/#adapt-the-import-to-your-project) when the project needs additions or exceptions, then rebuild the output.

## 4. Review and commit the files

Group pages separate **How to use this group** from **Rules**, with each rule nested under **Rules**.
Review the generated group pages. Read applicable rules in full where included; otherwise follow each **Read full rule** link.
Full definitions put **Guidance** first, followed by **Source and attribution**, including the rule source, declared library license links, and preserved source metadata. If no library license is declared, the footer omits the license entry. Provenance uses a singular `license` field containing one declaration or `null`.
Inspect `.code-rules/generated/libraries/<source-name>/README.md` for each library’s identity, revision, and links to declared terms.
License and notice copies live under that library’s `licenses/` directory; provenance records their original and generated paths.
Confirm that replacements contain the intended obligations and that excluded rules are absent from active output.
Commit configuration, local rules, vendor content, and generated files together.

Ordinary coding and review can then read the rules offline.

## 5. Point agents at the index

Add the [rule-loading snippet](/for-agents/#project-instructions) to the project's existing `AGENTS.md`.
Keep other project instructions intact.
The generated index is the entry point for selecting both technologies and practices.

## 6. Check consistency in CI

The `check` command verifies that committed inputs produce the committed output.
The check does not inspect application code for compliance.
A reviewing agent performs that separate assessment using the same resolved rules.
