# Resolution and rendering

`internal/build` owns pure resolution and generation through `Generate`. Its inputs are validated configuration, selected library catalogs with original retained bytes, local files, and output options. It returns one complete generated file map or an error with no partial output. It does not fetch, read, or install files, and leaves its inputs unchanged.

Generation applies exclusions, complete local replacements, and local additions. A local group metadata record takes precedence as a whole; all source metadata remains available for provenance. Rule identity and origin travel together.

The generated files include individual resolved rules, discovery indexes, retained terms, library summaries, and provenance. Full-rule group pages must fit the inline byte limit and page line limit; otherwise summaries are used. Summary pagination preserves complete entries and navigation. A zero index line limit selects the default of 750 lines; a nil inline byte limit selects 8 KiB, while zero forces summaries.

Resolution, rendering, pagination, and their intermediate representations are private. Tests exercise the complete operation through its public interface and cover detailed algorithms inside the package.

Public references own [project files](../docs/src/content/docs/reference/files.mdx), [authoring formats](../docs/src/content/docs/reference/rule-library-format.mdx), and [provenance](../docs/src/content/docs/reference/provenance.md). Tests beside the Go implementation cover rendering, link relocation, deterministic ordering, pagination, and copied documentation examples. Run the [repository validation commands](../CONTRIBUTING.md#validate-changes).

## API boundary

Use `code-rules project build` and `code-rules project check` for supported project operations. The repository's Go implementation is under `internal/`; it is not a public Go SDK.

Internally, resolution consumes validated configuration, library catalogs with retained bytes, and local files. Rendering returns a complete map of generated paths to bytes without network or filesystem effects. Project operations verify persisted snapshots and install or compare that output.

Invalid input returns an error with its location. Missing or mismatched vendor snapshots require `code-rules project sync`; the offline builder never silently fetches a replacement. Binary attachments and declared terms retain their original bytes.

The project layer loads `_source.json`, verifies digests, and checks filesystem containment. Pure resolution checks source selections against configuration without reading that record or computing workspace digests.

## Rendering limits

Full-rule group pages have an 8 KiB inline limit and must also fit the 750-line page limit. These are rendering defaults, not CLI flags. The internal renderer accepts `GroupInlineMaxBytes` as an explicit inline limit; zero forces summaries.
The measurement counts UTF-8 bytes, including instructions, metadata, examples, attribution, and the footer.
A single large rule can require a summary index. The line count is measured in Markdown source lines, not visual wrapping.
This is a provisional delivery default, not an empirically validated threshold for agent compliance.

Indexes use deterministic natural ordering for numeric portions of IDs, so rule-2 precedes rule-10. A summary page is split only when it exceeds the 750-source-line default. The internal renderer names this setting `IndexMaxLines`.
An oversized index becomes a directory listing every numbered sibling part, such as `testing.part-1.md`.
Each part contains complete entries, a Page X of Y label, and direct previous/next links where applicable, plus an all-pages directory link. Inspect all parts before selecting rules.
The root group index uses the same mechanism when necessary.
If one entry or the complete part directory cannot fit, building fails with an actionable error instead of omitting entries.
Full rule bodies are never shortened to fit an index budget. Complete any truncated reads of those files.
This budget is a delivery setting, not a measured model-compliance threshold.

## Rendering links and headings

Inline rules and standalone rule files come from the same resolved definition. Relative links are relocated from the actual output directory.
Links to declared license and notice files target their generated copies. Other retained files use workspace-relative links. Code Rules rejects filesystem references to other rule documents instead of generating fallback repository links. Missing assets fail generation on every host.
Relative links in raw HTML are rejected; use Markdown links, images, or reference definitions instead.
Inline rules nest their headings and namespace Markdown reference labels.
Same-file fragment links in inline rules point to the standalone definition, avoiding ambiguity between repeated headings across rules.
