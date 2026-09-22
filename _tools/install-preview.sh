#!/bin/sh
# Install a preview as ./code-rules, preserving the old file until download and chmod succeed.
# Usage: sh install-preview.sh HOST OWNER/REPO MAC_ARM MAC_INTEL LINUX_ARM LINUX_INTEL
# The four platform arguments are immutable GitHub artifact IDs from one approved build.
set -eu

if [ "$#" -ne 6 ]; then
  echo "Usage: install-preview.sh HOST OWNER/REPO MAC_ARM MAC_INTEL LINUX_ARM LINUX_INTEL" >&2
  exit 1
fi

hostname=$1
repository=$2
platform=$(uname -sm)
case "$platform" in
  'Darwin arm64') artifact=$3 ;;
  'Darwin x86_64') artifact=$4 ;;
  'Linux aarch64') artifact=$5 ;;
  'Linux x86_64') artifact=$6 ;;
  *) echo "Unsupported platform: $platform" >&2; exit 1 ;;
esac

if [ -d ./code-rules ]; then
  printf '%s\n' "Cannot install: $(pwd)/code-rules is an existing folder." \
    "Nothing was changed. Run this command from a different directory." >&2
  exit 1
fi

# A temporary file in the destination directory permits replacement with a single rename.
tmp=$(mktemp ./.code-rules.XXXXXX)
trap 'rm -f "$tmp"' EXIT
# archive:false uploads return raw bytes despite the artifact API's /zip suffix.
gh api --hostname "$hostname" "/repos/$repository/actions/artifacts/$artifact/zip" > "$tmp"
test -s "$tmp"
chmod +x "$tmp"
mv -f "$tmp" ./code-rules
echo "Downloaded and wrote ./code-rules"
