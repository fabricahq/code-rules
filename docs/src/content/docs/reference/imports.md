---
title: "How imports work"
description: "How selected library rules become project guidance and what to review when they change."
---

An **import** brings rules from a library into your project. You choose the library revision and groups in `.code-rules/config.yaml`. Code Rules copies those files into `.code-rules/vendor/`, combines them with your local rules, and writes the guidance agents read under `.code-rules/generated/`.

For the steps, follow [Import rules](/guides/select-rules/). For the directory layout, see [Project files](/reference/files/).

## From library rules to project guidance

Run `code-rules project sync` after adding a library or changing its revision or selected groups. Sync fetches the selected files and builds the generated guidance. Once sync stores the library files, `code-rules project build` can regenerate guidance without network access.

Your project's three rule directories have different owners:

| Directory | What belongs there |
| --- | --- |
| `local/` | Rules and replacements you author for this project. |
| `vendor/` | Original library files copied from a specific Git commit. |
| `generated/` | Active rules and indexes for agents to read. |

Edit `local/` or configuration to change project policy. Sync replaces `vendor/`, and build or sync replaces `generated/`.

## What gets copied

Each library has a source name in configuration, such as `acme-rules`. Its copy lives under `vendor/acme-rules/` and includes:

- Every rule in the selected groups, including rules you later exclude or replace.
- The selected groups' metadata and each rule's own assets.
- Shared assets when a selected rule or its Markdown assets link to them.
- The library manifest and its declared license and notice files.

The copy contains no Git history. Keeping original rules lets you review upstream changes even when your project uses a replacement. For asset layout and link rules, see [Supporting assets](/reference/rule-library-format/#supporting-assets).

## How Code Rules selects the rules your agents read

Configuration selects groups by name or with `"*"`, `"practices/*"`, or `"techs/*"`. The selected revision determines which groups exist. Code Rules reports a missing group or invalid rule instead of silently skipping it. For the selectors and configuration fields, see [Configuration](/reference/configuration/).

From those groups, Code Rules removes rules you explicitly excluded, substitutes local rules for imported rules you replaced, and adds your other local rules. The result is the set of **active rules** in generated guidance. A replacement supplies its entire local definition and appears once. It must be in the same group as the imported rule.

Rule IDs include their source, such as `acme-rules:practices/testing/check-retries`. Two libraries can contribute rules with the same path. Neither source takes priority because of its position in configuration. Code Rules does not detect contradictions in rule text; [resolve conflicting rules](/guides/conflicting-guidance/) explicitly.

## Where agents read the result

Agents start at `generated/RULES.md`, open relevant group indexes, and read applicable rules in full. Generated files also include library summaries, retained license and notice files, and [provenance](/reference/provenance/) showing rule origins and replacements.

Importing guidance does not check whether application code follows it. Your project decides how agents use and enforce its rules.

## What changes when you update

A later sync fetches the revision allowed by your configuration. A full commit stays fixed. A tag can move, and a version range can select a newer matching tag. Sync reports changed files; review the diff before committing.

### Tracing rules to their source

Code Rules records the revision you requested and the exact commit it imported. For a version range, it also records the selected tag. On GitHub.com and GitLab.com, generated links to original rules use the imported commit. For other hosts, links use the stored copy. See [Provenance](/reference/provenance/) to trace a rule or inspect a replacement.

## Checks before updating your files

Sync validates all selected libraries before replacing your stored imports or generated guidance. A failed fetch or invalid library leaves the previous complete set in place. Selected rules must be valid even when you exclude or replace them.

Code Rules rejects missing exception targets, a rule both excluded and replaced, reused replacement files, invalid metadata, unsafe paths, and symbolic links. For a failed or interrupted update, follow [Sync and recovery](/reference/sync/) rather than editing managed files.

## Git access and supported files

Imports require Git 2.30 or later on macOS or Linux. Code Rules uses your Git credentials and certificate and host-key settings. It accepts explicit HTTPS and SSH repository addresses; [Repository addresses](/reference/configuration/#repository-addresses) lists the forms. Credentials are not saved in configuration or provenance.

Code Rules does not execute library code. Selected symbolic links, submodules, and Git LFS pointers are unsupported.

### Supporting files

A rule can link to its own `assets/<rule-name>/` directory or to the library-root `assets/` directory. Missing files, links into another rule's private assets, and links to another rule document fail import. Put shared explanations in the root asset directory.

Within a selected group, Markdown files outside asset directories count as rules, including files in nested folders. Declared terms and the group-root `README.md` are exceptions. Put other supporting Markdown in an asset directory. Markdown assets must be UTF-8; binary images and other assets are allowed.

## Import limits and version errors

Imports enforce time, file-count, and retained-file size limits. If you hit a limit, use the error to identify what needs reducing. These limits do not cap Git's network traffic or temporary disk use. For exact limits, inspect the imports package through the [implementation map](/for-agents/#inspect-implementation-and-tests).

| Error | What to check |
| --- | --- |
| `version-not-found` | No eligible tag matches the version range. Check published tags and the range. |
| `ambiguous-version` | Conflicting tags represent the highest matching version. Correct the tags or select an exact revision. |
| `ref-changed` | A tag moved while sync was fetching it. Retry or select an exact commit. |

For the code paths and tests behind imports, [inspect the implementation](/for-agents/#inspect-implementation-and-tests).
