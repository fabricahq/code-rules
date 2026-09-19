#!/bin/sh
# Install an official Code Rules release without sudo or shell-profile edits.
# Downloads and checksums come from the same release; they are not independent signatures.

# Keep execution at the end so a truncated piped download cannot start installation.
main() (
    set -eu
    version=latest
    install_dir=${HOME:?HOME must be set}/.local/bin
    repository=https://github.com/fabricahq/code-rules
    temporary=
    staging=
    trap 'rm -rf "$temporary" "$staging"' EXIT
    trap 'exit 1' HUP INT TERM

    fail() { printf 'code-rules installer: %s\n' "$*" >&2; exit 1; }
    download() {
        curl --fail --silent --show-error --location --retry 3 \
            --proto '=https' --proto-redir '=https' --tlsv1.2 "$@"
    }
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --version|--install-dir)
                [ "$#" -ge 2 ] || fail "$1 requires a value"
                case "$1" in
                    --version) version=${2#v} ;;
                    --install-dir) install_dir=$2 ;;
                esac
                shift 2 ;;
            --help|-h)
                printf '%s\n' 'Usage: sh install.sh [--version VERSION] [--install-dir DIRECTORY]' \
                    'Defaults: latest stable release, ~/.local/bin. Rerun to upgrade.' \
                    'No sudo is used and no shell configuration is changed.'
                exit 0 ;;
            *) fail "unknown option: $1 (see --help)" ;;
        esac
    done
    case "$install_dir" in
        /*) ;;
        *) fail '--install-dir must be an absolute path' ;;
    esac
    case "$install_dir" in
        *:*) fail 'installation directory cannot contain a colon (PATH separator)' ;;
        *'
'*) fail 'installation directory cannot contain a newline' ;;
    esac
    for command in curl tar mktemp awk uname chmod mv mkdir; do
        command -v "$command" >/dev/null 2>&1 || fail "required command not found: $command"
    done
    if command -v sha256sum >/dev/null 2>&1; then
        checksum=sha256sum
    elif command -v shasum >/dev/null 2>&1; then
        checksum=shasum
    else
        fail 'install sha256sum or shasum to verify release downloads'
    fi
    case "$(uname -s)" in
        Darwin) os=darwin ;;
        Linux) os=linux ;;
        *) fail 'supported operating systems are macOS and Linux' ;;
    esac
    case "$(uname -m)" in
        arm64|aarch64) arch=arm64 ;;
        x86_64|amd64) arch=amd64 ;;
        *) fail 'supported processor architectures are arm64 and amd64' ;;
    esac
    if [ "$version" = latest ]; then
        resolved=$(download --output /dev/null --write-out '%{url_effective}' "$repository/releases/latest") ||
            fail 'cannot find the latest release; check GitHub Releases or use --version'
        case "$resolved" in
            "$repository"/releases/tag/v*) version=${resolved##*/v} ;;
            *) fail 'GitHub did not resolve a published release' ;;
        esac
    fi
    # Restrict versions before inserting them into filenames or release URLs.
    printf '%s\n' "$version" | LC_ALL=C grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$' ||
        fail 'version must look like 0.1.0 or 0.2.0-rc.1'
    archive=code-rules_${version}_${os}_${arch}.tar.gz
    temporary=$(mktemp -d "${TMPDIR:-/tmp}/code-rules-download.XXXXXX")
    base=$repository/releases/download/v$version
    printf 'Downloading Code Rules %s for %s/%s...\n' "$version" "$os" "$arch"
    download --output "$temporary/SHA256SUMS" "$base/SHA256SUMS" || fail 'could not download release checksums'
    download --output "$temporary/archive.tar.gz" "$base/$archive" || fail "could not download $archive"
    expected=$(awk -v file="$archive" '$2 == file { count++; hash=$1 } END { if (count != 1 || length(hash) != 64 || hash ~ /[^0-9a-f]/) exit 1; print hash }' "$temporary/SHA256SUMS") ||
        fail 'release checksums must contain exactly one valid entry for this archive'
    if [ "$checksum" = sha256sum ]; then
        actual=$(sha256sum "$temporary/archive.tar.gz")
    else
        actual=$(shasum -a 256 "$temporary/archive.tar.gz")
    fi
    actual=${actual%% *}
    [ "$actual" = "$expected" ] || fail 'checksum mismatch; existing installation was not changed'

    # Extract only named regular files to stdout, never archive-controlled paths or links.
    tar -tzf "$temporary/archive.tar.gz" > "$temporary/members" || fail 'invalid release archive'
    awk '($0 != "code-rules" && $0 != "LICENSE.md" && $0 != "README.txt") || seen[$0]++ { exit 1 } END { if (!seen["code-rules"] || !seen["LICENSE.md"]) exit 1 }' "$temporary/members" ||
        fail 'unexpected or missing release archive files'
    tar -tvzf "$temporary/archive.tar.gz" > "$temporary/types" || fail 'invalid release archive'
    awk 'substr($0,1,1) != "-" { exit 1 }' "$temporary/types" || fail 'release archive contains a link or non-regular file'
    [ ! -L "$install_dir/code-rules" ] || fail 'destination is a symlink; choose another --install-dir'
    [ ! -d "$install_dir/code-rules" ] || fail 'destination is a directory; choose another --install-dir'
    [ ! -L "$install_dir/code-rules.LICENSE" ] && [ ! -d "$install_dir/code-rules.LICENSE" ] || fail 'license destination is a symlink or directory'
    mkdir -p "$install_dir" || fail 'cannot create installation directory; choose a writable --install-dir'
    staging=$(mktemp -d "$install_dir/.code-rules-install.XXXXXX")
    tar -xzOf "$temporary/archive.tar.gz" code-rules > "$staging/code-rules"
    tar -xzOf "$temporary/archive.tar.gz" LICENSE.md > "$staging/code-rules.LICENSE"
    [ -s "$staging/code-rules" ] && [ -s "$staging/code-rules.LICENSE" ] || fail 'release executable or license is empty'
    chmod 755 "$staging/code-rules"
    chmod 644 "$staging/code-rules.LICENSE"
    # Check that the verified executable runs before replacing a working installation.
    "$staging/code-rules" --version || fail 'downloaded executable cannot run; existing installation was not changed'
    mv -f "$staging/code-rules.LICENSE" "$install_dir/code-rules.LICENSE"
    mv -f "$staging/code-rules" "$install_dir/code-rules"
    printf 'Installed Code Rules %s to %s/code-rules\n' "$version" "$install_dir"
    case ":${PATH:-}:" in
        *:"$install_dir":*) ;;
        *)
            printf '\nAdd this directory to your PATH: %s\n' "$install_dir"
            printf '%s\n' 'For sh, bash, or zsh, run the following; add it to your shell profile only if you want it to persist:'
            quoted=$(printf '%s' "$install_dir" | sed "s/'/'\\\\''/g")
            printf "export PATH='%s':\"\$PATH\"\n" "$quoted"
            printf '%s\n' 'For fish, add the directory with fish_add_path. No shell files were changed.' ;;
    esac
    existing=$(command -v code-rules || true)
    if [ -n "$existing" ] && [ "$existing" != "$install_dir/code-rules" ]; then
        printf 'PATH currently selects %s. Put %s first to use this installation.\n' "$existing" "$install_dir"
    fi
)
main "$@"
