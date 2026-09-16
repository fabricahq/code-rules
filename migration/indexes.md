# Bounded discovery indexes

`build.RenderIndexes` produces `RULES.md` and group summary indexes for a resolved rule set. Every summary links to the complete standalone rule. Effective local group guidance takes precedence. Empty groups remain discoverable. Group headers repeat the resolved names, group-level reading cues, and shared reading instructions from `RULES.md`, including on numbered pages. A local metadata definition replaces imported guidance; multiple imported definitions keep their source labels when no local override exists.

`build.IndexPages` measures UTF-8 bytes, preserves whole entries, and returns either one complete page or a directory plus numbered sibling parts. Every returned file fits the requested positive byte budget. An oversized entry or complete parts directory produces an error and no partial map.

This slice implements summary delivery only. [Full-rule group delivery](inline-groups.md) extends the complete build with automatic inline groups. Filesystem output remains a separate capability. The existing TypeScript CLI remains the complete executable. The native `/walkthrough/pr25` page lets reviewers vary fixture size and byte budgets and inspect every returned file.

The root selection procedure and numbered group instructions preserve the TypeScript renderer’s wording and Markdown structure. The group footer omits the provenance link because the summary-only API does not produce that file. Root reading and evidence instructions also appear on group pages so direct readers receive the same guidance. Group cues use the approved singular `whenToRead` string with an explicit label.
