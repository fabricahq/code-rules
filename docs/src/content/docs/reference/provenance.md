---
title: "Provenance"
description: "Inspect imported revisions, resolved rule origins, replacements, and retained license declarations."
---

Use provenance records to trace a rule to its source, investigate an unexpected update, or build an integration.
You do not need to read them to apply rules. Agents should start with the [generated rule index](/reference/files/#where-agents-start).

Paths below are relative to the configuration directory, normally `.code-rules/`. Code Rules writes these records; do not edit them manually.

| To find out... | Read... |
| --- | --- |
| Which commit and files were imported | `vendor/<source-name>/_source.json` |
| Where an active rule came from, or why it was replaced | `generated/provenance.json` |
| A library's revision and declared terms in a readable summary | `generated/libraries/<source-name>/README.md` |

## Imported snapshots

Each `vendor/<source-name>/_source.json` describes one imported library. The containing directory supplies the source name.

| Field | Meaning |
| --- | --- |
| `formatVersion` | Snapshot format version, currently `1`. |
| `repository` | The source repository. |
| `ref` or `version` | The requested revision or version constraint. |
| `resolvedCommit` | The full commit SHA that was imported. |
| `groupSelection` | The configured group list or wildcard selector. |
| `groups` | The concrete groups included in the snapshot. |
| `files` | Retained library-relative paths mapped to lowercase SHA-256 hex digests of their original bytes. |

The `files` map excludes `_source.json` itself. Snapshot records contain no timestamps or machine-specific paths.

`groupSelection` is the configured list or selector (`"*"`, `"practices/*"`, or `"techs/*"`). Pattern snapshots must record the exact selector and supply every group within its scope at that revision.
Legacy snapshots without this field imply the explicit `groups` list; they cannot satisfy a wildcard selection.
Offline build and check verify the recorded source selection against configuration and the retained files against their digests.
For a commit-valued ref, the resolved commit must equal that SHA.
For a tag, offline checks use the recorded commit and do not verify the tag's current remote target.
Version-constraint snapshots additionally require `resolvedTag` and `resolvedVersion`. Code Rules verifies that the tag represents the recorded version and that the version satisfies the configured constraint.
It cannot verify offline that this was the highest available version or authenticate the tag-to-commit association. Exact-ref snapshots omit version-selection fields.

## Resolved rule origins

`generated/provenance.json` records the tool version, every named source with its requested ref or constraint and resolved commit, and active rule origins.
Version sources also record `resolvedTag` and `resolvedVersion`. Imported rule origins use the selected tag as `ref`, while the source record retains the requested constraint.
Each source records `groupSelection` alongside the expanded `groups` list, keeping the adoption intent and actual included groups visible.
The top-level `groups` array records each group ID, all source-labeled `guidance` metadata, and `effectiveGuidanceSources`. Local guidance wins when present; otherwise all library guidance remains effective.
A replacement record uses the local rule ID and origin. Its `upstream` field identifies the replaced source rule; `replacementReason` records the project decision.
Generated output excludes changing timestamps and machine-specific paths.

These metadata files support repeatability and diagnosis.
They do not independently authenticate a locally modified snapshot against its remote repository.

## License declarations

Each source and rule has a singular `license` field: one declaration object, or `null` when undeclared.

Source records include `license` with library-relative paths and a `licenseFiles` list. The declaration is singular; the lists identify files belonging to that declaration.
Each active imported rule records that same declaration, its `attribution`, and `licenseBasis: "library"`. Its original `files` and `attributionFiles` paths start with `vendor/<source>/` and resolve from the configuration directory, above `generated/`. Both source and rule declarations add `generatedFiles` and `generatedAttributionFiles`, relative to `generated/`. Each generated path corresponds to the original path at the same array position. Generated rule links use these standardized copies.
Local additions and original local replacements have no imported library license: their `license` is `null` and `licenseBasis` is `undeclared`. An upstream rule ID does not assign the upstream license to a local replacement. Its source library's declaration remains in the source record.

For example, a rule imported from a library declaring MIT can have this provenance:

```json
{
  "licenseBasis": "library",
  "license": {
    "spdxExpression": "MIT",
    "files": ["vendor/licensed/LICENSE.md"],
    "attributionFiles": ["vendor/licensed/NOTICE.md"],
    "generatedFiles": ["libraries/licensed/licenses/LICENSE.md"],
    "generatedAttributionFiles": ["libraries/licensed/licenses/notices/001.md"]
  },
  "attribution": [{
    "url": "https://github.com/sindresorhus/eslint-plugin-unicorn/blob/5d9d745c5365b6fdb824db1122ff982dd824b11a/docs/rules/no-for-each.md",
    "description": "Adapted from Sindre Sorhus's ESLint Unicorn rule; added task guidance."
  }]
}
```

These are preserved declarations, not a legal verification status. `licenseBasis: "undeclared"` means no applicable terms were declared to Code Rules; it does not mean public domain or permission to redistribute.

For refresh commands and snapshot errors, see [Sync and recovery](/reference/sync/). For authoring declarations, see [Rule and library format](/reference/rule-library-format/).
