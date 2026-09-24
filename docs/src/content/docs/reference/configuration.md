---
title: "Configuration"
description: "Choose library sources, revisions, groups, and project exceptions in config.yaml."
---

Your project's `.code-rules/config.yaml` records the libraries and groups it uses. It also records exclusions and local replacements. Run `code-rules project init` from the project root to create this file. For a project with only local rules, use `sources: {}`.

## Complete example

```yaml
schemaVersion: 1
sources:
  acme-rules:
    repository: https://github.com/YOUR-ORG/YOUR-RULES.git
    version: ">= 1.2.0, < 2.0.0"
    groups:
      - practices/testing
      - techs/typescript
    exclude: {}
    replace: {}
```

The example has every required field, but its repository, version range, and group paths are illustrative. Replace them with a library you can access and groups that exist at the selected revision. Run `code-rules project sync` after changing a source, its revision, or its groups. Then run `code-rules project check` and review the generated rules. For the command sequence, follow [Import rules](/guides/select-rules/).

## Fields

| Field | What to enter |
| --- | --- |
| `schemaVersion` | `1`, the supported configuration format. |
| `sources` | A map of source names to libraries, or `{}` for a local-only project. |
| `sources.<name>.repository` | A complete HTTPS or SSH Git address. |
| `sources.<name>.ref` | An exact tag or full commit SHA. Use this **or** `version`. |
| `sources.<name>.version` | A version constraint such as `">= 1.2.0, < 2.0.0"`. Use this **or** `ref`. |
| `sources.<name>.groups` | An array of group paths, or one supported wildcard string. |
| `sources.<name>.exclude` | Library-relative rule IDs mapped to reasons for leaving them out. |
| `sources.<name>.replace` | Library-relative rule IDs mapped to a local `file` and a `reason`. |

Each source needs exactly one of `ref` or `version`, plus `repository` and `groups`. Keep `exclude: {}` and `replace: {}` when you have no exceptions. Unknown fields, duplicate YAML keys, aliases, anchors, and explicit tags are rejected. Local groups come from `local/<group-id>/_group.yaml`; they need no source entry.

## Import every group

Set `groups` to one quoted selector:

| Selector | Groups imported |
| --- | --- |
| `"*"` | All technology and practice groups. |
| `"practices/*"` | All practice groups. |
| `"techs/*"` | All technology groups. |

For example, change the example source's `groups` field to `groups: "*"` to select its whole library. New groups in that scope enter when you sync to a newer revision, so review the resulting diff. Exclusions and replacements still apply.

A wildcard must be the whole `groups` value. You cannot mix it with explicit group paths or use arbitrary patterns such as `techs/**`.

## Repository addresses

Use a complete Git URL or scp-style SSH address, for example:

```text
https://github.com/YOUR-ORG/YOUR-RULES.git
git@git.example.org:engineering/rules.git
```

HTTPS and SSH addresses can include nested repository paths; the `.git` suffix is optional. Git uses your existing credentials. Do not put credentials in configuration. Local paths, `file:`, plain HTTP, and shorthand names such as `owner/name` are unsupported. See [How imports work](/reference/imports/) for Git access and file limits.

## Source names and rule identity

Choose a stable source name such as `acme-rules`. It must start with a lowercase letter and then use lowercase letters, digits, or hyphens. `local` is reserved. Declare a repository only once.

An imported rule ID includes that source name, for example `acme-rules:practices/testing/verify-behavior`. A project-authored rule uses `local:`, such as `local:practices/testing/test-project-contracts`. Renaming a source changes imported rule IDs. Keys under that source's `exclude` and `replace` maps omit the source prefix.

## Commit or tag references

Use `ref` for an exact tag such as `v1.2.3` or a full commit SHA. A plain name resolves as a tag, not a branch; branch names and abbreviated commit SHAs are unsupported. A tag can move, so `sync` may fetch different content for the same tag later. A full commit SHA keeps the selected content fixed. Offline `build` and `check` use the stored commit without contacting Git.

## Semantic version constraints

To select the highest matching release tag during sync, use `version`:

```yaml
version: ">= 1.2.0, < 2.0.0"
```

Constraints use [HashiCorp go-version syntax](https://github.com/hashicorp/go-version). Separate comparisons with commas; `~> 1.2.3` admits releases from 1.2.3 up to, but not including, 1.3.0. npm caret ranges (`^`) and wildcard versions (`1.2.x`) are unsupported. Only complete release tags such as `1.2.3` or `v1.2.3` participate. A prerelease needs a constraint that explicitly admits it.

If no tag matches, sync fails instead of selecting another branch or release. Sync resolves the range again each time; offline build and check use the stored tag and commit. To keep one immutable revision, use a full commit SHA in `ref`. The CLI's `project add library --ref` flag also accepts a range and writes it to `version` in this file.

## Source-scoped exceptions

Put exceptions under the source that owns each imported rule. These fields are partial additions to an existing source entry; keep its repository, revision, groups, and other exceptions:

```yaml
sources:
  acme-rules:
    exclude:
      practices/testing/avoid-snapshots: This project uses reviewed contract snapshots.
    replace:
      techs/typescript/prefer-type-aliases:
        file: local/techs/typescript/prefer-interfaces.md
        reason: Our public extension API relies on declaration merging.
```

The IDs are illustrative and must exist in the selected groups. The `file` path starts at `.code-rules/` and must stay under `local/`. An exclusion removes only that source's rule. A replacement uses the complete local rule once in the active guidance. Keep reasons in configuration; generated provenance records the decisions. For the full workflow, follow [Customize imported rules](/guides/customize/).

## Group selection

Select complete groups for each source. The same group from two libraries combines their rules; source order does not establish priority. A local rule can join an imported group. If no selected library provides the group's metadata, add `local/<group-id>/_group.yaml`.

Local metadata supplies the project's group description and reading cues when it exists. Otherwise, generated guidance labels each contributing library's group metadata by source. Code Rules does not infer an override from matching rule names.

## Conflicting rules

Rules from different sources can require incompatible actions even when their IDs differ. Code Rules does not read their prose to choose a priority. Follow [Resolve conflicting rules](/guides/conflicting-guidance/) to review them and record the intended policy.

For the implementation and tests behind these fields, see [Inspect implementation and tests](/for-agents/#inspect-implementation-and-tests).
