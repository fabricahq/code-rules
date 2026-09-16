# Linked assets

PR #22 extends local library loading with complete rule-owned asset directories and referenced shared assets. Every retained Markdown file is inspected with Goldmark v2.1.1, including reference definitions and HTML href/src attributes; fenced code and inline code are ignored. The dependency owns CommonMark syntax. Our code owns path containment, supported destinations, and directory retention.

Rules and Markdown attachments cannot link to other rule documents, even if those rules are selected or retained. Self-links, anchors, allowed assets, declared terms, and external URLs remain supported. Put shared supporting explanations under `assets/` so rules remain independently selectable. This is a user-approved Go behavior change from the pinned TypeScript reference, which permits cross-rule links. Shared assets are loaded only when referenced, then checked recursively. Cycles terminate. Missing targets, escaping paths, another rule's asset directory, malformed UTF-8 Markdown, LFS pointers, symlinks, hard links, reserved paths, and case/Unicode collisions fail without a partial catalog. Owned directories are complete, including unreferenced binary companions. All reads share the existing file, byte, discovery, and cancellation limits.

`rules.MarkdownTargets` exposes the distinct resolved dependency inventory. Goldmark handles Markdown syntax; this slice applies source-relative path and ownership policy. The rule's original `Document` remains unchanged.

The walkthrough at `/walkthrough/pr22` exercises only attachments and links with temporary fixtures. Existing rule and group validation remains covered by its own tests. No Git operation, output rendering, or project installation is introduced here.

Validation includes real-filesystem success/failure probes and 13 pinned TypeScript/Go Markdown discovery comparisons with independent expected targets. Broader renderer and Git-import scenarios remain pending in the migration inventory.

The Go target additionally discovers real HTML `href` and `src` attributes in Markdown so attachments cannot bypass link ownership checks. Fenced examples, inline code, comments, and script text are not dependencies. This is an approved difference from the reference's Markdown-only discovery.
