---
title: "Provenance"
description: "Where to find a rule's source, imported version, replacement reason, and declared license."
---

**Provenance** records where your project's rules came from. The main file is `.code-rules/generated/provenance.json`. It identifies the libraries and versions your project uses, the source of each active rule, and any local replacements.

Read it when you want to understand why a rule is present, investigate a library update, or find out why your project replaced a rule. Tools can also read the JSON to report this information. To find and follow the rules themselves, agents start with [the generated rule index](/reference/files/#where-agents-start).

This page shows where the records live, how Code Rules creates them, and how to trace a rule back to its source.

## Where to look

Code Rules keeps three related records:

| File | What it tells you |
| --- | --- |
| `generated/provenance.json` | Where active rules came from, which library versions are in use, and why rules were replaced. |
| `vendor/<source-name>/_source.json` | Which commit and original files were imported from one library. |
| `generated/libraries/<source-name>/README.md` | A readable summary of one library's revision and declared license terms. |

These paths are relative to the configuration directory, normally `.code-rules/`. The **source name** is the name you gave a library in your configuration, such as `team`.

Code Rules writes these files. To change the information they describe, edit your configuration or local rules and run the appropriate [sync or build command](/reference/sync/).

## How the records are created

1. **You select libraries and rules.** Configuration records the library versions, groups, exclusions, and replacements your project wants to use.
2. **Sync records what it imports.** `code-rules sync` copies each library's selected files and writes its `_source.json` record with the exact Git commit and file checksums.
3. **Generation records the result.** Sync or build combines the imported and local rules, then writes `generated/provenance.json` alongside the guidance your agents read.

An **active rule** is one included in that generated guidance. An excluded rule has no active rule entry. A local replacement has an entry that also identifies the imported rule it replaced.

These files describe the current result, rather than a running history of every update. They contain no changing timestamps or machine-specific paths.

## Trace a rule to its source

Open `generated/provenance.json` and find the rule's ID in the `rules` array. A rule ID includes its source name, such as `team:practices/testing/check-retries`.

Each rule entry includes:

| Field | What to look for |
| --- | --- |
| `id` and `group` | The active rule's identity and group. |
| `origin` | The source and file supplying the active rule. Imported origins also identify the repository, requested ref or selected tag, and exact commit. |
| `upstream` | The imported rule's origin when a local rule replaces it; otherwise `null`. |
| `replacementReason` | Your configured reason for the replacement; otherwise `null`. |
| `license`, `licenseBasis`, and `attribution` | Declared terms and source credits, explained below. |

Local origins use `source: "local"`. Their repository, revision, and commit fields are `null` because the rule comes from your project.

### Example: explain a local replacement

Suppose your project imports the `team` library but replaces its `check-retries` rule with `local/practices/testing/service-retries.md`. You record the reason in configuration: “Use the retry limits required by this service.”

The replacement's entry contains these fields. This excerpt omits the other origin and license fields:

```json
{
  "id": "local:practices/testing/service-retries",
  "group": "practices/testing",
  "origin": {
    "source": "local",
    "file": "practices/testing/service-retries.md"
  },
  "upstream": {
    "source": "team",
    "file": "practices/testing/check-retries.md"
  },
  "replacementReason": "Use the retry limits required by this service."
}
```

Read this as: agents receive the local `service-retries` rule, it replaces `team`'s `check-retries` rule, and the reason comes from your project configuration. The imported rule's full origin also records its repository and exact commit.

## Inspect library versions and group guidance

The same `generated/provenance.json` file contains three other top-level fields:

| Field | What it records |
| --- | --- |
| `toolVersion` | The Code Rules version that generated the files. |
| `sources` | Each named library, its repository, requested revision or version range, exact imported commit, selected groups, and declared terms. |
| `groups` | Each group's ID, descriptions and reading guidance, and which sources supply the effective guidance. |

For each source, `groupSelection` records what you asked for, while `groups` lists the groups imported. For example, `"practices/*"` asks for all practice groups; the list records which ones existed at the imported revision.

When you select a version range, the source also records `resolvedTag` and `resolvedVersion`. The source keeps the requested range in `version`; individual imported rule origins use the selected tag as `ref`.

Within each group record, `guidance` keeps the metadata labeled by source. `effectiveGuidanceSources` identifies which sources supply the guidance agents see. Local group metadata takes precedence when present; otherwise the guidance from all contributing libraries remains effective.

## Inspect the original imported files

Each library has a separate `vendor/<source-name>/_source.json` file. It describes the **snapshot**: the original library files copied from one Git commit. The directory name identifies the source.

| Field | Meaning |
| --- | --- |
| `formatVersion` | The snapshot format version, `1`. |
| `repository` | The library's repository address. |
| `ref` or `version` | The exact revision or version range you requested. |
| `resolvedCommit` | The full Git commit SHA imported. |
| `resolvedTag` and `resolvedVersion` | The tag and version selected for a version range. Omitted for an exact ref. |
| `groupSelection` | Your configured group list or selector: `"*"`, `"practices/*"`, or `"techs/*"`. |
| `groups` | The groups included in the snapshot. |
| `files` | Each retained library-relative path and its SHA-256 checksum, written as lowercase hexadecimal text. |

A **checksum** detects whether a file's contents differ from the recorded copy. The `files` map covers the original file bytes and excludes `_source.json` itself.

For a wildcard selection, the snapshot must contain every group in the selected scope at that revision and record the exact selector. Older records without `groupSelection` imply the explicit `groups` list; they cannot satisfy a wildcard selection.

### What offline checks can verify

`code-rules build` and `code-rules check` compare the recorded library selection with configuration and compare stored files with their checksums. Revision checks depend on how you selected the library:

| Selection | What Code Rules verifies offline |
| --- | --- |
| Exact commit | `resolvedCommit` equals the requested commit. |
| Exact tag | Uses the recorded commit without checking where the remote tag points now. |
| Version range | The recorded tag represents `resolvedVersion`, and that version satisfies your configured range. |

Offline checks cannot prove that a selected version was the highest available or that a remote tag points to the recorded commit. These records also cannot authenticate files against the remote repository if someone changed both the local files and their records.

## Find declared licenses and source credits

Provenance keeps license declarations and attribution alongside rule origins. Each source and rule has one `license` object, or `null` when no license was declared.

An imported rule from a library with a declared license records that declaration and `licenseBasis: "library"`. Within the license object, `spdxExpression` records a declared license expression, such as `MIT`. The rule's `attribution` field preserves source credits as URLs and descriptions.

The license record distinguishes original files from the copies retained with generated guidance:

| Field inside `license` | Paths in a source record | Paths in a rule record |
| --- | --- | --- |
| `files` | Original license files, relative to the library root. | Original license files under `vendor/<source-name>/`, relative to the configuration directory. |
| `attributionFiles` | Original notice files, relative to the library root. | Original notice files under `vendor/<source-name>/`, relative to the configuration directory. |
| `generatedFiles` | Retained license copies, relative to `generated/`. | The same generated license copies. |
| `generatedAttributionFiles` | Retained notice copies, relative to `generated/`. | The same generated notice copies. |

For example, a rule's `files` entry might be `vendor/team/LICENSE.md`, with `libraries/team/licenses/LICENSE.md` at the same position in `generatedFiles`. Generated rule links point to the retained copy. Notice paths correspond in the same way.

The source record also has a `licenseFiles` list of the library-relative files belonging to its declaration. These lists describe files for one library-wide license declaration.

Local additions and original local replacements have `license: null` and `licenseBasis: "undeclared"`. Replacing an imported rule does not automatically assign its library's license to the local replacement. The library's declaration remains in its source record.

These fields preserve declarations; they are not a legal verification status. `"undeclared"` does not mean public domain or permission to redistribute. For how authors declare terms and credits, see [Rule and library format](/reference/rule-library-format/).
