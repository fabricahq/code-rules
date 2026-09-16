# Effective rule resolution

`internal/build.Resolve` combines a parsed configuration, validated selected library catalogs, and local file bytes. It returns ID-sorted groups with effective definitions, origins, replacement provenance, and supporting inventories. Active rule text remains owned by each `rules.Rule.Document`. Source retained files keep excluded and replaced upstream documents so later vendoring and update comparisons can recover every original. Active documents are not duplicated there.

Catalogs retain the selector used by `library.Load`; it must match configuration, including wildcard spelling. Version-selected sources require a chosen tag satisfying their constraint, and retain that tag and normalized version for provenance. All candidate rules validate before exclusions. Exception targets must exist. A local replacement can replace one upstream definition and retains its upstream origin and reason. Additional local rules require group metadata, supplied either locally or by an imported group. `Resolve` chooses `effectiveGuidance` once for every group: the complete local metadata definition wins when present. Without local metadata, all imported definitions remain effective in source-name order. Original definitions remain in `guidance` for provenance. Renderers consume the resolved choice and do not repeat this policy.

Inputs are not modified. Supporting byte slices are shared read-only. Catalogs must come from `library.Load`, and configuration must come from `rules.ParseConfiguration`. This boundary does not verify Git authenticity or snapshot freshness. It does not fetch repositories, render output, or write project files.

The `/walkthrough/pr23` page calls the native resolver through disposable library fixtures. Presets cover imported rules, exclusions, replacements, local additions, guidance precedence, missing targets, invalid excluded definitions, and local-only projects.

Local rule documents and retained Markdown attachments use the same link policy as libraries: filesystem links to other rule documents fail before adoption. Self-links, anchors, allowed assets, and external references remain valid.
