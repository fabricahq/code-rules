#!/bin/bash
# Check that Git 2.30 or later is installed. Syllabus needs it to fetch sources.

if ! command -v git >/dev/null 2>&1; then
  echo "Git isn't installed. Install it, for example with: xcode-select --install"
  exit 1
fi
version=$(git --version | sed -E 's/^git version ([0-9]+)\.([0-9]+).*/\1 \2/')
major=${version% *}
minor=${version#* }
echo "Found $(git --version)"
if [ "$major" -lt 2 ] || { [ "$major" -eq 2 ] && [ "$minor" -lt 30 ]; }; then
  echo "Syllabus needs Git 2.30 or later."
  exit 1
fi
