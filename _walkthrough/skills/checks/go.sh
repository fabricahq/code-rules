#!/bin/bash
# Check that Go is installed. The walkthrough builds the CLI from source.

if ! command -v go >/dev/null 2>&1; then
  echo "Go isn't installed. Install it from https://go.dev/dl/ or with: brew install go"
  exit 1
fi
echo "Found $(go version)"
version=$(go env GOVERSION | sed -E 's/^go([0-9]+)\.([0-9]+).*/\1 \2/')
major=${version% *}
minor=${version#* }
if [ "$major" -lt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -lt 21 ]; }; then
  echo "This Go is too old to download the Go version Code Rules needs. Install Go 1.21 or later."
  exit 1
fi
echo "Code Rules needs Go 1.27.2. If yours is older, the build downloads that toolchain into the work folder."
