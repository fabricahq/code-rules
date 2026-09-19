# Resolution and rendering

`internal/build` owns pure resolution and generation through `Generate`. Its inputs are validated configuration, selected library catalogs with original retained bytes, local files, and output options. It returns one complete generated file map or an error with no partial output. It does not fetch, read, or install files, and leaves its inputs unchanged.

Generation applies exclusions, complete local replacements, and local additions. A local group metadata record takes precedence as a whole; all source metadata remains available for provenance. Rule identity and origin travel together.

The generated files include individual resolved rules, discovery indexes, retained terms, library summaries, and provenance. Full-rule group pages must fit the inline byte limit and page line limit; otherwise summaries are used. Summary pagination preserves complete entries and navigation. A zero index line limit selects the default of 750 lines; a nil inline byte limit selects 8 KiB, while zero forces summaries.

Resolution, rendering, pagination, and their intermediate representations are private. Tests exercise the complete operation through its public interface and cover detailed algorithms inside the package.

The [format reference](../docs/src/content/docs/reference/files.mdx) owns externally visible paths and metadata. Run the [repository validation commands](../README.md#validate-changes).
