#!/bin/bash
# Build the checkout's native CLI and open an isolated, supplementary Runbooks session.
set -euo pipefail
runbook_dir=$(cd "$(dirname "$0")" && pwd)
source_dir=$(cd "$runbook_dir/../.." && pwd)
command -v runbooks >/dev/null
command -v go >/dev/null
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/code-rules-runbook-bin.XXXXXX")
export CODE_RULES_DEMO_BINARY="$build_dir/code-rules"
(cd "$source_dir" && go build -o "$CODE_RULES_DEMO_BINARY" ./cmd/code-rules)
printf 'Native CLI: %s\nRunbook: http://localhost:4392\n' "$CODE_RULES_DEMO_BINARY"
exec runbooks open "$runbook_dir/runbook.mdx" --port 4392 --working-dir ::tmp --no-telemetry
