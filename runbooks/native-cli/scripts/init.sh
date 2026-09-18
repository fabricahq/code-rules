#!/bin/bash
# Initialize the project: Creates the agent README, configuration, and local rules directory. No source is fetched.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
printf 'Working directory: %s\n\n' "$PWD"
"$CODE_RULES_DEMO_BINARY" init
printf '\nFile: .code-rules/config.json\n'
cat .code-rules/config.json
mkdir -p "$GENERATED_FILES/project"
cp -R .code-rules "$GENERATED_FILES/project/"
printf '\nSnapshot available in Generated Files: project/.code-rules/\n'
