# Release the CLI

The package is `@fabricahq/code-rules`; its executable is `code-rules`. Node.js 22.18+ and macOS/Linux are supported. CI tests the packed executable with Node 22.18 and 24 on both platforms.

## Prepare publication once

1. Confirm the npm organization/package is owned by Fabrica. Do not publish under an unrelated existing name.
2. Approve the tool's distribution license, add `LICENSE.md`, and set `package.json.license`. The release guard rejects `UNLICENSED` or missing license text.
3. Bootstrap the first npm package under the owner account if required, then configure its trusted publisher for GitHub repository `fabricahq/code-rules`, workflow `publish.yml`, environment `npm`. Permit direct publishing. Keep credentials out of the repository.
4. Configure the GitHub `npm` environment with your release approval policy. A branch push runs tests only; publication starts on a published GitHub release.

See [npm trusted publishing](https://docs.npmjs.com/trusted-publishers/) for account setup and first-publication constraints. The workflow uses npm 11 and OIDC, not a stored npm token.

## Prepare each version

1. Set a complete semantic version in `package.json` and refresh `bun.lock`. Keep prerelease versions on npm's `next` tag.
2. Run `bun run check` and `bun run test:package`. The package test installs into a fresh prefix, uses Node through the executable's shebang, and runs from temporary projects.
3. Inspect `npm pack --dry-run`. Only the executable, sourcemap, template, README, package metadata, and approved license belong in the tarball. Runtime dependencies remain normal npm dependencies.
4. Install that tarball into a real pilot repository, add local rules, import a compatible library, and inspect licensing/provenance. Commit the result. Record the pilot evidence before declaring the first release ready.
5. For later releases, test upgrading a committed project from the previous published version, then regenerate and check. The first-release suite proves reinstall preservation; it cannot prove migration from a release that does not exist yet.
6. Merge the release changes. Create tag `v<version>` on the reviewed commit and publish a matching GitHub release. Mark prereleases as prereleases.

`publish.yml` checks the tag, channel, and license; runs repository checks; packs once; tests that exact tarball; uploads it as a workflow artifact; and publishes it with provenance. No PR creates tags or publishes automatically.

## Local walkthrough

Use the locally packed artifact for review before npm publication. The walkthrough must invoke the installed executable, expose one command per step, and show the actual project files. Fixture transport routing, when needed to keep the example local, must be disclosed and confined to the walkthrough process.
