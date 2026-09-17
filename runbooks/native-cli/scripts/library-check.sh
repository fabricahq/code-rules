#!/bin/bash
# Validate the whole library: Checks every library group, rule, and relevant attachment. Expect one group and one rule. An unlicensed-library warning is expected in this demo.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/library"
"$CODE_RULES_DEMO_BINARY" library check
mkdir -p "$GENERATED_FILES/library"
cp rule-library.json "$GENERATED_FILES/library/"
cp -R techs "$GENERATED_FILES/library/"
printf 'Inspect the files above or on disk: %s\n' "$CODE_RULES_DEMO_ROOT"
