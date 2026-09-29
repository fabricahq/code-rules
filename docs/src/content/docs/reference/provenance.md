---
title: "Provenance"
description: "Where to find a rule's source, imported version, replacement reason, and declared license."
---

**Provenance** records where your project's rules came from. The main file is `generated/provenance.json` in the **Code Rules directory**, so its path from the project root is `.code-rules/generated/provenance.json`. It identifies the libraries and versions your project uses, the source of each active rule, and any local replacements.

Read it when you want to understand why a rule is present, investigate a library update, or find out why your project replaced a rule. Tools can also read the JSON to report this information. To find and follow the rules themselves, agents start with [the generated rule index](/reference/files/#where-agents-start).

This page shows where the records live, how Code Rules creates them, and how to trace a rule back to its source.

## Where to look

Code Rules keeps three related records:

| File | What it tells you |
| --- | --- |
| `generated/provenance.json` | Where active rules came from, which rule versions are in use, and why rules were replaced. |
| `vendor/<source-name>/_source.json` | Which commit and original files were imported from one library. |
| `generated/libraries/<source-name>/README.md` | A readable summary of the imported library release, rule versions, and declared license terms. |

These paths are relative to the Code Rules directory, `.code-rules/`. The **source name** is the name you gave a library in your configuration, such as `team`.

Code Rules writes these files. To change the information they describe, edit your configuration or local rules and run the appropriate [sync or build command](/reference/sync/).

## How the records are created

1. **You select libraries and rules.** Configuration records the libraries, selected groups and rules, pins, exclusions, and replacements your project wants to use.
2. **Sync and update record what they import.** `code-rules project sync` and `code-rules project update` copy each library's selected files and write its `_source.json` record with the exact Git commit, rule versions, and file checksums.
3. **Generation records the result.** Sync, update, or build combines the imported and local rules, then writes `generated/provenance.json` alongside the guidance your agents read.

An **active rule** is one included in that generated guidance. An excluded rule has no active rule entry. A local replacement has an entry that also identifies the imported rule it replaced.

These files describe the current result, rather than a running history of every update. They contain no changing timestamps or machine-specific paths.

## Trace a rule to its source

Open `generated/provenance.json` and find the rule's ID in the `rules` array. A rule ID includes its source name, such as `team:practices/testing/check-retries`.

Each rule entry includes:

| Field | What to look for |
| --- | --- |
| `id` and `group` | The active rule's identity and group. |
| `origin` | The source and file supplying the active rule. Imported origins also identify the repository and the exact commit the rule came from. `version` records the rule's version, such as `"1.3.0"`, and `release` the library release that published it; both are `null` when the imported file isn't a published version. |
| `upstream` | The imported rule's origin, including its `version`, when a local rule replaces it; otherwise `null`. |
| `replacementReason` | Your configured reason for the replacement; otherwise `null`. |
| `license`, `licenseBasis`, and `attribution` | Declared terms and source credits, explained below. |

Local origins use `source: "local"`. Their `repository`, `resolvedCommit`, `version`, and `release` fields are `null` because the rule comes from your project. A local fork of a library rule records its source in `attribution` instead.

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
    "file": "practices/testing/check-retries.md",
    "version": "2.1.0"
  },
  "replacementReason": "Use the retry limits required by this service."
}
```

Read this as: agents receive the local `service-retries` rule, it replaces version 2.1.0 of `team`'s `check-retries` rule, and the reason comes from your project configuration. The imported rule's full origin also records its repository and exact commit.

## Inspect library versions and group guidance

The same `generated/provenance.json` file contains three other top-level fields:

| Field | What it records |
| --- | --- |
| `toolVersion` | The Code Rules version that generated the files. |
| `sources` | Each named library, its repository, its `pins` and `ref`, the library release that supplied its group metadata and terms, selected groups, and declared terms. |
| `groups` | Each group's ID, descriptions and reading guidance, and which sources supply the effective guidance. |

For each source, `groupSelection` and `ruleSelection` record what you asked for, while `groups` lists the groups imported. For example, `"practices/*"` asks for all practice groups; the list records which ones existed in the imported library.

Within each group record, `guidance` keeps the metadata labeled by source. `effectiveGuidanceSources` identifies which sources supply the guidance agents see. Local group metadata takes precedence when present; otherwise the guidance from all contributing libraries remains effective.

## Inspect the original imported files

Each library has a separate `vendor/<source-name>/_source.json` file. It describes the **snapshot**: the original library files the project imported, and which version of each rule they belong to. It works like a lockfile: `code-rules project sync` restores exactly what it records. The directory name identifies the source.

```json
{
  "formatVersion": 2,
  "repository": "https://github.com/fabricahq/public-rules.git",
  "pins": { "practices/testing/verify-backoff": { "version": "1.3.0", "reason": "Waiting on the author's response to #45." } },
  "release": 3,
  "resolvedCommit": "9e07b3d6f0c1a4b85e2d7c3f9a61b04e8d52c7aa",
  "groupSelection": ["practices/testing"],
  "ruleSelection": ["techs/go/wrap-errors-with-operation"],
  "groups": ["practices/testing"],
  "rules": {
    "practices/testing/verify-backoff": { "version": "1.3.0", "release": 2, "commit": "4f1c2a95…" },
    "practices/testing/verify-retry-limits": { "version": "2.0.0", "release": 3, "commit": "9e07b3d6…" },
    "techs/go/wrap-errors-with-operation": { "version": "1.1.0", "release": 3, "commit": "9e07b3d6…" }
  },
  "files": { "practices/testing/verify-retry-limits.md": "4c1f…e9a2", "…": "…" }
}
```

| Field | Meaning |
| --- | --- |
| `formatVersion` | The snapshot format version, `2`. |
| `repository` | The library's repository address. |
| `pins` and `ref` | The source's pins and `ref` from configuration when the snapshot was recorded. Each is omitted when configuration has none. |
| `release` | The newest library release among the imported rule versions, or the library release your `ref` names. It supplies the group metadata and license files. Omitted when your `ref` isn't a library release. |
| `resolvedCommit` | The full Git commit SHA of that library release, or of the revision your `ref` names. |
| `rules` | Each imported rule's ID, whether imported through a group or individually selected, its `version`, the `release` that published it, and that library release's full `commit`. |
| `groupSelection` | Your configured group list or selector: `"*"`, `"practices/*"`, or `"techs/*"`. |
| `ruleSelection` | Your configured list of individually selected rules. Omitted when you select none. |
| `groups` | The groups imported in full. A group reached only through individually selected rules isn't listed. |
| `files` | Each retained path and its SHA-256 checksum, written as lowercase hexadecimal text. Files of rules from a different library release than `release` are stored under `_releases/<number>/`. |

A **checksum** detects whether a file's contents differ from the recorded copy. The `files` map covers the original file bytes and excludes `_source.json` itself.

A rule's `version` and `release` are `null` when the imported file isn't a published version, which can happen only when your `ref` names a revision other than a library release. Generated guidance shows no version for it.

For a wildcard selection, the snapshot must contain every group in the selected scope and record the exact selector. Older records without `groupSelection` imply the explicit `groups` list; they cannot satisfy a wildcard selection.

### What offline checks can verify

`code-rules project build` and `code-rules project check` compare the recorded library selection with configuration and compare stored files with their checksums. Version checks depend on how the source chooses versions:

| Choice | What Code Rules verifies offline |
| --- | --- |
| Newest versions, the default | Each pinned rule records its pinned version. Other rules may record any published version. |
| One revision, with `ref` | For a commit SHA, `resolvedCommit` equals it. For a library release, every rule records the version that library release published. A tag's recorded commit is used without checking where the tag points now. |

For every source, offline checks also verify that `rules` lists exactly the imported rules, and that generated provenance and guidance show the same versions.

Offline checks cannot prove that a recorded version was the newest available, or that recorded versions match the library's release tags. These records also cannot authenticate files against the remote repository if someone changed both the local files and their records.

## Find declared licenses and source credits

Provenance keeps license declarations and attribution alongside rule origins. Each source and rule has one `license` object, or `null` when no license was declared.

An imported rule from a library with a declared license records that declaration and `licenseBasis: "library"`. Within the license object, `spdxExpression` records a declared license expression, such as `MIT`. The rule's `attribution` field preserves source credits as URLs and descriptions.

The license record distinguishes original files from the copies retained with generated guidance:

| Field inside `license` | Paths in a source record | Paths in a rule record |
| --- | --- | --- |
| `files` | Original license files, relative to the library root. | Original license files under `vendor/<source-name>/`, relative to the Code Rules directory. |
| `attributionFiles` | Original notice files, relative to the library root. | Original notice files under `vendor/<source-name>/`, relative to the Code Rules directory. |
| `generatedFiles` | Retained license copies, relative to `generated/`. | The same generated license copies. |
| `generatedAttributionFiles` | Retained notice copies, relative to `generated/`. | The same generated notice copies. |

For example, a rule's `files` entry might be `vendor/team/LICENSE.md`, with `libraries/team/licenses/LICENSE.md` at the same position in `generatedFiles`. Generated rule links point to the retained copy. Notice paths correspond in the same way.

The source record also has a `licenseFiles` list of the library-relative files belonging to its declaration. These lists describe files for one library-wide license declaration.

Local additions and original local replacements have `license: null` and `licenseBasis: "undeclared"`. Replacing an imported rule does not automatically assign its library's license to the local replacement. The library's declaration remains in its source record.

These fields preserve declarations; they are not a legal verification status. `"undeclared"` does not mean public domain or permission to redistribute. For how authors declare terms and credits, see [License metadata](/reference/library-format/#license-metadata) and [Rule attribution](/reference/rule-format/#rule-attribution).
