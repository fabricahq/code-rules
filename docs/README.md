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

Use Tailwind utilities for component layout, spacing, typography, and interaction states.
The site uses Tailwind v4 through its Vite plugin and [Starlight’s Tailwind integration](https://starlight.astro.build/guides/css-and-tailwind/).
Starlight owns the reset; do not add Tailwind Preflight alongside it.

- `src/styles/tailwind.css` defines CSS layer order, scans `src/` for utilities, and exposes the existing palette as Tailwind tokens.
  It also owns the shared button, section-label, and aside styles.
- `src/styles/custom.css` owns the light/dark theme values and overrides for generated Starlight navigation and Markdown content.
- `src/styles/home.css` handles the generated Starlight homepage shell and rule-example code markup.
- Component styles remain for illustration palette values, playback visibility, and generated syntax highlighting that cannot take utility classes directly.

Prefer semantic colors such as `text-ink`, `text-muted`, and `bg-surface`.
They follow Starlight’s saved theme and automatic system preference.
Use the shared responsive variants instead of inline viewport queries:

- `sm:` starts at 30rem (480px), matching Starlight's smaller file-tree cutoff.
- `md:` starts at 50rem (800px), matching Starlight's desktop navigation and sidebar.
- `lg:` starts at 72rem (1152px), matching Starlight's wide layout and right sidebar.

These are defined together in `src/styles/tailwind.css`; pixel equivalents assume the default browser font size.
Use `max-sm:` and `max-md:` for smaller layouts, or stack variants such as `sm:max-md:` for a range.
Starlight hardcodes its own media queries, so keep our definitions aligned with its installed `style/util.css` and `user-components/FileTree.astro` when upgrading.
Tailwind's default breakpoint scale is disabled to avoid mixing two conventions.
Keep complete utility names in source so Tailwind can find them.
Do not construct partial class names dynamically.

Use the docs to explain the product experience and its contracts.
Keep implementation tasks in the internal planning material.
Documentation pages identify the site as a design preview; the marketing homepage intentionally omits the release notice.
Keep release availability explicit in the project status and installation guides, and publish installation instructions only after a package exists.

Before handing off visual changes, inspect desktop and narrow layouts, light and dark themes, keyboard navigation, and search in a production preview.
Run `bun run check` to verify website/tooling formatting, lint, types, tests, the static build, and links between built pages. The Go build tests also compile the copyable rule examples through the current resolver and renderer.
