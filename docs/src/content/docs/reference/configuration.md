---
title: "Configuration"
description: "Fields in the proposed code-rules/config.json format."
---

`code-rules/config.json` records the project's sources, selected groups, and exceptions.
A project can import rules directly from multiple canonical libraries, pinning each one independently.
The following fields describe the proposed first-release interface.

## Complete example

```json
{
  "schemaVersion": 1,
  "sources": {
    "fabrica": {
      "repository": "fabricahq/.code-rules-example",
      "ref": "v1.0.0",
      "groups": [
        "techs/typescript",
        "practices/testing"
      ],
      "exclude": {},
      "replace": {
        "techs/typescript/prefer-type-aliases": {
          "file": "local/techs/typescript/prefer-interfaces.md",
          "reason": "Our public extension API relies on declaration merging."
        }
      }
    },
    "acme": {
      "repository": "acme/.code-rules",
      "ref": "<full Git commit SHA>",
      "groups": [
        "techs/react",
        "practices/testing",
        "practices/observability"
      ],
      "exclude": {},
      "replace": {}
    }
  },
  "localGroups": []
}
```

Repository names, groups, and rule IDs in examples are illustrative.
Replace them with libraries and rules your project can access.

## Fields

| Field | Meaning |
| --- | --- |
| `schemaVersion` | Configuration format version; proposed initial value: `1`. |
| `sources` | Map of stable source names to library configurations. Use an empty object for a project with only local groups. |
| `sources.<name>.repository` | Explicit upstream repository in `owner/name` form. |
| `sources.<name>.ref` | Full Git commit SHA or exact tag name, such as `v1.0.0`. |
| `sources.<name>.groups` | Groups to import from this source, such as `techs/typescript` or `practices/testing`. |
| `localGroups` | Local-only group IDs, each backed by `_group.json` under `local/`. |
| `sources.<name>.exclude` | Map of this library's rule IDs to exclusion reasons. |
| `sources.<name>.replace` | Map of this library's rule IDs to a local `file` and a `reason`. |

Include `localGroups` as an empty array when unused.
Each source includes its own `exclude` and `replace` objects, empty when unused.
Replacement paths resolve relative to the configuration directory and must stay under its `local/` directory.

Sources use `ref` rather than `commit`, and each source owns its groups and exceptions.
The earlier singular `source` and top-level `groups`, `exclude`, and `replace` fields are no longer part of the proposed format.
The schema version remains `1` because no configuration format has shipped.

## Source names and rule identity

Choose stable source names such as `fabrica` or `acme`.
Names must match `[a-z][a-z0-9-]*`; `local` is reserved for project-authored rules.
Declare each repository once, with its own ref and selected groups.

An imported rule's project ID is `<source-name>:<library-rule-id>`:

```text
fabrica:practices/testing/verify-retry-limits
acme:practices/testing/verify-retry-limits
local:practices/testing/test-project-contracts
```

A library rule ID is its path without `.md`.
The source prefix keeps identical paths from different libraries distinct.
Renaming a source changes its generated project rule IDs and any external references to those IDs.
Its nested exclusion and replacement keys remain library-relative.
Replacing a repository under an existing source name also requires reviewing those targets.

## Commit or tag references

Set `ref` to either a full commit SHA or an exact tag name.
A tag may also use the explicit `refs/tags/<name>` form.
Branch names, abbreviated commit SHAs, and version ranges are not supported.
A plain name resolves only as a tag, even when a branch has the same name.
Both lightweight and annotated tags must resolve to a commit.

During `sync`, resolve each configured ref and record its full `resolvedCommit` in that source's vendored provenance.
Fetch rule content and construct source links using the resolved commit.
Configuration records what the project requested; provenance records the exact content it imported.

Tags can move.
An explicit `sync` resolves tags again and reports any change from the previously recorded commit for review.
Offline `build`, `check`, and ordinary agent work use the committed snapshot without resolving tags again.
Use a commit SHA when the configured reference itself must be immutable.

## Source-scoped exceptions

Keys inside a source's `exclude` and `replace` use the library-relative rule ID, without a source prefix.
For example, `sources.fabrica.exclude["practices/testing/verify-retry-limits"]` affects only Fabrica's rule.
An identically named rule from `acme` remains active.
Full paths distinguish matching filenames in different groups of the same library.

Generated files and review findings retain source-qualified IDs because they appear outside the configuration's source nesting.

## Group selection

Select complete groups separately for each source, then exclude individual rules when necessary.
Each selected group must exist in that source at its pinned revision.
Selecting `practices/testing` from two sources combines both sets of rules into one effective testing file.
Source order never establishes precedence.

Local rules can join any imported group.
For a group with no imported source, declare it in `localGroups` and provide local group metadata.
Do not list an imported group in `localGroups`.
Local rules in undeclared groups are errors rather than silently ignored inputs.

The generated index shows linked group names and applicability guidance. When multiple sources contribute to a group, their guidance remains labeled by source.
It also includes local-only group metadata.
The importer does not silently choose one library's description over another's.

## Conflicting rules

Source-qualified IDs prevent naming collisions, but distinct rules can still require incompatible behavior in the same situation.
Both remain active unless the project explicitly excludes or replaces a rule under its owning source.
The importer does not infer priority from source order or interpret prose to resolve contradictions.

See [Conflicting guidance](/guides/conflicting-guidance/) for examples, an agent review prompt, and ways to resolve competing instructions.

## Version boundaries

`schemaVersion` describes this configuration.
Each library's `formatVersion` describes its authoring format.
The importer version describes the executable tool.

Each source's `ref` selects the requested version.
Vendored provenance records the resolved commit used by offline commands; it is importer-owned output, not a second user-selected version.
Direct imports from multiple sources are part of the first-release design.
Libraries that themselves inherit and republish other libraries remain later work.

See [Files and formats](/reference/files/) for library metadata and [Adapt rules](/guides/select-rules/#adapt-the-import-to-your-project) for examples of exceptions.
