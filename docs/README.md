# Code Rules documentation

The [Fabrica Code Rules](https://code-rules.fabricahq.com) website documents the Go CLI and its file formats.

The site uses Astro and Starlight, with human-facing guides, a reference, and a dedicated “For agents” section.
Its visual direction follows [FM Linear's documentation](https://github.com/josh-padnick/fm-linear/tree/main/docs): monochrome typography, generous spacing, and rounded example panels.
The starting design reference is commit `bbb6bb3f1b94d0506120277179128066270f2e3b`.
Shared theme tokens and the accessible-callout transform are adapted from that project.
Homepage content and illustrations are original Code Rules examples; no FM Linear brand imagery is included.

## Run locally

From the repository root:

```sh
bun install --frozen-lockfile
bun run docs:dev
```

Use the local URL printed by Astro.

```sh
bun run docs:check
bun run docs:build
bun run docs:links
bun run docs:preview
```

The output is `docs/dist/`.
The rendered-link checker and its command-level tests live in `_tools/` beside these docs.
The root TypeScript, lint, and test commands include these Bun scripts; Astro checks the site code.
`bun run docs:links` uses Bun and parse5 to check local destinations and fragments in the built HTML.
For a different build directory, run `bun docs/_tools/check-doc-links.ts <directory>` from the repository root.
The production build includes the search index.
Hosting is not configured, and these commands do not publish the site.
Set Astro's `site` option after selecting a production URL; sitemap generation is skipped until then.

## Maintain the docs

Write pages under `src/content/docs/` and keep navigation in `astro.config.mjs` aligned with their slugs.
Homepage composition lives in `src/components/HomePage.astro` and its illustration components.

### Styling

Follow the [CSS and Tailwind guidelines](_internal/css.md) for styling ownership, responsive breakpoints, design tokens, and reuse.

### Product documentation

Use the docs to explain the product experience and its contracts.
Keep implementation tasks in the internal planning material.
Documentation pages identify the site as a design preview; the marketing homepage intentionally omits the release notice.
Keep release availability explicit in the project status and installation guides, and publish installation instructions only after a package exists.

Before handing off visual changes, inspect desktop and narrow layouts, light and dark themes, keyboard navigation, and search in a production preview.
Run `bun run check` to verify website/tooling formatting, lint, types, tests, the static build, and links between built pages. The Go build tests also compile the copyable rule examples through the current resolver and renderer.

### Link checks

`bun run check` checks links and anchors within the built site, including absolute URLs to `code-rules.fabricahq.com`. To also check external links, run `bun run docs:build && bun run docs:links:all` from the repository root.

The external check follows redirects and verifies anchors in static HTML. It permits only public network destinations, including after redirects. It fetches each URL once, retries temporary failures, and exits with an error listing the source pages and failing links. HTTP errors (including access denials), timeouts, and oversized HTML responses fail the check; they are never silently treated as valid links.

Checks cover rendered hyperlinks, including linked downloads. URLs in code examples are not hyperlinks. Email links and other non-HTTP schemes are skipped. Fragments in non-HTML downloads are not checked; anchors created only by client-side JavaScript cannot be verified. Do not add broad exclusions to hide a failed check: fix the link, use a stable destination, or retry if the site is temporarily unavailable.
