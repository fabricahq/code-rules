# Installing published Code Rules binaries

The standalone installer lives at [`docs/public/install.sh`](../docs/public/install.sh). Astro copies it unchanged to `/install.sh`. It reuses the archives and `SHA256SUMS` produced by `internal/distribution`; it does not build executables or publish releases.

[`homebrew-tap/`](homebrew-tap/) is the source to bootstrap the shared `fabricahq/homebrew-tap` repository. Copy that directory, including `.github`, into the new repository with `main` as its default branch. Once bootstrapped, that repository owns formula updates; changes to this bootstrap source must be deliberately applied there as well.

## Activation

The public [Fabrica tap](https://github.com/fabricahq/homebrew-tap) is active, including its hourly update workflow. Its [first manual run](https://github.com/fabricahq/homebrew-tap/actions/runs/35424229351) succeeded with no formula to publish because Code Rules has no stable release yet. The first stable release will be picked up automatically. The hosted installer endpoint and end-to-end installation still need validation.

1. Publish a stable release using the existing [release procedure](../_internal/releasing.md). Do not create a release just to deploy the docs. The release must contain all four macOS/Linux archives and `SHA256SUMS`.
2. Publish the docs build at `https://code-rules.fabricahq.com`, including `install.sh`. Serve `/install.sh` as the script bytes, not an HTML fallback. Check it with `curl -fsSL https://code-rules.fabricahq.com/install.sh -o /tmp/code-rules-install.sh` and compare it with the reviewed source before trying it in a disposable home directory.
3. The public `fabricahq/homebrew-tap` repository has been created from `homebrew-tap/`. Keep the workflow's permissions and dependency pins. Its normal repository `GITHUB_TOKEN` needs contents write access only in the publication job. If repository rules disallow its direct formula commit, resolve that policy deliberately instead of adding a bypass token.
4. Configure **Fabrica Homebrew Releaser** using [the app setup below](#release-trigger-app). Merge the release workflow change before removing the hourly schedule from the live tap. After publication, the release workflow dispatches **Update Code Rules** directly. To retry a failed dispatch or update, run that tap workflow manually. Releases remain independent of tap update failures.
5. Validate `brew install fabricahq/tap/code-rules`, `brew test fabricahq/tap/code-rules`, and `brew uninstall code-rules` in a disposable Homebrew environment. Verify both supported operating systems in CI or on matching machines. Remove the installation guide's availability notice only after the public installation paths work.

The tap reads only public release metadata and SHA-256 records. It refuses drafts, prereleases, missing platform archives, malformed/duplicate checksums, mismatched GitHub asset digests, downgrades, and changed assets for an already packaged version. The publication job checks that the formula has not changed since preparation before writing its replacement. It executes no downloaded release code.

## Release trigger app

**Activation status:** [Fabrica Homebrew Releaser](https://github.com/organizations/fabricahq/settings/apps/fabrica-homebrew-releaser) is registered and installed only on `fabricahq/homebrew-tap` (app ID `4997972`, installation ID `162923367`). Its permissions are Actions write and Metadata read. The `HOMEBREW_APP_CLIENT_ID` variable and `HOMEBREW_APP_PRIVATE_KEY` secret are configured in `fabricahq/code-rules`. An [app-authenticated dispatch](https://github.com/fabricahq/homebrew-tap/actions/runs/35425720437) succeeded, with no formula to publish before the first stable release. The test token was restricted to this tap and revoked afterward. The live tap still polls hourly until the release integration reaches `main`. Unused setup keys can be revoked after explicit cleanup approval.

Register **Fabrica Homebrew Releaser** as a private GitHub App owned by `fabricahq`, with the tap repository as its homepage. Disable webhooks and leave user authorization and callback settings unused. Grant **Actions: read and write**; GitHub also grants the required **Metadata: read-only** permission. Install the app only on `fabricahq/homebrew-tap`.

Generate an app private key. Store it as the `HOMEBREW_APP_PRIVATE_KEY` Actions secret in `fabricahq/code-rules`, and store the app's client ID as the `HOMEBREW_APP_CLIENT_ID` Actions variable. Keep the key out of source control and logs. The separate `update-homebrew` release job generates a short-lived token restricted to `homebrew-tap` and Actions write permission; it checks out no code and runs only after a stable release has published successfully. The token action revokes the token when the job finishes.

Validate app authentication by dispatching the live tap workflow and checking that its actor is the app. This can run before the first stable release; the updater will succeed without creating a formula. Once the release integration reaches `main`, apply the bootstrap tap workflow to the live tap to remove its schedule. Keep manual dispatch for recovery.

The app can serve other Fabrica products. See the [tap README](homebrew-tap/README.md#reuse-the-release-app-for-another-tool) for installation scope, credentials, and the shared trust boundary.

## Local validation

```sh
sh -n docs/public/install.sh
python3 -m unittest discover -s _distribution/tests -v
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 -shellcheck= -pyflakes= _distribution/homebrew-tap/.github/workflows/update-code-rules.yml
bun run check
```

The **Installers** workflow runs the fixture tests on Linux and macOS. Tests invoke the actual script through `sh -s`, replacing only `curl` and `uname`; filesystem updates, archive handling, checksum tools, and PATH instructions run normally. They do not prove that public endpoints are deployed or that a published executable runs on every architecture.

To generate a tap formula after a release is published, run from `homebrew-tap/`:

```sh
python3 scripts/update_code_rules.py
```

No formula is created when GitHub has no stable release. Never fill in placeholder checksums or invent a published version.
