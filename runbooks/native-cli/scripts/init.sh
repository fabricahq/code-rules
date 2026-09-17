#!/bin/bash
# Initialize the project: Creates .code-rules/config.json and the local rules directory. No source is fetched.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
"$CODE_RULES_DEMO_BINARY" init
cat .code-rules/config.json
