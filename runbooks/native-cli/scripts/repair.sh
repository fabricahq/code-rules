#!/bin/bash
# Repair and check again: Build replaces the edited generated file from its source. Check then returns exit 0. The file browser now shows the repaired project.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
printf 'Working directory: %s\n\n' "$PWD"
"$CODE_RULES_DEMO_BINARY" build
"$CODE_RULES_DEMO_BINARY" check
mkdir -p "$GENERATED_FILES/project"
cp -R .code-rules "$GENERATED_FILES/project/"
