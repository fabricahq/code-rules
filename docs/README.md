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
Shared typography and theme tokens live in `src/styles/custom.css`.
Homepage composition lives in `src/components/HomePage.astro` and `src/styles/home.css`.

Use the docs to explain the product experience and its contracts.
Keep implementation tasks in the internal planning material.
The root homepage and every documentation page identify the site as a design preview.
Update that notice when the release state changes, and publish installation instructions only after a package exists.

Before handing off visual changes, inspect desktop and narrow layouts, light and dark themes, keyboard navigation, and search in a production preview.
Run `bun run check` to verify website/tooling formatting, lint, types, tests, the static build, and links between built pages. The Go build tests also compile the copyable rule examples through the current resolver and renderer.
