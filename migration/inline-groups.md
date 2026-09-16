# Full-rule group delivery

`build.Prepare` includes every rule in a group page when the complete UTF-8 document fits both `IndexMaxBytes` and `GroupInlineMaxBytes`. The inline limit defaults to 8 KiB when omitted. An explicit zero forces summaries. Negative inline limits return a validation error.

If the whole group cannot fit, the renderer uses the existing summary and pagination behavior. It never includes only some bodies or truncates a rule. Empty groups keep an explicit empty-group message. Standalone rule files remain available in every mode, and all rules are validated before delivery selection can hide a bad link.

Inline rule headings nest below the group heading and instructions. Asset and license links are relative to the group file. Same-file fragments point to the corresponding standalone rule to avoid ambiguous anchors. Reference labels are unique per rule, so two rules can use the same authored label without linking to the wrong attachment. Input documents remain unchanged.

This preserves the TypeScript delivery decision in `src/builds/render.ts` and the link/heading behavior in `src/builds/markdown.ts`. The Go renderer continues to use its existing Markdown formatting, singular license declaration, and resolved local guidance. Generated reference labels use a stable rule-ID digest to keep labels below Markdown's length limit. This slice covers `builds.render.01` and the inline portions of `.02` and `.03`; it does not claim full migration completion.

The `/walkthrough/pr27` page invokes `build.Prepare` through native Go and shows both inline and fallback examples. PR25 still invokes the explicitly summary-only `build.RenderIndexes` interface. Project writes and the production CLI remain separate capabilities.

Validation: `go test -race ./...`, `go vet ./...`, and Staticcheck, plus browser checks of full-rule, fallback, reference-isolation, and error scenarios. Tests cover both exact byte limits, multibyte bodies, zero/negative limits, unchanged standalone output, and failure atomicity.
