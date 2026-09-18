#!/bin/bash
# Open a supplementary Runbooks session whose Prepare step builds this checkout's CLI.
set -euo pipefail
runbook_dir=$(cd "$(dirname "$0")" && pwd)
CODE_RULES_DEMO_SOURCE_DIR=$(cd "$runbook_dir/../.." && pwd)
export CODE_RULES_DEMO_SOURCE_DIR
command -v runbooks >/dev/null
command -v go >/dev/null
printf 'Source checkout: %s\nRunbook: http://localhost:4392\nRun Prepare to build a temporary binary and print its path.\n' "$CODE_RULES_DEMO_SOURCE_DIR"
exec runbooks open "$runbook_dir/runbook.mdx" --port 4392 --working-dir ::tmp --no-telemetry
