#!/bin/bash
# Observe the expected check failure: Expect exit 1 and RULES.md in the changed list. The block is red intentionally: the CLI reports stale output and leaves the manual edit intact. Continue to Repair.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
"$CODE_RULES_DEMO_BINARY" check
