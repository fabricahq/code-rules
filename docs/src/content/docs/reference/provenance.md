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
| `basedOn` | The library version of the replaced rule that the local rule is based on, as the exclusion's [`basedOn`](/reference/configuration/#exclude-or-replace-a-rule) records it, such as the version a fork copied; otherwise `null`. It can differ from `upstream`'s `version`, which is the version the project imports. |
| `license`, `licenseBasis`, and `attribution` | Declared terms and source credits, explained below. |

Local origins use `source: "local"`. Their `repository`, `resolvedCommit`, `version`, and `release` fields are `null` because the rule comes from your project. A [fork](/reference/cli/#fork-a-library-rule) of a library rule records its source in `attribution` instead, when the library is on GitHub.com or GitLab.com or its repository address is an HTTPS URL.

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
  "replacementReason": "Use the retry limits required by this service.",
  "basedOn": "2.0.0"
}
```

Read this as: agents receive the local `service-retries` rule, it replaces version 2.1.0 of `team`'s `check-retries` rule, it is based on the library's version 2.0.0, and the reason comes from your project configuration. The imported rule's full origin also records its repository and exact commit.

## Inspect library versions and group guidance

The same `generated/provenance.json` file contains four other top-level fields:

| Field | What it records |
| --- | --- |
| `generatedNotice` | A reminder that Code Rules generates the file, and how to regenerate it. |
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
  "checksum": "5d1b…c07e",
  "repository": "https://github.com/fabricahq/public-rules.git",
  "release": 3,
  "resolvedCommit": "9e07b3d6f0c1a4b85e2d7c3f9a61b04e8d52c7aa",
  "groupSelection": [
    "practices/testing"
  ],
  "ruleSelection": [
    "techs/go/wrap-errors-with-operation"
  ],
  "groups": [
    "practices/testing"
  ],
  "rules": {
    "practices/testing/verify-backoff": {
      "version": "1.3.0",
      "release": 2,
      "commit": "4f1c2a95…"
    },
    "practices/testing/verify-retry-limits": {
      "version": "2.0.0",
      "release": 3,
      "commit": "9e07b3d6…"
    },
    "techs/go/wrap-errors-with-operation": {
      "version": "1.1.0",
      "release": 3,
      "commit": "9e07b3d6…"
    }
  },
  "retiredRules": [
    "practices/testing/check-retry-backoff"
  ],
  "files": {
    "practices/testing/verify-retry-limits.md": "4c1f…e9a2",
    "…": "…"
  }
}
```

| Field | Meaning |
| --- | --- |
| `formatVersion` | The snapshot format version, `2`. An older format, such as `1`, fails with advice to delete `.code-rules/vendor/` and run `code-rules project sync`, which records the sources again. With no record to restore, that sync imports each rule's newest version, except rules you pin and sources that set `ref`, so it can move rules to versions your project didn't use before; pin a rule, or set `ref: release/<number>`, first to keep what you had. A newer format, written by a later Code Rules, fails with `unsupported-source-record` and asks you to upgrade Code Rules instead, since syncing would rewrite the project in the older format. Unknown fields are always rejected. |
| `checksum` | The SHA-256 checksum of the record's other fields, as sync wrote them. Offline, `code-rules project build` and `code-rules project check` refuse a record whose checksum doesn't match, or that has none, as changed outside `code-rules project sync`, such as by hand or in a merge resolution, because its versions and releases can't be trusted without the library. `code-rules project sync`, `code-rules project update`, and a [fork](/reference/cli/#fork-a-library-rule) refuse it too and write nothing, as sync does when it removes a source whose record holds the group metadata a local rule needs; a removed source whose record no local rule needs is discarded with the rest of its files: it never records again a record it didn't write, whatever the record says, since no check of individual facts can establish that an edited record is right. Records from earlier preview builds, which have no checksum, are refused the same way. To recover, restore a record sync wrote with `git checkout -- .code-rules/vendor/<source-name>/_source.json`, or, during a merge conflict, take one side with `git checkout --ours` or `git checkout --theirs` for that file, then run `code-rules project sync`, which applies any configuration changes; or delete `.code-rules/vendor/<source-name>/` and run `code-rules project sync`, which imports the source again, with its unpinned rules at their newest versions. To resolve a merge conflict in `_source.json`, take one side of the file rather than combining them, and let sync apply the merged configuration. A record that sync can't read at all, such as one with an unknown field, fails with advice to delete `.code-rules/vendor/<source-name>/` and sync, which imports the source again, with its unpinned rules at their newest versions. |
| `repository` | The library's repository address. |
| `ref` | The source's `ref` from configuration when the snapshot was recorded, omitted when configuration has none. Changing it to another spelling of the same revision, such as `refs/tags/release/2` for `release/2`, keeps the recorded spelling, so the record stays current. Pins and exclusions aren't recorded: configuration owns them, and a pinned rule's `version` in `rules` is the pinned version. So adding, rewording, or removing a pin that moves nothing, or an exclusion, by hand or with a fork, never makes the record out of date, unless the entry names a retired rule that `retiredRules` doesn't list yet; then offline checks ask for `code-rules project sync`, which records the retirement. Generated guidance and `generated/provenance.json` show the configured pins and their reasons. |
| `release` | The library release that supplied the library-wide files: the group metadata, shared assets, and license files. It is the newest library release when the source was added, selected more rules, or last ran a full `code-rules project update`, and never older than the `release` of any imported rule; plain `code-rules project sync` keeps it. With `ref`, it's the library release your `ref` names, and it's omitted when your `ref` isn't a library release. |
| `resolvedCommit` | The full Git commit SHA of that library release, or of the revision your `ref` names. |
| `rules` | Each imported rule's ID, whether imported through a group or individually selected, its `version`, the `release` that published it, and that library release's full `commit`. |
| `retiredRules` | The IDs of the rules the library retired that the source would otherwise import, sorted, and empty when there are none. A retired rule the source still imports, such as one a pin keeps at its last version, is in both `rules` and `retiredRules`, which says it's retired, and the generated library README marks it so. The rules it lists are those your `groups` or `rules` select, and those the previous record imported or listed, so an exclusion left after you deselect a retired rule still only warns. It records what the library told sync, not your configuration, so offline checks can tell an exclusion of a retired rule from a typo. `code-rules project update` refreshes it; `code-rules project sync` keeps it byte for byte unless the selection, `ref`, or `release` changes, so a pin or exclusion edit leaves `_source.json` unchanged when the record already covers the rule it names. When you pin or exclude a retired rule the record doesn't list yet, sync checks it against the library and adds it. |
| `groupSelection` | Your configured group list or selector: `"*"`, `"practices/*"`, or `"techs/*"`. |
| `ruleSelection` | Your configured list of individually selected rules. Omitted when you select none. |
| `groups` | The groups imported in full. A group reached only through individually selected rules isn't listed. |
| `files` | Each retained path and its SHA-256 checksum, written as lowercase hexadecimal text. Each file is stored at its path in the library, including the files of rules from a library release other than `release`, because the snapshot holds one version of each rule. |

