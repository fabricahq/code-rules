# Installing published Code Rules binaries

The standalone installer lives at [`docs/public/install.sh`](../docs/public/install.sh). Astro copies it unchanged to `/install.sh`. It reuses the archives and `SHA256SUMS` produced by `internal/distribution`; it does not build executables or publish releases.

The executable embeds the repository's `LICENSE.md`, available with `code-rules --license`. Release archives also include that file; the installer places only the executable in the installation directory.

The installer appends shell-specific PATH setup after a successful installation. Pass `--no-update-path` to leave startup files untouched. Tests use isolated homes and exercise profile preservation, repeated installs, quoting, and manual fallback.

## Activation

1. Publish a stable release using the existing [release procedure](../_engineering/releasing.md). Do not create a release just to deploy the docs. The release must contain all four macOS/Linux archives and `SHA256SUMS`.
2. Publish the docs build at `https://code-rules.fabricahq.com`, including `install.sh`. Serve `/install.sh` as the script bytes, not an HTML fallback. Check it with `curl -fsSL https://code-rules.fabricahq.com/install.sh -o /tmp/code-rules-install.sh` and compare it with the reviewed source before trying it in a disposable home directory.
3. Validate installation, upgrades, and removal in a disposable environment on both supported operating systems. Check that only the executable is installed and that `code-rules --license` prints its embedded license. The hosted endpoint and end-to-end installation still need validation.

## Homebrew ownership and automatic updates

The [Fabrica tap](https://github.com/fabricahq/homebrew-tap) owns Homebrew formula generation, validation, tests, and publication. Keep the updater in that repository; this repository owns the CLI's release binaries and checksums.

After publishing a stable release, the [release workflow](../.github/workflows/release.yml) triggers the tap's **Update Code Rules** workflow through Fabrica Homebrew Releaser. The tap validates the release assets and commits the formula directly to `main`. Routine formula updates require no pull request or human approval. Prereleases do not update Homebrew.

Changes to updater code and workflows go through review in the tap repository. If an automatic update fails, investigate the failure and rerun the tap workflow manually. A failed tap update leaves the published CLI release intact. See [release integration and recovery](../_engineering/releasing.md#homebrew-updates) and the [tap's maintenance instructions](https://github.com/fabricahq/homebrew-tap#readme).

Validate `brew install fabricahq/tap/code-rules`, `brew test fabricahq/tap/code-rules`, and removal against a published release in a disposable environment before enabling Homebrew installation instructions.

## Local validation

```sh
sh -n docs/public/install.sh
go test -race -count=1 ./internal/distribution -run '^TestStandaloneInstaller$'
bun run check
```

The **Installers** workflow runs the Go tests on Linux and macOS. The suite builds a native executable for an installation smoke test; no prebuilt binary or Python runtime is required. Most tests replace `curl` and `uname`; the native smoke test uses the runner's actual platform. Filesystem updates, archive handling, checksum tools, and PATH instructions run normally.

These tests do not prove that the public endpoint is deployed or that a published executable runs on every architecture. Run the tap's own tests in its repository when changing Homebrew update behavior.
