---
title: "Customize imported rules"
description: "Add project guidance or make an explicit exception to an imported rule."
slug: guides/customize
---

An imported library gives your project shared guidance, but a project may need a more specific rule or an exception. Keep those decisions in `.code-rules/local/` and `.code-rules/config.yaml` so the generated rules show what your agents should follow.

## Prerequisites

Before you begin:

- [Set up your project](/start-here/set-up-project/) if you haven't already.
- [Import rules from a library](/guides/select-rules/) so you have rules to customize.
- Run project commands from anywhere in your Git repository. Outside Git, run them from the project root. All file paths below are relative to the project root.

## Choose how to customize

The examples use a testing group (`practices/testing`) imported from the fictional `acme-rules` library. Use your own imported library's name, group, and rule paths when following along.

Suppose an imported rule says "Don't use snapshot tests," but your project requires them to catch unexpected changes to API responses. Those instructions conflict: your agent cannot follow both.

You have two options:

- **Exclude the rule:** Leave it out of the generated guidance without putting another rule in its place.
- **Replace the rule:** Leave out the imported rule and give agents a local rule instead. For example: "Use snapshot tests only for API responses, and review every snapshot change."

Replacing has a similar result to excluding a rule and adding your own, but it also records the connection: which rule you replaced and why. Agents read your local version; the project's provenance retains that connection.

If you only want to add a requirement that doesn't conflict with the imported rules, **add a local rule**. Adding a rule doesn't override an imported one.

## Exclude an imported rule

Choose this option if the imported snapshot ban does not apply to your project and you do not need a replacement rule. In `.code-rules/config.yaml`, add the rule's library-relative ID and your reason under the source that provides it:

```yaml
sources:
  acme-rules:
    exclude:
      practices/testing/avoid-snapshot-tests: This project requires reviewed API response snapshots.
```

This snippet shows only the field to add. Merge it into the existing `sources.acme-rules` entry, keeping its `repository`, `ref` or `version`, `groups`, and any other exceptions. Use the rule's path within the library, without `.md` or the `acme-rules:` source prefix. The exclusion affects only this source; a rule with the same path from another source stays active.

## Replace an imported rule

Choose this option to give agents your own snapshot rule instead of the imported ban. First, create a draft in the same `practices/testing` group:

```sh
code-rules project add rule practices/testing/review-api-snapshots \
  --title 'Review API snapshots' \
  --when-to-read 'When changing public API responses or their tests.' \
  --impact MEDIUM \
  --impact-description 'Catches unexpected changes to the public API response shape.' \
  --non-interactive
```

Open `.code-rules/local/practices/testing/review-api-snapshots.md` and replace the entire draft, including its `<!-- code-rules:draft -->` marker, with a complete rule. For example:

````md
---
title: Review API snapshots
whenToRead: When changing public API responses or their tests.
impact: MEDIUM
impactDescription: Catches unexpected changes to the public API response shape.
---

# Review API snapshots

Use snapshot tests only for public API response shapes. Review each snapshot change against the intended API contract before accepting it. Test other behavior with direct assertions.

Check that a deliberate response-shape change updates the snapshot and receives review, while an unintended change fails the test.
````

See [Write a rule](/guides/write-rules/) for more on its format. A replacement supplies its **complete** local definition, so include every instruction agents still need.

Next, point the imported rule at that file in `.code-rules/config.yaml`:

```yaml
sources:
  acme-rules:
    replace:
      practices/testing/avoid-snapshot-tests:
        file: local/practices/testing/review-api-snapshots.md
        reason: This project requires reviewed API snapshots; other behavior needs direct assertions.
```

Merge this partial snippet into the existing `sources.acme-rules` entry and keep its other fields. If you previously excluded this rule, remove that exclusion. A rule cannot be excluded and replaced at the same time.

The key identifies the imported rule; `file` points to your replacement, relative to `.code-rules/`. Agents read your local version, and [provenance](/reference/provenance/) records what it replaced and why. See [Configuration](/reference/configuration/) for the full field requirements.

## Add a local rule

Choose this option when your new requirement can coexist with the imported rules. Follow [Write a rule](/guides/write-rules/) to add and complete a local rule. For example, if the imported testing rules say nothing about testing failed API requests, you could add a rule requiring those checks.

The local rule joins the selected group; it does not override any imported rule. If the two instructions conflict, choose an exclusion or replacement instead.

## Generate and review updated rules

After following the option that fits your project, write the updated rules and reading indexes to `.code-rules/generated/`, then check the result:

```sh
code-rules project build
code-rules project check
```

Open `.code-rules/generated/RULES.md` and follow the link to the affected group. Confirm that an excluded rule is absent, a replacement appears instead of the original, or an added rule appears alongside the imported rules.

Review the diff in `.code-rules/config.yaml`, `local/`, and `generated/`, then commit the changed files together.

If you also changed a source revision or selected groups, run `code-rules project sync` and `code-rules project check` instead. Sync fetches the library and builds the guidance; review and commit changed `vendor/` snapshots too. Code Rules does not choose between conflicting rules by source order or similar wording. [Resolve any remaining conflict explicitly](/guides/conflicting-guidance/).
