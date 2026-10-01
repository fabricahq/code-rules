# Distribution

This note covers how users get the Code Rules binary: the installation channels, what each one verifies, and what they depend on. [Releases](releasing.md) covers how the release files are built, attested, and published. The [installation guide](../docs/src/content/docs/start-here/install.md) owns the user-facing instructions.

## Channels

Every channel installs files from a published [GitHub Release](https://github.com/fabricahq/code-rules/releases). None of them builds the binary.

- **Standalone installer.** [`docs/public/install.sh`](../docs/public/install.sh) is owned by this repository. Astro copies it unchanged into the site build, and the [Documentation workflow](../.github/workflows/docs.yml) publishes it to GitHub Pages as `https://code-rules.fabricahq.com/install.sh` after every push to `main` that passes the docs checks. The site deploys independently of releases, so an installer change reaches users when it merges, not at the next release. By default the script asks GitHub for the latest stable release when it runs; `--version` selects any release, including a prerelease.
- **Homebrew.** [fabricahq/homebrew-tap](https://github.com/fabricahq/homebrew-tap) owns formula generation, tests, and publication. After a stable release, the release workflow runs the tap's **Update Code Rules** workflow; see [Homebrew updates](releasing.md#homebrew-updates) for the trigger and its credentials. The tap ignores the dispatched tag and version and resolves the latest stable release itself. It generates the formula from that release's archives and checksums, installs and tests it on each native platform, then commits it to the tap's `main`.
- **Manual download.** Users download an archive and `SHA256SUMS`, verify them, and extract the executable, as the installation guide's [manual download section](../docs/src/content/docs/start-here/install.md) describes.
- **Library CI.** `code-rules library init` writes [`internal/library/check-workflow.yml`](../internal/library/check-workflow.yml) to a library's `.github/workflows/code-rules.yml`. Its install step downloads one exact version's `linux_amd64` archive and `SHA256SUMS` with the GitHub CLI, verifies the attestation and the checksum, and extracts only the executable. The version is the one that ran `init`. A development build (`0.0.0-...`) has no release to pin, so its workflow looks up the latest release when it runs.
- **PR previews.** Maintainers can download preview executables from a PR. They are unreleased, unattested, and not an installation channel; see [Testing PR preview builds](releasing.md#testing-pr-preview-builds).

## Trust chain

1. On the release PR, [build-release.yml](../.github/workflows/build-release.yml) builds the four archives, `manifest.json`, and `SHA256SUMS` from the release commit, with read-only permissions and no secrets. Release Planner attests each file's build provenance in the PR's run.
2. Merging the PR approves publication. The `publish` job verifies each file's attestation, then publishes exactly the files the PR built.
3. The `attest-release` job downloads the published files and attests them again from `main`. Only approved, published files have this attestation, so verifying channels require a signature from `.github/workflows/release-planner.yml` with source ref `refs/heads/main`.

What each channel checks:

| Channel | Attestation | Checksum |
| --- | --- | --- |
| Standalone installer | No | The archive must match its single entry in `SHA256SUMS` |
| Homebrew tap | `SHA256SUMS`, from `release-planner.yml` on `main` and a GitHub-hosted runner | The formula records each archive's checksum, and Homebrew checks it on install |
| Manual download | `SHA256SUMS`, if the user has the GitHub CLI | The user compares the archive's checksum |
| Library CI | `SHA256SUMS`, from `release-planner.yml` on `main`, on every run | The archive's line must appear in `SHA256SUMS` |

The standalone installer downloads `SHA256SUMS` and the archive from the same release over HTTPS. The checksum catches a corrupt or truncated download, not a release whose files were replaced together; verifying an attestation would require the GitHub CLI and a GitHub login, which the installer doesn't assume. It also checks the archive's members and runs the new executable's `--version` before replacing an existing installation.

### The release file contract

Channels outside this repository depend on these details, and some copies can't be updated:

- **Archive names:** `code-rules_<version>_<os>_<arch>.tar.gz` for `darwin` and `linux` on `amd64` and `arm64`, in a release tagged `v<version>`. The installer, the library workflow, and the tap's formula generator build these names.
- **`SHA256SUMS`:** one `<sha256>  <file>` line per archive, as `sha256sum` prints it. The installer requires exactly one entry for its archive, and the library workflow matches the whole line.
- **Archive members:** `code-rules`, `README.txt`, and `LICENSE.md`, all regular files. The installer rejects any other member, a link, or a missing executable or license. The formula installs `code-rules` and `LICENSE.md`.
- **Attestation signer:** `fabricahq/code-rules/.github/workflows/release-planner.yml`, from `refs/heads/main`. The tap, the manual instructions, and every generated library workflow name it.

Renaming an archive, dropping a platform, or changing the checksum format makes the installer fail for that release, fails the tap update, and fails every library workflow pinned to it. Renaming the release workflow or publishing from another branch makes every attestation check reject new releases. Generated library workflows are committed in other repositories, so a contract change must keep working for the workflows already in use, or ship with instructions for regenerating them. `build-release.yml` fails a build whose file set or archive names differ from the contract.

## What the binary carries

The executable is a static build (`CGO_ENABLED=0`) whose version is set at link time. It embeds the repository's [`LICENSE.md`](../LICENSE.md) through [`license.go`](../license.go), so `code-rules --license` prints it in every build. Each release archive holds the executable, a short `README.txt`, and `LICENSE.md`; `manifest.json` and `SHA256SUMS` are separate release files.

The standalone installer places only the executable in `~/.local/bin`, or the `--install-dir` directory; the embedded license covers the installed copy. It then adds PATH setup once to the startup files of zsh, bash, sh, dash, or fish, unless `--no-update-path` is passed. It never uses `sudo`. Homebrew installs the executable and puts `LICENSE.md` in the formula's prefix.

## When something breaks

- **A release is published but the tap update failed.** The release is unaffected. The tap opens an issue for a failed update, and its daily drift check opens one when the formula lags the latest stable release. Rerun as [Homebrew updates](releasing.md#homebrew-updates) describes; any rerun publishes the formula for the latest stable release, whichever release triggered it.
- **`/install.sh` is served as HTML or as a stale copy.** Compare it with `main`: `curl -fsSL https://code-rules.fabricahq.com/install.sh | cmp - docs/public/install.sh`. GitHub Pages responses can be cached for a few minutes. If the difference persists, check the latest Documentation run on `main`: a failed check blocks deployment. Pages settings and the custom domain are managed in `infra-live`.
- **An archive or architecture is missing.** `build-release.yml` refuses an incomplete bundle, and publication stages files on a draft release and publishes last, so a failed run leaves no partially published release; retry it as [Release Planner's guide](https://release-planner.fabricahq.com/start-here/release/) describes. Don't upload a missing file by hand: it has no attestation and no entry in the attested `SHA256SUMS`. Publish a new release instead.
- **An attestation check fails.** Check that the command names the signer workflow and source ref above and that the GitHub CLI is logged in. Then check the release's `attest-release` job; if it failed, use **Re-run failed jobs** on the Release run. The `downstream` job also waits for that job. If the attestation exists and still doesn't match, treat the files as untrusted and don't work around the check.

## Verifying an installer change

Run the local checks:

```sh
sh -n docs/public/install.sh
go test -race -count=1 ./internal/distribution -run '^TestStandaloneInstaller$'
bun run check
```

The [Installers workflow](../.github/workflows/installers.yml) runs the same syntax check and test on Linux and macOS, with zsh and fish installed so the PATH setup is evaluated by each supported shell. The tests replace `curl` and serve generated archives, so they cover every platform's archive name, upgrades, checksum and archive failures, and PATH setup without the network. They replace `uname` too, except in one smoke test that installs and runs a natively built executable. They don't download a published release, run the published executables, or check the deployed endpoint.

Before merging, run the changed script against a published release in a disposable home:

```sh
home=$(mktemp -d)
env -i HOME="$home" PATH=/usr/bin:/bin SHELL=/bin/zsh sh docs/public/install.sh --version 0.1.0
ls -A "$home/.local/bin"
cat "$home/.zshrc"
"$home/.local/bin/code-rules" --license | head -n 3
```

Run the install line twice. Expect only `code-rules` in the installation directory and one PATH entry in `.zshrc`. After the merge deploys, check the hosted copy as described above. When changing Homebrew behavior, run the tap's own tests in its repository.
