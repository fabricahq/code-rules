# Resolution and rendering

`internal/build` owns pure resolution and generation. Its inputs are validated configuration, selected library catalogs with original retained bytes, and local files. It does not fetch, read, or install files.

`Resolve` applies exclusions, complete local replacements, and local additions. A local group metadata record takes precedence as a whole; all source metadata remains available for provenance. Rule identity and origin travel together.

`Prepare` combines individual resolved rules, discovery indexes, retained terms, library summaries, and provenance into one generated file map. Errors return no partial output. Full-rule group pages must fit the inline byte limit and page line limit; otherwise summaries are used. Summary pagination preserves complete entries and navigation.

The [format reference](../docs/src/content/docs/reference/files.mdx) owns externally visible paths and metadata. Tests beside the Go implementation cover rendering, link relocation, deterministic ordering, pagination, and copied documentation examples. Run the [repository validation commands](../README.md#validate-changes).
