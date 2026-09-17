#!/bin/bash
# Initialize a separate library: Creates a manifest in a separate disposable directory. This demo declares no license; it does not publish or fetch anything.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/library"
"$CODE_RULES_DEMO_BINARY" library init
cat rule-library.json
