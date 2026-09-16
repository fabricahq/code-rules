# Standalone rule rendering

`build.RenderRules` turns a `build.Resolve` result into generated-path-to-Markdown values. It writes no files and leaves authoritative source documents unchanged.

Each standalone rule includes applicability, impact, guidance, origin, attribution, license links, and original frontmatter. Goldmark identifies actual Markdown links and reference definitions. Source edits relocate destinations while leaving other body text intact. Empty destinations point to the retained original file. Fragment-only links remain local to the standalone document. Missing local links and relative raw-HTML references fail.

Retained paths use workspace-relative links. Unretained rule paths on recognized hosts use immutable commit URLs. Supporting assets must be retained; they cannot silently fall back to a remote URL. Generated license destinations are fixed, with copying supplied by the licensed-output slice.

A matching leading plain-text title is omitted. Top-level headings nest below Guidance. Reference labels need no namespace in standalone files; combining rule bodies is intentionally outside this slice. Group delivery, index pagination, provenance, and output persistence are not implemented here.

The `/walkthrough/pr24` native walkthrough exercises complete returned files, retained assets, pinned links, references, empty destinations, code examples, Setext headings, and error cases.
