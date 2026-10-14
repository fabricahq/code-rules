---
title: "Share defaults with presets"
description: "Publish an organization's default libraries, groups, and skills as a preset, and start each project from it."
---

A **preset** is a starting configuration that a library publishes: the sources, groups, rules, and skills an organization wants every project to begin with, and where to install skills. A project that **extends** the preset gets all of it from one line of configuration, and adopts changes to it with `code-rules project update`.

Presets help when several projects share the same defaults. Without one, each project copies the same sources and selections into its own configuration, and they drift apart as people edit them. With one, the organization changes the defaults once, and each project reviews and adopts the change when it updates.

In this guide, you'll write a preset in a library, publish it, extend it from a project, and see how the project's own configuration combines with it. The first half is for library maintainers; the second, from [Extend a preset](#extend-a-preset), is for projects.

## Write a preset

A preset is a YAML file in the library's `presets/` folder, at the library root. Its file name, without `.yaml`, is the preset's name, which must use only lowercase letters, digits, and hyphens, and can't start or end with a hyphen or contain two in a row. An organization usually keeps its presets in its shared library, such as `<organization>/.code-rules`:

```text
.code-rules/
  rule-library.yaml
  presets/
    fabrica-defaults.yaml
  practices/
    ...
  skills/
    ...
```

The file uses the same shape as a project's configuration, with `schemaVersion`, `skillTargets`, and `sources`:

```yaml
schemaVersion: 2
skillTargets: [claude, agents]
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    groups:
      - practices/testing
      - practices/error-handling
    skills:
      - skills/build-feature
      - skills/fix-bug
  fabrica-org:
    repository: https://github.com/fabricahq/.code-rules.git
    groups:
      - techs/go
    skills:
      - skills/ship-service
    exclude:
      techs/go/prefer-table-tests:
        reason: Our services use one test per behavior.
```

Each source works exactly as it does in a [project's configuration](/reference/configuration/#fields): libraries with `groups`, `rules`, `skills`, `exclude`, `pins`, and `ref`, and [plain repositories](/reference/configuration/#plain-repositories) with `ref`. A preset can name its own library as a source, as `fabrica-org` does here. A preset can't:

- Extend another preset: `extends` is rejected.
- Name local files: an exclusion's `replacedBy` and `basedOn` are rejected, because a preset has no `local/` folder. A project that replaces a rule or skill writes the replacement in its own configuration.

A preset only selects. Each rule and skill still comes from its own source, with its own version, credit, and license, so a preset never republishes another library's rules or skills.

## Check and publish a preset

`code-rules library check` validates each file in `presets/`: its name, its YAML, and each field, as a project's configuration is validated before sync. It doesn't fetch the repositories a preset names, so a misspelled group or skill ID shows up when a project syncs. Test a preset by extending it from a project with `extends.ref` set to your branch's commit before you publish it.

Presets are [library-wide files](/reference/rule-versions/#what-a-version-covers), like group metadata and shared assets: they aren't versioned and need no change notes. The next library release publishes them:

```sh
code-rules library release
```

Projects that extend the preset see the change, in their update preview, the next time they run `code-rules project update`. Removing a selection from a preset removes those rules and skills from projects that adopt the update, unless a project selects them itself, so say why in the pull request that changes the preset.

## Extend a preset

In a project, set `extends` in `.code-rules/config.yaml` to the library and preset name:

```yaml
schemaVersion: 2
extends:
  repository: https://github.com/fabricahq/.code-rules.git
  preset: fabrica-defaults
sources: {}
```

`extends` requires `schemaVersion: 2`. Then sync:

```sh
code-rules project sync
```

Sync reads the preset from the library's newest library release, records that library release in `.code-rules/vendor/_preset/_source.json`, stores the preset file beside it, and imports every source the preset declares, as if you'd written them yourself. Later syncs keep the recorded library release, so the preset never changes underneath you. Offline commands, such as `code-rules project build` and `code-rules project check`, use the stored copy.

To keep the preset at one revision, add `ref` with a library release tag, such as `release/7`, or a full commit SHA, as for a source's [`ref`](/reference/configuration/#import-one-revision).

## Combine your configuration with the preset

The project's configuration adds to the preset's:

- **New sources.** A source with a name the preset doesn't use is added beside the preset's sources.
- **The same source.** When you declare a source with the same name as one in the preset, its `repository` must match the preset's. Your `groups`, `rules`, and `skills` are added to the preset's selections.
- **Exceptions and versions.** Your `exclude`, `pins`, and `skillUses` entries win over the preset's for the same rule or skill. Your `ref` and `skillsRoot` win when you set them.
- **Install targets.** Your `skillTargets` wins when you set it, including to `[]` for none.

For example, this project adds a group and a skill to the preset's `fabrica` source, replaces one of its skills with a local fork, and pins another:

```yaml
schemaVersion: 2
extends:
  repository: https://github.com/fabricahq/.code-rules.git
  preset: fabrica-defaults
sources:
  fabrica:
    repository: https://github.com/fabricahq/public-rules.git
    groups:
      - practices/observability
    skills:
      - skills/review-change
    exclude:
      skills/fix-bug:
        reason: We use our own bug workflow.
        replacedBy: local/skills/fix-bug
        basedOn: "1.0.0"
    pins:
      skills/build-feature:
        version: "1.2.0"
        reason: Waiting on feedback about the new review step.
```

You can add to the preset and make exceptions to it, but you can't remove its selections or its exclusions. To diverge that far, stop extending it: copy the sources you want to keep into your configuration, remove `extends`, and run `code-rules project sync`. Removing `extends` without copying them removes everything the preset selected.

`generated/provenance.json` records the preset, and marks each source the preset declares with `fromPreset`; see [Provenance](/reference/provenance/#inspect-library-versions-and-group-guidance).

## Update to a newer preset

A full `code-rules project update` moves the preset to its library's newest library release, along with everything else. The preview starts with what the preset added or removed, then lists the rules and skills those changes bring in or drop under their sources:

```text
preset fabrica-defaults: release 6 -> 7
  added     fabrica  skills  skills/review-change
  removed   fabrica  groups  practices/error-handling
  added     acme     source  https://github.com/acme/.code-rules.git
```

Review it like any update, then confirm. Updating one source or one rule doesn't move the preset, and a preset with `ref` stays put. See [Update a preset](/guides/update/#update-a-preset).

## One level only

A project extends at most one preset, and a preset can't extend another. To build on an organization's preset for one team, add the team's sources and selections in each project's configuration, or publish a separate team preset that names the same sources.

## Private presets

A preset in a private library works like any private library: Code Rules reads it with your Git credentials, and every source it names must be readable with them too. Everyone who syncs the project, including CI, needs access to the preset's library and every source it names.
