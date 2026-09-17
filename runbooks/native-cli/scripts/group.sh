#!/bin/bash
# Add local Go guidance: Creates techs/go group metadata. Its description explains the subject; whenToRead tells the agent when to open it.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
"$CODE_RULES_DEMO_BINARY" local add group techs/go --name Go --description 'Go conventions for this project.' --when-to-read 'When writing or reviewing Go code.'
cat .code-rules/local/techs/go/_group.json
