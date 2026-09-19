# Installing published Code Rules binaries

The standalone installer lives at [`docs/public/install.sh`](../docs/public/install.sh). Astro copies it unchanged to `/install.sh`. It reuses the archives and `SHA256SUMS` produced by `internal/distribution`; it does not build executables or publish releases.

[`homebrew-tap/`](homebrew-tap/) is the source to bootstrap the shared `fabricahq/homebrew-tap` repository. Copy that directory, including `.github`, into the new repository with `main` as its default branch. Once bootstrapped, that repository owns formula updates; changes to this bootstrap source must be deliberately applied there as well.

## Activation

Neither endpoint is live just because this branch contains its source. As of this implementation, GitHub reports no published Code Rules releases and no `fabricahq/homebrew-tap` repository.

1. Publish a stable release using the existing [release procedure](../_internal/releasing.md). Do not create a release just to deploy the docs. The release must contain all four macOS/Linux archives and `SHA256SUMS`.
2. Publish the docs build at `https://code-rules.fabricahq.com`, including `install.sh`. Serve `/install.sh` as the script bytes, not an HTML fallback. Check it with `curl -fsSL https://code-rules.fabricahq.com/install.sh -o /tmp/code-rules-install.sh` and compare it with the reviewed source before trying it in a disposable home directory.
3. Create the public `fabricahq/homebrew-tap` repository from `homebrew-tap/`. Keep the workflow's permissions and dependency pins. Its normal repository `GITHUB_TOKEN` needs contents write access only in the publication job; no cross-repository token or personal access token is required. If repository rules disallow its direct formula commit, resolve that policy deliberately instead of adding a bypass token.
4. Run **Update Code Rules** manually for the first stable release. The workflow checks for new stable releases hourly after that. GitHub schedules can be delayed. Releases remain independent of tap update failures; rerun the tap workflow to recover.
5. Validate `brew install fabricahq/tap/code-rules`, `brew test fabricahq/tap/code-rules`, and `brew uninstall code-rules` in a disposable Homebrew environment. Verify both supported operating systems in CI or on matching machines. Remove the installation guide's availability notice only after the public installation paths work.

The tap reads only public release metadata and SHA-256 records. It refuses drafts, prereleases, missing platform archives, malformed/duplicate checksums, mismatched GitHub asset digests, downgrades, and changed assets for an already packaged version. The publication job checks that the formula has not changed since preparation before writing its replacement. It executes no downloaded release code.

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