A **checksum** detects whether a file's contents differ from the recorded copy. The `files` map covers the original file bytes and excludes `_source.json` itself.

A rule's `version` and `release` are `null` when the imported file isn't a published version, which can happen only when your `ref` names a revision other than a library release. Generated guidance shows no version for it.

For a wildcard selection, the snapshot must contain every group in the selected scope and record the exact selector.

### What offline checks can verify

`code-rules project build` and `code-rules project check` compare the recorded library selection with configuration and compare stored files with their checksums. Version checks depend on how the source chooses versions:

| Choice | What Code Rules verifies offline |
| --- | --- |
| Newest versions, the default | Each pinned rule records its pinned version. Other rules may record any published version. `release` is at least as new as every rule's `release`. |
| One revision, with `ref` | For a commit SHA, `resolvedCommit` equals it. For a library release `release/N`, the source records release N, and every rule records a published version from library release N or earlier. A tag's recorded commit is used without checking where the tag points now. |

For every source, offline checks first verify the record's `checksum`, so a record changed outside sync fails until you restore a record sync wrote or import the source again. They also verify that `rules` lists exactly the imported rules, and that generated provenance and guidance show the same versions. Each exclusion must name a rule in `rules` or `retiredRules`; any other exclusion could be a typo that leaves the rule you meant active, so the check reports that it names no rule the library supplies and asks you to run `code-rules project sync`, which checks it against the library.

Offline checks cannot prove that a recorded version was the newest available, or that recorded versions match the library's release tags; `code-rules project sync` checks the latter against the release records before it imports anything. These records also cannot authenticate files against the remote repository if someone changed both the local files and their records.

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

For example, a rule's `files` entry might be `vendor/team/LICENSE.md`, with `libraries/team/licenses/LICENSE.md` at the same position in `generatedFiles`. Generated rule links point to the retained copy. Notice paths correspond in the same way, by position in `attributionFiles` and `generatedAttributionFiles`; generated notice copies are numbered, such as `libraries/team/licenses/notices/001.md`.

The source record also has a `licenseFiles` list of the library-relative files belonging to its declaration. These lists describe files for one library-wide license declaration.

Local additions and original local replacements have `license: null` and `licenseBasis: "undeclared"`. Replacing an imported rule does not automatically assign its library's license to the local replacement. The library's declaration remains in its source record.

These fields preserve declarations; they are not a legal verification status. `"undeclared"` does not mean public domain or permission to redistribute. For how authors declare terms and credits, see [License metadata](/reference/library-format/#license-metadata) and [Rule attribution](/reference/rule-format/#rule-attribution).
