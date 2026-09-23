# Installing published Code Rules binaries

The standalone installer lives at [`docs/public/install.sh`](../docs/public/install.sh). Astro copies it unchanged to `/install.sh`. It reuses the archives and `SHA256SUMS` produced by `internal/distribution`; it does not build executables or publish releases.

The executable embeds the repository's `LICENSE.md`, available with `code-rules --license`. Release archives also include that file; the installer places only the executable in the installation directory.

The installer appends shell-specific PATH setup after a successful installation. Pass `--no-update-path` to leave startup files untouched. Tests use isolated homes and exercise profile preservation, repeated installs, quoting, and manual fallback.

## Activation

1. Publish a stable release using the existing [release procedure](../_engineering/releasing.md). Do not create a release just to deploy the docs. The release must contain all four macOS/Linux archives and `SHA256SUMS`.
2. Publish the docs build at `https://code-rules.fabricahq.com`, including `install.sh`. Serve `/install.sh` as the script bytes, not an HTML fallback. Check it with `curl -fsSL https://code-rules.fabricahq.com/install.sh -o /tmp/code-rules-install.sh` and compare it with the reviewed source before trying it in a disposable home directory.
3. Validate installation, upgrades, and removal in a disposable environment on both supported operating systems. Check that only the executable is installed and that `code-rules --license` prints its embedded license. The hosted endpoint and end-to-end installation still need validation.

## Local validation

```sh
sh -n docs/public/install.sh
go test -race -count=1 ./internal/distribution -run '^TestStandaloneInstaller$'
bun run check
```

The **Installers** workflow runs the Go tests on Linux and macOS. The suite builds a native executable for an installation smoke test; no prebuilt binary or Python runtime is required. Most tests replace `curl` and `uname`; the native smoke test uses the runner's actual platform. Filesystem updates, archive handling, checksum tools, and PATH instructions run normally.

These tests do not prove that the public endpoint is deployed or that a published executable runs on every architecture.
