---
title: "Configuration"
description: "Fields in the proposed .code-rules/config.json format."
---

`.code-rules/config.json` records the project's sources, selected groups, and exceptions.
A project can import rules directly from multiple canonical libraries, pinning each one independently.
The offline builder validates the fields below. The format remains unreleased, and CLI fetching and installation are separate work.

## Complete example

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
      "replace": {
        "techs/typescript/prefer-type-aliases": {
          "file": "local/techs/typescript/prefer-interfaces.md",
          "reason": "Our public extension API relies on declaration merging."
        }
      }
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

Repository names, groups, and rule IDs in examples are illustrative.
Replace them with libraries and rules your project can access.

## Fields

| Field | Meaning |
| --- | --- |
| `schemaVersion` | Configuration format version; the builder accepts `1`. |
| `sources` | Map of stable source names to library configurations. Use an empty object for a project with only local groups. |
| `sources.<name>.repository` | Explicit HTTPS or SSH Git address, including scp-style SSH. See [Repository addresses](#repository-addresses). |
| `sources.<name>.ref` | Full Git commit SHA or exact tag name, such as `v1.0.0`. Mutually exclusive with `version`. |
| `sources.<name>.version` | npm semantic version constraint, such as `^1.2.0`. Mutually exclusive with `ref`. |
| `sources.<name>.groups` | Required group selection: an array of IDs such as `techs/typescript`, or `"*"`, `"practices/*"`, or `"techs/*"` to select all groups in that scope. |
| `sources.<name>.exclude` | Map of this library's rule IDs to exclusion reasons. |
| `sources.<name>.replace` | Map of this library's rule IDs to a local `file` and a `reason`. |

Unknown configuration fields are rejected, including unknown source and replacement fields.
Local groups are discovered from `local/<group-id>/_group.json`; no source entry or separate group list is required.
The former `localGroups` field is rejected with migration guidance. Remove it and keep the group metadata files.
Each source includes its own `exclude` and `replace` objects, empty when unused.
Replacement paths resolve relative to the configuration directory and must stay under its `local/` directory.

Each source specifies exactly one of `ref` or `version` and owns its groups and exceptions.
The earlier singular `source` and top-level `groups`, `exclude`, and `replace` fields are no longer part of the proposed format.
The schema version remains `1` because no configuration format has shipped.

## Import every group

Set `groups` to the string `"*"` to adopt the whole library:

```json
{
  "schemaVersion": 1,
  "sources": {
    "team": {
      "repository": "https://github.com/my-team/rules.git",
      "ref": "v1.0.0",
      "groups": "*",
      "exclude": {},
      "replace": {}
    }
  }
}
```

Choose one of three supported selectors:

| Selector | Included groups |
| --- | --- |
| `"*"` | Every technology and practice group. |
| `"practices/*"` | Every practice group. |
| `"techs/*"` | Every technology group. |

Each selector includes all groups in its scope at the selected revision, including empty groups with valid metadata.
Rule exclusions and replacements still apply. Agents still select relevant rules for each task.
When you adopt a newer revision, newly added groups join the selection; the planned update report must identify those additions.
Offline builds do not discover changes on the remote repository.

Keep `groups` required. Use one supported selector string or an explicit array of group IDs. Wildcard arrays, mixed selectors, and arbitrary globs such as `techs/**` are unsupported.
Local metadata can describe a group that is also selected from a library, including through a wildcard.

Snapshots record both the original `groupSelection` and the concrete `groups` list.
A pattern snapshot must record the exact selector in `groupSelection` and contain every group within that scope at its resolved commit.
For example, `"practices/*"` requires all practice groups, while `"*"` requires both kinds.
The builder compares discovered groups with the recorded expansion and validates every group and rule within the selected scope.
An old partial snapshot is insufficient even if its recorded groups look complete; changing selection intent requires sync.
Legacy snapshots without `groupSelection` represent their explicit `groups` list and remain valid for list-based configuration.
This completeness declaration comes from the snapshot supplier; offline checks do not independently authenticate it against the remote repository.

## Repository addresses

Use a complete Git address so the host is explicit:

```json
"repository": "https://gitlab.com/my-team/engineering/rules.git"
```

Supported forms include:

```text
https://github.com/my-team/rules.git
https://gitlab.com/my-team/engineering/rules.git
ssh://git@git.example.org:2222/srv/rules.git
git@git.example.org:engineering/rules.git
git@git.example.org:/srv/rules.git
```

Nested GitLab namespaces, private hosts, and explicit ports are supported.
The `.git` suffix is optional. Git uses the caller's credentials; do not embed HTTPS credentials or SSH passwords in configuration.
SSH usernames are allowed.

Keep exact revisions in `ref`, version constraints in `version`, and selected groups in `groups`.
Repository addresses do not accept query strings, fragments, `git::` prefixes, getter options, or `//subdirectory` selection.
Local paths, `file:`, unauthenticated `git:`, plain HTTP, and remote-helper protocols are outside this format.
The earlier `owner/name` shorthand is no longer accepted; use `https://github.com/owner/name.git` instead.

Code Rules preserves the supplied address in provenance and requires the snapshot to match it exactly.
Duplicate detection normalizes host spelling and recognizes standard GitHub.com and GitLab.com transport aliases and optional `.git` suffixes.
GitHub path matching ignores case; generic repository paths and GitLab paths retain case.
Other hosts retain their transport, username, port, and path distinctions.
For example, `git@host:rules.git` is home-relative, while `ssh://git@host/rules.git` is absolute. Code Rules does not equate them.

Validation rejects ambiguous paths, including dot segments, encoded separators in URLs, malformed URI escapes, raw whitespace, and backslashes.
SCP paths are literal Git paths, so percent signs in that form are not URI escapes.
Syntax validation does not fetch or authenticate a repository.

Generated source links use pinned GitHub.com or GitLab.com URLs for recognized standard endpoints.
For other hosts, they point to the retained source file under `vendor/<source>/`.
If a relative document or image is missing from that snapshot, generation fails with instructions to retain it or supply an explicit URL.
Code Rules does not infer a host's web interface from its name. See [How imports work](/reference/imports/).

## Source names and rule identity

Choose stable source names such as `fabrica` or `acme`.
Names must match `[a-z][a-z0-9-]*`; `local` is reserved for project-authored rules.
Declare each repository once, with its own revision selection and selected groups.

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
Branch names and abbreviated commit SHAs are unsupported. Put version ranges in `version`, not `ref`.
A plain name resolves only as a tag, even when a branch has the same name.
Both lightweight and annotated tags must resolve to a commit.

In the `sync` workflow, resolve each configured ref and record its full `resolvedCommit` in that source's vendored provenance.
Fetch rule content and construct source links using the resolved commit.
Configuration records what the project requested; provenance records the exact content it imported.

Tags can move.
An explicit `sync` resolves tags again and reports any change from the previously recorded commit for review.
Offline `build`, `check`, and ordinary agent work use the committed snapshot without resolving tags again.
Use a commit SHA when the configured reference itself must be immutable.

## Semantic version constraints

Use `version` instead of `ref` to select the highest matching semantic version tag:

```json
{
  "repository": "https://github.com/example/rules.git",
  "version": "^1.2.0",
  "groups": "*",
  "exclude": {},
  "replace": {}
}
```

Ranges use [npm semver syntax](https://github.com/npm/node-semver#ranges), not HashiCorp's constraint grammar.
For stable releases:

| Constraint | Eligible versions |
| --- | --- |
| `^1.2.3` | At least 1.2.3, below 2.0.0 |
| `~1.2.3` | At least 1.2.3, below 1.3.0 |
| `>=1.2.3 <2.0.0` | Explicit lower and upper bounds |
| `1.2.x` | Any patch release within 1.2 |
| `1.2.3` | Exactly that semantic version |

Prereleases follow npm's default rules: a range must explicitly admit a prerelease for the same major/minor/patch tuple.
For example, `>=2.0.0-beta.1 <2.0.0` admits later betas of 2.0.0; `^1.2.0` does not admit 2.0.0 betas.
Caret ranges below 1.0 have narrower compatibility bounds; `^0.2.0` stays below 0.3.0.

Only complete version tags such as `1.2.3` or `v1.2.3` participate. The optional prefix is lowercase `v`.
Partial tags such as `v1`, names such as `release-1.2.3`, and branches are ignored during version selection.
Lightweight and annotated tags are supported, but the selected tag must resolve to a commit.
Use `ref` to select an exact tag outside this naming convention.

Imports selects by semantic version precedence, not tag date or Git listing order.
If tags at the highest matching precedence point to different objects, import fails as ambiguous. Build metadata does not affect precedence.
Aliases pointing to the same commit are allowed; the lexicographically first tag spelling is selected deterministically.
If the selected tag changes between discovery and fetching, import fails rather than silently accepting a different revision.
No matching tag is an error; Imports does not fall back to a branch or unrelated release.

Snapshots and generated provenance record the requested `version`, `resolvedTag`, `resolvedVersion`, and `resolvedCommit`.
The normalized version retains any SemVer build metadata and omits the leading `v`.
A new explicit import resolves the constraint again. Offline Builds checks the recorded tag and version against the constraint, then uses the stored commit without querying Git.
The development `sync` command combines re-importing and safe file updates. See [Sync and recovery](/reference/sync/).

## Source-scoped exceptions

Keys inside a source's `exclude` and `replace` use the library-relative rule ID, without a source prefix.
For example, `sources.fabrica.exclude["practices/testing/verify-retry-limits"]` affects only Fabrica's rule.
An identically named rule from `acme` remains active.
Full paths distinguish matching filenames in different groups of the same library.

Generated files and review findings retain source-qualified IDs because they appear outside the configuration's source nesting.

## Group selection

Select complete groups separately for each source, then exclude individual rules when necessary.
Each selected group must exist in that source at its pinned revision.
Selecting `practices/testing` from two sources combines both sets of rules into one effective testing group, with a group page and individual resolved rule files.
Source order never establishes precedence.

Every valid rule file under `local/` automatically joins its adopted group; individual local rules do not need source entries.
Local rules can join any imported group. A matching filename does not override an imported rule.
A local file referenced by `replace` appears once under its local ID; the target is removed from active output.
The complete local definition supplies the metadata, guidance, attribution, and assets. Configuration and provenance retain the replacement relationship.
A local `_group.json` defines a group, including an empty group, without any configuration entry.
If local metadata exists for an imported group, its complete description and reading cues take precedence for project discovery.
Otherwise, descriptions from every contributing library remain source-labeled.
Provenance retains all group metadata and identifies the sources supplying the effective discovery guidance.
Local rules without either local or imported group metadata are errors, with the missing `_group.json` path in the diagnostic.
The root `local/README.md` is directory documentation, not a rule; other misplaced Markdown files are still validated.

The generated index shows group names, applicability guidance, and explicit **Open group** links. Without local metadata, guidance from multiple libraries remains labeled by source.
Local metadata supplies the complete project description when present.
The importer does not silently choose one library's description over another's.

## Conflicting rules

Source-qualified IDs prevent naming collisions, but distinct rules can still require incompatible behavior in the same situation.
Both remain active unless the project explicitly excludes or replaces a rule under its owning source.
The importer does not infer priority from source order or interpret prose to resolve contradictions.

See [Conflicting guidance](/guides/conflicting-guidance/) for examples, an agent review prompt, and ways to resolve competing instructions.

## Version boundaries

`schemaVersion` describes this configuration.
Each library's `formatVersion` describes its authoring format.
The caller-supplied `toolVersion` identifies the tool that generated the output.

Each source's `ref` or `version` selects the requested revision.
Vendored provenance records the resolved commit used by offline commands; it is importer-owned output, not a second user-selected version.
Direct imports from multiple sources are part of the first-release design.
Libraries that themselves inherit and republish other libraries remain later work.

See [Files and formats](/reference/files/) for library metadata and [Adapt rules](/guides/select-rules/#adapt-the-import-to-your-project) for examples of exceptions.
