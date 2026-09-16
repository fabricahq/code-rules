# Standalone rule rendering

`build.RenderRules` turns a `build.Resolve` result into generated-path-to-Markdown values. It writes no files and leaves authoritative source documents unchanged.

Each standalone rule includes applicability, impact, guidance, origin, attribution, license links, and original frontmatter. Goldmark identifies actual Markdown links and reference definitions. Source edits relocate destinations while leaving other body text intact. Empty destinations point to the retained original file. Fragment-only links remain local to the standalone document. Missing local links and relative raw-HTML references fail.

Retained paths use workspace-relative links. Filesystem links to other rule documents fail, whether their targets are retained, excluded, or unselected. No remote fallback makes those dependencies valid. Supporting assets must be retained; they cannot silently fall back to a remote URL. Generated license destinations are fixed, with copying supplied by the licensed-output slice.

A matching leading plain-text title is omitted. Headings, including those inside lists and blockquotes, nest below Guidance. Shared reference definitions used by images and ordinary links point to the same retained files. Reference labels need no namespace in standalone files; combining rule bodies is intentionally outside this slice. Group delivery, index pagination, provenance, and output persistence are not implemented here.

The `/walkthrough/pr24` native walkthrough exercises complete returned files, retained assets, rejected cross-rule links, references, empty destinations, code examples, Setext headings, and error cases.
