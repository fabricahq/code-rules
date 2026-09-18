#!/bin/bash
# Deliberately edit generated output: Appends a demo-only line to the disposable RULES.md. This simulates a manual edit that the next check must report.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
printf 'Working directory: %s\n\n' "$PWD"
test -f .code-rules/generated/RULES.md
printf '\nManual demo edit.\n' >> .code-rules/generated/RULES.md
tail -4 .code-rules/generated/RULES.md
mkdir -p "$GENERATED_FILES/project"
cp -R .code-rules "$GENERATED_FILES/project/"
printf '\nSnapshot available in Generated Files: project/.code-rules/\n'
