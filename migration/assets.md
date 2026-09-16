# Linked assets

PR #22 extends local library loading with complete rule-owned asset directories and referenced shared assets. Every retained Markdown file is inspected with Goldmark v2.1.1, including reference definitions; fenced code and inline code are ignored. The dependency owns CommonMark syntax. Our code owns path containment, supported destinations, and directory retention.

A link to another rule validates its existence without adopting that rule or reading its group. Shared assets are loaded only when referenced, then checked recursively. Cycles terminate. Missing targets, escaping paths, another rule's asset directory, malformed UTF-8 Markdown, LFS pointers, symlinks, hard links, reserved paths, and case/Unicode collisions fail without a partial catalog. Owned directories are complete, including unreferenced binary companions. All reads share the existing file, byte, discovery, and cancellation limits.

The new `rules.MarkdownLinks` API also records byte ranges in the original text, allowing a later renderer to change destinations without reserializing the original document. `MarkdownTargets` exposes the distinct resolved dependency inventory. The rule's original `Document` remains unchanged.

The walkthrough at `/walkthrough/pr22` exercises only attachments and links with temporary fixtures. Existing rule and group validation remains covered by its own tests. No Git operation, output rendering, or project installation is introduced here.

Validation includes real-filesystem success/failure probes and ten pinned TypeScript/Go Markdown discovery comparisons with independent expected targets. Broader renderer and Git-import scenarios remain pending in the migration inventory.
