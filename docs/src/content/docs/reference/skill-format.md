---
title: "Skill format"
description: "What Code Rules accepts in a skill folder and its SKILL.md: IDs, frontmatter, metadata keys, links, files, and every library check."
---

This page defines what Code Rules accepts in a [skill](/concepts/skill/): its folder, its `SKILL.md` file, and the Code Rules keys in its metadata. The same format applies to skills in a library and to a project's local skills. For how to write a skill well, see [Write a skill](/guides/write-skills/).

A skill uses the [Agent Skills](https://agentskills.io) format. Code Rules adds three optional `metadata` keys and checks the rest of the format as the Agent Skills specification defines it, so a skill that passes `code-rules library check` also works in tools that read Agent Skills directly.

## Skill folders

A library's skills live in its root `skills/` directory, one folder per skill, beside its rule groups. [Library format](/reference/library-format/) shows the whole layout.

```text
skills/
  build-feature/
    SKILL.md
    references/
      slicing.md
    scripts/
      verify.sh
  review-tests/
    SKILL.md
```

Each skill folder holds a `SKILL.md` file and anything the skill bundles. The Agent Skills format suggests `scripts/` for code the agent runs, `references/` for documents it reads when needed, and `assets/` for templates and other files it uses, but any layout inside the folder works. The `assets/` folder inside a skill belongs to the skill; it isn't a rule asset directory, and the library-root `assets/` directory isn't part of any skill.

`skills/` holds only skill folders, and is flat: a skill folder can't contain another skill's `SKILL.md`, and it can't contain `_group.yaml`. Skills don't belong to groups, so a skill folder is never inside `techs/` or `practices/`.

A project's local skills use the same format in `.code-rules/local/skills/<name>/`. See [Local skills](#local-skills).

### Git-tracked files

A skill's content is the files Git tracks in its folder. Files Git ignores, such as `__pycache__/`, `node_modules/`, or `.venv/` that a skill's scripts create, are neither versioned nor installed. `code-rules library check` warns about a file in a skill folder that Git neither tracks nor ignores, because it isn't part of the skill until you commit it. A library outside a Git repository has no library releases, and check reads every file in each skill folder.

Symbolic links, submodules, and Git LFS pointers aren't supported in a skill folder, as for every file Code Rules imports.

### File modes

A file's executable bit is part of the skill's content. Git records it as mode `100755` for an executable file and `100644` for any other file. Code Rules keeps the mode in the library's vendored copy in a project and in every installed copy, so a step that runs `./scripts/verify.sh` works after install. Changing only a file's mode is a change to the skill, and after the first library release it needs a change note, like any other change.

### File contents

Markdown files in a skill folder, including `SKILL.md`, must be UTF-8, so their links can be checked. Other files can be binary; Code Rules keeps their original bytes. The [import limits](/reference/imports/#import-limits-and-version-errors) on file size and count apply to skill files as to any other imported file.

## Skill IDs

A skill's ID is its path in the library without the trailing `/SKILL.md`: `skills/<name>`, such as `skills/build-feature`. In a project, the ID is qualified by the source name the project chose, such as `fabrica:skills/build-feature`, and a local skill's ID is `local:skills/<name>`.

`<name>` follows the Agent Skills name grammar:

- 1 to 64 characters,
- only lowercase letters `a`–`z`, digits `0`–`9`, and hyphens `-`,
- not starting or ending with a hyphen, and
- no two hyphens in a row.

The folder name is the skill's name, and it must equal the `name` field in its `SKILL.md`. Renaming the folder changes the skill's ID. In a library, a rename retires the old ID, replaced by the new one; see [Retire or replace a skill](/guides/write-skills/#retire-or-replace-a-skill). A retired skill's ID can't be reused.

The name is also the folder name of every installed copy, such as `.claude/skills/build-feature/`, so two skills with the same name can't be installed in one project, even from different sources. See [Project files](/reference/files/).

## Frontmatter

`SKILL.md` starts with YAML frontmatter, from the first byte of the file, followed by a non-blank Markdown body with the skill's instructions:

```markdown
---
name: build-feature
description: Build a feature from an agreed spec, one slice at a time. Use when starting implementation of a new feature or a change to an existing one.
license: MIT
metadata:
  fg-skill-type: playbook
  fg-rules-used: |
    practices/architecture/decide-module-boundaries-first
    practices/testing/test-behavior-not-implementation
  fg-skills-used: |
    skills/tdd
---

1. Read the spec and name the slice you're building.
2. ...
```

These are the Agent Skills fields. Unknown top-level fields are rejected, because tools that read Agent Skills, such as claude.ai, reject them too.

| Field | Meaning |
| --- | --- |
| `name` | Required. The skill's name, following the [name grammar](#skill-ids). Must equal the folder name. |
| `description` | Required. One non-blank string of at most 1,024 characters saying what the skill does and when to use it. Agents read it to decide when to load the skill, so name the task that should trigger it. |
| `license` | Optional string naming the skill's license, such as an SPDX expression. Informational in a library; see [Licenses](#licenses). |
| `compatibility` | Optional non-blank string of at most 500 characters describing what the skill needs to run, such as tools, network access, or a product. |
| `allowed-tools` | Optional string listing tools the skill is approved to use, separated by spaces, as the Agent Skills format defines it. Code Rules preserves it and doesn't interpret it. |
| `metadata` | Optional map of string keys to string values. Code Rules reads the [three keys below](#metadata) and preserves every other key unchanged. |

The frontmatter uses one YAML document. Duplicate keys, anchors, aliases, and explicit tags are rejected. A value that YAML reads as something other than a string, such as `true` or `2026-10-14`, must be quoted where a string is required.

A skill has no version field, and no metadata key sets its version. In a library, its version comes from library releases; see [How a skill is versioned](/concepts/skill/#how-a-skill-is-versioned).

## Metadata

Code Rules adds three optional keys under `metadata`. Each value is a string, as the Agent Skills format requires, so the file stays valid everywhere skills are read.

| Key | Value | Checks |
| --- | --- | --- |
| `fg-skill-type` | `playbook` or `review` | Must be a type on the canonical list. Unknown values fail `code-rules library check`. |
| `fg-rules-used` | The IDs of the rules the skill applies, one per line | Each must be a current rule in this library. |
| `fg-skills-used` | The IDs of the skills this skill uses, `skills/<name>`, one per line | Each must be a current skill in this library, other than this one, and the skills used can't form a cycle. |

Other `metadata` keys, such as `author`, are preserved and ignored. Keys that start with `fg-` are reserved for Code Rules, so any other `fg-` key fails `code-rules library check`.

### Skill type

`fg-skill-type` labels the skill as a `playbook` or a `review`; see [Skill types](/concepts/skill/#skill-types). Leave it out when neither fits. Code Rules shows the type, and `code-rules project list skills --type` filters by it, but it installs, checks, and updates every skill the same way, whatever its type.

The canonical type list is `canonical-skill-types.yaml` in the Code Rules repository. Each Code Rules release builds the list in, so a new type needs a Code Rules release, and your library's CI must use that release before a skill can declare the type. Only `code-rules library check` checks the type: a project imports a skill whose type its Code Rules doesn't know without complaint, and shows the type as written.

### Rules a skill applies

`fg-rules-used` lists the rules the skill applies, one library rule ID per line, in a YAML block string:

```yaml
metadata:
  fg-rules-used: |
    practices/architecture/decide-module-boundaries-first
    practices/testing/test-behavior-not-implementation
    practices/verification/verify-on-the-surface-users-touch
```

Each line is a rule ID in this library: the rule's path without `.md`, and without a source name, because each project chooses its own source names. Surrounding whitespace and blank lines are ignored. Each ID can appear once. A skill can cite rules from any group in its library, and only rules in its library.

When a project installs the skill, Code Rules imports every cited rule, even from groups the project didn't select, and appends a generated section to the installed `SKILL.md` that links to the project's version of each one. See [Installed copies](#installed-copies).

### Skills a skill uses

`fg-skills-used` lists the other skills this skill uses, one skill ID per line, in the same form:

```yaml
metadata:
  fg-skills-used: |
    skills/architect
    skills/tdd
```

Each line is a skill ID in this library, `skills/<name>`. A skill can't use itself, and the skills used can't form a cycle, directly or through other skills. When a project installs the skill, Code Rules also installs every skill it uses, and the skills those use in turn, from the same source.

## Links

Links in a skill's Markdown files must be absolute URLs, such as `https://agentskills.io`, or relative paths that stay inside the skill's own folder, such as `references/slicing.md` or `scripts/verify.sh`. Installed copies are self-contained, so a skill can't link to a rule file, another skill, the library-root `assets/` directory, or any other file in the library. Cite rules with `fg-rules-used` instead, and the installed copy links to the project's version of each one.

Code Rules checks Markdown links, images, reference definitions, and the `href` and `src` attributes of HTML. Relative links must point to a file that exists in the skill's folder. Links to a heading in the same file, such as `#verify`, are allowed. Code Rules keeps external URLs as links and never downloads them. Files other than Markdown, such as scripts, aren't checked for links.

## Licenses

A library's license, declared in `rule-library.yaml`, covers its skills as it covers its rules. The `license` field in `SKILL.md` is informational: Code Rules preserves it, and when it differs from the library's SPDX expression, `code-rules library check` warns. Skills from a plain repository have no library declaration, so their license comes from the skill or the repository instead. See [License rules](/guides/license-rules/#license-skills).

## Validation

`code-rules library check` validates every skill in the library, whatever a project selects. It reports every skill problem it finds, one per line, each starting with the file or folder it's about, and fails with the code `invalid-skill`:

```text
$ code-rules library check
Error: skills don't pass library check:
  - skills/build-feature/SKILL.md: fg-rules-used names practices/testing/test-behaviour-not-implementation, which isn't a rule in this library
  - skills/build-feature/SKILL.md: fg-skills-used names skills/tdd-loop, which isn't a skill in this library
  - skills/build-feature/SKILL.md: fg-skill-type is design; use playbook or review, or leave it out
```

Check fails with these problems:

| Problem | Message |
| --- | --- |
| A file directly in `skills/` | `skills/README.md: skills/ holds only skill folders` |
| A folder name that breaks the [name grammar](#skill-ids) | `skills/Build_Feature: a skill name uses 1 to 64 lowercase letters, digits, and hyphens, without a leading, trailing, or doubled hyphen` |
| A skill folder without `SKILL.md` | `skills/build-feature: expected SKILL.md in the skill folder` |
| A skill inside another skill's folder | `skills/build-feature/steps/SKILL.md: a skill folder can't contain another skill; move it to its own folder in skills/` |
| Group metadata in a skill folder | `skills/build-feature/_group.yaml: a skill folder can't contain group metadata` |
| A symbolic link, submodule, or Git LFS pointer | `skills/build-feature/scripts/verify.sh: symbolic links, submodules, and Git LFS pointers aren't supported in a skill folder` |
| A Markdown file that isn't UTF-8 | `skills/build-feature/references/slicing.md: expected UTF-8 Markdown` |
| No frontmatter at the start of `SKILL.md` | `skills/build-feature/SKILL.md: expected YAML frontmatter at the start of the file` |
| Invalid YAML, such as a duplicate key or an alias | `skills/build-feature/SKILL.md: ` followed by the YAML problem and its line |
| A blank body | `skills/build-feature/SKILL.md: expected instructions after the frontmatter` |
| An unfinished draft | `skills/build-feature/SKILL.md: complete the draft and remove its code-rules:draft marker` |
| An unknown top-level field | `skills/build-feature/SKILL.md: unknown field version; SKILL.md accepts name, description, license, compatibility, allowed-tools, and metadata` |
| A missing or blank `name` or `description` | `skills/build-feature/SKILL.md: description is required` |
| A `name` that differs from the folder name | `skills/build-feature/SKILL.md: name is build-features, but the folder is build-feature; they must match` |
| A `description` over 1,024 characters | `skills/build-feature/SKILL.md: description is 1,310 characters; use at most 1,024` |
| A `compatibility` over 500 characters, or blank | `skills/build-feature/SKILL.md: compatibility is 612 characters; use at most 500` |
| A field or `metadata` value that isn't a string | `skills/build-feature/SKILL.md: metadata.fg-skill-type must be a string` |
| A reserved `fg-` key Code Rules doesn't define | `skills/build-feature/SKILL.md: metadata.fg-groups isn't a Code Rules key; use fg-skill-type, fg-rules-used, or fg-skills-used` |
| A type that isn't on the canonical list | `skills/build-feature/SKILL.md: fg-skill-type is design; use playbook or review, or leave it out` |
| A cited rule that doesn't exist | `skills/build-feature/SKILL.md: fg-rules-used names practices/testing/test-behaviour-not-implementation, which isn't a rule in this library` |
| A cited rule with a source name or `.md` | `skills/build-feature/SKILL.md: fg-rules-used names fabrica:practices/testing/test-behavior-not-implementation; cite a rule by its ID in this library, such as practices/testing/test-behavior-not-implementation` |
| A cited rule that a library release or a pending change note retired, with a replacement | `skills/build-feature/SKILL.md: fg-rules-used names practices/testing/check-retry-backoff, which is retired and replaced by practices/testing/verify-retries; cite the replacement` |
| A cited rule that was retired without a replacement | `skills/build-feature/SKILL.md: fg-rules-used names practices/code-design/comment-intent, which is retired; remove it` |
| An ID listed twice | `skills/build-feature/SKILL.md: fg-rules-used names practices/testing/verify-retries twice` |
| A used skill that doesn't exist | `skills/build-feature/SKILL.md: fg-skills-used names skills/tdd-loop, which isn't a skill in this library` |
| A used skill that is retired | `skills/build-feature/SKILL.md: fg-skills-used names skills/triage-bug, which is retired and replaced by skills/fix-bug; use the replacement`, or `which is retired; remove it` when nothing replaces it |
| A skill that uses itself | `skills/build-feature/SKILL.md: fg-skills-used names the skill itself` |
| Skills that use each other in a cycle | `skills/build-feature/SKILL.md: fg-skills-used forms a cycle: skills/build-feature -> skills/tdd -> skills/build-feature` |
| A link that leaves the skill folder | `skills/build-feature/SKILL.md: link leaves the skill folder: ../../practices/testing/verify-retries.md; link inside skills/build-feature/ or use an absolute URL` |
| A relative link to a missing file | `skills/build-feature/SKILL.md: link target doesn't exist: references/slices.md` |
| A skill that reuses the ID of a retired skill | `skills/triage-bug reuses the ID of a skill that release/5 retired. Retired IDs can't be reused; give the skill a new ID.` |

After the first library release, check also compares every skill with the latest library release and fails, with the code `change-notes`, when a skill's change notes don't match its changes, exactly as for rules. See [Change notes](/reference/rule-versions/#change-notes) and [library check](/reference/cli/#library-check).

Check warns, without failing, when:

- A skill's `license` differs from the library's declared SPDX expression: `skills/build-feature/SKILL.md: license is Apache-2.0, but rule-library.yaml declares MIT. The library's license applies to its skills; make them match.`
- A skill declares a `license` and the library declares none: `skills/build-feature/SKILL.md: license is MIT, but rule-library.yaml declares no license. Declare the library's terms.`
- A file in a skill folder is neither tracked nor ignored by Git: `skills/build-feature/scripts/new-check.sh isn't tracked by Git, so it isn't part of the skill. Add it with git add, or ignore it.`

Check validates the format, not whether the steps are good. Use [Write a skill](/guides/write-skills/) to review a skill's guidance.

## Installed copies

When a project installs a skill, Code Rules writes a copy of its folder into each of the project's install targets, such as `.claude/skills/build-feature/`. Every file is copied as the source has it, with its executable bit, and `SKILL.md` gets two changes:

- A marker comment becomes the first line after the frontmatter, so the frontmatter still starts the file:

  ```markdown
  <!-- code-rules:installed fabrica:skills/build-feature 1.3.0. Do not edit; fork it with code-rules project add skill --from. -->
  ```

- When the skill cites rules, a generated **Rules this skill applies** section is appended at the end. It links to the project's version of each rule the skill cites, including a local rule that replaces one, and marks a rule the project excluded. A **Skills this skill uses** line follows when the skill uses others.

Don't edit installed copies: sync rewrites them, and `code-rules project check` fails when one differs from what sync would write. [Installed skills](/reference/files/#installed-skills) shows the complete generated section and how Code Rules records the folders it owns.

## Local skills

A project's local skills live in `.code-rules/local/skills/<name>/` and use this format, with two differences:

- They cite rules and skills by project ID, with the source name, because a project can use several sources: `fabrica:practices/testing/test-behavior-not-implementation`, `local:practices/architecture/follow-bounded-contexts`, or `fabrica:skills/tdd`. Each ID must name a configured source, or `local`; Code Rules imports each cited rule, and installs each used skill, from the source it names.
- They have no versions and need no change notes.

`code-rules project add skill` creates one, and `code-rules project add skill --from` forks an imported skill into one; see [project add skill](/reference/cli/#project-add-skill). Sync and build validate local skills with the checks above and fail on the first problem.

## Skills from plain repositories

A plain repository, one without `rule-library.yaml`, can supply skills too. Each folder directly inside its skills root that has a `SKILL.md` file is a skill, with the ID `skills/<folder-name>`. Code Rules checks these skills leniently: where a skill breaks this format, such as a `name` that doesn't match its folder, a missing `description`, or a link outside its folder, sync warns and installs the skill as it is. The checks that protect your project still fail: unsafe paths, symbolic links, submodules, Git LFS pointers, and the import limits. A plain repository's `fg-rules-used` and `fg-skills-used` are ignored; the project declares the skills one uses with `skillUses` instead. See [Plain repositories](/reference/imports/#plain-repositories).

## Related pages

- [Skill](/concepts/skill/) explains what a skill is and how it relates to rules.
- [Write a skill](/guides/write-skills/) walks through writing, checking, and versioning a skill.
- [Library format](/reference/library-format/) covers where skills live in a library.
- [Rule and skill versions](/reference/rule-versions/) defines what a skill version covers.
