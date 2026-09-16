# Effective rule resolution

`internal/build.Resolve` combines a parsed configuration, validated selected library catalogs, and local file bytes. It returns ID-sorted groups with effective definitions, origins, replacement provenance, and supporting inventories. Original rule text remains owned by each `rules.Rule.Document`.

Catalogs retain the selector used by `library.Load`; it must match configuration, including wildcard spelling. Version-selected sources require a chosen tag satisfying their constraint, and retain that tag and normalized version for provenance. All candidate rules validate before exclusions. Exception targets must exist. A local replacement can replace one upstream definition and retains its upstream origin and reason. Additional local rules require group metadata, supplied either locally or by an imported group. Local group guidance takes precedence for display; all source guidance remains available in the result.

Inputs are not modified. Supporting byte slices are shared read-only. Catalogs must come from `library.Load`, and configuration must come from `rules.ParseConfiguration`. This boundary does not verify Git authenticity or snapshot freshness. It does not fetch repositories, render output, or write project files.

The `/walkthrough/pr23` page calls the native resolver through disposable library fixtures. Presets cover imported rules, exclusions, replacements, local additions, guidance precedence, missing targets, invalid excluded definitions, and local-only projects.
