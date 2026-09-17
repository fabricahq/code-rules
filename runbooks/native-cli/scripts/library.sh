#!/bin/bash
# Initialize a separate library: Creates a manifest in a separate disposable directory. This demo declares no license; it does not publish or fetch anything.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/library"
printf 'Working directory: %s\n\n' "$PWD"
"$CODE_RULES_DEMO_BINARY" library init
printf '\nFile: rule-library.json\n'
cat rule-library.json
mkdir -p "$GENERATED_FILES/library"
cp rule-library.json "$GENERATED_FILES/library/"
printf '\nSnapshot available in Generated Files: library/\n'
