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
The rendered-link checker lives in `_tools/link-checker/`. `cli.ts` owns command arguments and exit status; `check-site.ts` coordinates validation. HTML parsing, external requests, public-network restrictions, and their tests stay inside that folder.
The root TypeScript, lint, and test commands include these Bun scripts; Astro checks the site code.
`bun run docs:links` uses Bun and parse5 to check local destinations and fragments in the built HTML.
For a different build directory, run `bun docs/_tools/link-checker/cli.ts <directory>` from the repository root.
The production build includes the search index.
These commands do not publish the site.

## Publishing

The Documentation workflow publishes the site, including `install.sh`, to GitHub Pages at <https://code-rules.fabricahq.com> after a push to `main` passes every docs check, including the external link check. Other branches only run the checks, and pull requests also get a [preview](#pull-request-previews).
The site is served at the domain root, and page links are root-relative; do not set Astro's `base`.
GitHub Pages settings, the custom domain, and its DNS record are managed in Fabrica's infrastructure repository, not here.

### Pull request previews

When a pull request into `main` changes anything under `docs/`, its build is published as a public preview on Cloudflare Pages, at `pr-<number>.code-rules-docs-previews.pages.dev`. A pull request comment links the preview and the changed pages. Every push updates the preview, and closing the pull request deletes its previews. A daily run also deletes any previews of closed pull requests that were left behind. Previews aren't indexed by search engines.

The repository is public, so anyone can open a pull request, and a build runs the pull request's code. The design keeps that code away from the Cloudflare token, which can change every Pages project in Fabrica's account:

1. The Documentation workflow builds and checks the site on a GitHub-hosted runner with no secrets, then uploads `docs/dist` as the `docs-site` artifact.
2. The Documentation previews workflow (`.github/workflows/docs-preview.yml`) runs main's copy of itself and its scripts in `.github/docs-preview/`. It never checks out pull request code, and treats the artifact as data.
3. Before deploying, it rejects symlinks and the top-level `_worker.js`, `functions`, `_routes.json`, `_headers`, and `_redirects` entries. Pages would run the first two as server code and use the others to redirect visitors or change headers. It adds its own `_headers`, with a Content Security Policy that keeps pages from loading anything from other sites or submitting forms. It runs wrangler, pinned by main's lockfile, from an empty directory, then confirms Cloudflare recorded a live static preview.
4. A build is matched to its pull request by repository, branch, and commit. Previews of branches in this repository opened by authors with write access deploy automatically. Others wait in the `docs-preview-approval` environment until a maintainer reviews the commit and approves the deployment from the workflow run that the comment links. A newer push cancels a pending approval.
5. Only the `docs-preview` environment holds the `DOCS_PREVIEW_CLOUDFLARE_API_TOKEN` secret, and it admits deployments from `main` only.

The workflow fails rather than deploy if `docs-preview` admits any branch but `main`, or `docs-preview-approval` has no required reviewers. GitHub creates a missing environment without protection the first time a job uses it.

The scripts' tests run with `bun run check`; the deploy job installs their own `package.json` so wrangler stays out of the site's dependencies. The Pages project is managed in Fabrica's infrastructure repository. To set up the repository side once:

```sh
repo=fabricahq/code-rules
for environment in docs-preview docs-preview-approval; do
  gh api -X PUT "repos/$repo/environments/$environment" --input - <<'EOF'
{"deployment_branch_policy": {"protected_branches": false, "custom_branch_policies": true}}
EOF
  gh api -X POST "repos/$repo/environments/$environment/deployment-branch-policies" -f name=main -f type=branch
done
gh api -X PUT "repos/$repo/environments/docs-preview-approval" --input - <<'EOF'
{"reviewers": [{"type": "User", "id": 4295964}],
 "deployment_branch_policy": {"protected_branches": false, "custom_branch_policies": true}}
EOF
gh secret set DOCS_PREVIEW_CLOUDFLARE_API_TOKEN --env docs-preview --repo "$repo"
```

Use a Cloudflare API token limited to the Fabrica account, with only **Cloudflare Pages: Edit** permission and an expiry date, kept in 1Password. User `4295964` is `josh-padnick`; add other maintainers as reviewers.

## Maintain the docs

Write pages under `src/content/docs/` in folders matching the sidebar sections: `start-here/`, `concepts/`, `guides/`, `reference/`, and `for-agents/`.
Keep the homepage at `index.mdx` and navigation in `astro.config.mjs` aligned with page slugs. When moving a public page, update links and add a redirect from its old URL.
Homepage composition lives in `src/components/HomePage.astro` and its illustration components.

### Styling

Follow the [CSS and Tailwind guidelines](_internal/css.md) for styling ownership, responsive breakpoints, design tokens, and reuse.

### Product documentation

Use the docs to explain the product experience and its contracts.
Keep implementation tasks in the internal planning material.
Keep release availability accurate in the installation guide.

Before handing off visual changes, inspect desktop and narrow layouts, light and dark themes, keyboard navigation, and search in a production preview.
Run `bun run check` to verify website/tooling formatting, lint, types, tests, the static build, and links between built pages. It doesn't run Go tests: `go test ./internal/build` renders the copyable rule examples through the current resolver and renderer.

### Link checks

`bun run check` checks links and anchors within the built site, including absolute URLs to `code-rules.fabricahq.com`. To also check external links, run `bun run docs:build && bun run docs:links:all` from the repository root.

The Documentation workflow runs both checks on every push and pull request. A broken link fails the workflow, even when the change does not touch a Markdown file.

The external check follows redirects and verifies anchors in static HTML. It permits only public network destinations, including after redirects. It fetches each URL once, retries temporary failures, and exits with an error listing the source pages and failing links. HTTP errors (including access denials), timeouts, and oversized HTML responses fail the check; they are never silently treated as valid links.

Checks cover rendered hyperlinks, including linked downloads. URLs in code examples are not hyperlinks. Email links and other non-HTTP schemes are skipped. Fragments in non-HTML downloads are not checked; anchors created only by client-side JavaScript cannot be verified. Do not add broad exclusions to hide a failed check: fix the link, use a stable destination, or retry if the site is temporarily unavailable.
