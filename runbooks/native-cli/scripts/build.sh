#!/bin/bash
# Build and read the generated Markdown: Build creates agent-facing Markdown and provenance from the local rule. Inspect RULES.md and the Go group page in Generated Files after this step.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
printf 'Working directory: %s\n\n' "$PWD"
"$CODE_RULES_DEMO_BINARY" build
printf '\nFile: .code-rules/generated/RULES.md\n'
cat .code-rules/generated/RULES.md
printf '\nFile: .code-rules/generated/groups/techs/go.md\n'
cat .code-rules/generated/groups/techs/go.md
mkdir -p "$GENERATED_FILES/project"
cp -R .code-rules "$GENERATED_FILES/project/"
