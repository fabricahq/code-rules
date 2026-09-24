---
title: "Provenance"
description: "Find a rule's source, imported version, replacement reason, and declared license."
---

**Provenance** tells you why a rule appears in your project's guidance and where it came from. To trace an imported rule, review a library update, or understand a local replacement, open `.code-rules/generated/provenance.json`. To read the rules themselves, start at [the generated rule index](/reference/files/#where-agents-start).

## Where to look

| File under `.code-rules/` | What it answers |
| --- | --- |
| `generated/provenance.json` | Which rules are active, where they came from, and which imported rules were replaced. |
| `vendor/<source-name>/_source.json` | Which commit and original files were copied from one library. |
| `generated/libraries/<source-name>/README.md` | A readable summary of that library's revision and declared terms. |

The source name is the name you gave a library in configuration, such as `acme-rules`. Code Rules writes these files. Change configuration or local rules, then [sync or build](/reference/sync/) to update them.

## How the records are created

Configuration names your libraries, revisions, groups, and exceptions. Sync copies the selected library files and records the exact imported commit. Sync or build then writes provenance for the resulting active rules. An excluded rule has no active entry; a local replacement has an entry naming the imported rule it replaced.

These records describe the current ruleset, not a history of every update. They contain no changing timestamps or machine-specific paths.

## Trace a rule to its source

In `generated/provenance.json`, find the rule's ID in the `rules` list. An imported ID includes its source name, such as `acme-rules:practices/testing/check-retries`. Read `origin` for the source file, repository, requested ref or selected tag, and exact commit. A local rule has `source: "local"` and no repository commit.

### Example: explain a local replacement

Suppose `acme-rules` supplies `practices/testing/check-retries.md`, but your project uses `local/practices/testing/service-retries.md` instead. A shortened provenance entry looks like this:

```json
{
  "id": "local:practices/testing/service-retries",
  "origin": {
    "source": "local",
    "file": "practices/testing/service-retries.md"
  },
  "upstream": {
    "source": "acme-rules",
    "file": "practices/testing/check-retries.md"
  },
  "replacementReason": "Use the retry limits required by this service."
}
```

Agents read the local rule. The `upstream` entry and reason show what it replaces and why. The full entry also records the imported rule's repository and exact commit.

## Inspect library versions and group guidance

In the `sources` list, find your source name. It records the requested `ref` or version range, the imported commit, and the groups included. For a version range, it also records the selected tag and version. A selector such as `"practices/*"` can include newly published groups on a later sync; compare the recorded groups when reviewing an update.

The `groups` list records group descriptions and reading cues by source. Local group metadata supplies the effective description when present. Otherwise, guidance from each contributing library remains available.

## Inspect the original imported files

Open `vendor/<source-name>/_source.json` to see the original files copied from one Git commit. The record lists your requested group selection, the groups found at that revision, and the imported file paths. The files remain under `vendor/<source-name>/`, even if your project excludes or replaces their rules.

### What offline checks can verify

`code-rules project build` and `code-rules project check` work from stored imports. They detect missing or changed imported files and a selection that no longer matches configuration. They cannot tell whether a remote tag moved or whether a newer matching version exists. The stored records also cannot authenticate a remote source if someone changes both the files and their records. Run [sync](/reference/sync/) to fetch the selected revision again.

## Find declared licenses and source credits

For an imported rule, provenance records the library's license declaration when present and keeps the rule's authored attribution. Generated rules link to retained copies of declared license and notice files under `generated/libraries/<source-name>/licenses/`.

A local addition or replacement does not inherit the imported library's license. Its rule entry has `license: null` and `licenseBasis: "undeclared"`, while the library's declaration remains in the source record. An absent declaration does not imply public-domain status or permission to redistribute. For authoring fields and retained terms, see [Rule and library format](/reference/rule-library-format/).

For the code paths and tests behind provenance, [inspect the implementation](/for-agents/#inspect-implementation-and-tests).
