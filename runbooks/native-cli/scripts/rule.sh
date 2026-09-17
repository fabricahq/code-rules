#!/bin/bash
# Add a complete rule: Writes an original Markdown body and creates a validated rule. Explicit flags make this step reproducible; the PR37 lab separately demonstrates terminal prompts.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
cat > body.md <<'BODY'
# Return errors to the caller

Return a descriptive error when an operation fails. Let the caller decide whether to retry, report, or stop.
BODY
"$CODE_RULES_DEMO_BINARY" local add rule techs/go/return-errors --title 'Return errors to the caller' --impact HIGH --impact-description 'Keep failures visible so callers can respond.' --when-to-read 'When calling fallible operations.' --body-file body.md
cat .code-rules/local/techs/go/return-errors.md
