#!/bin/bash
# Add local Go guidance: Creates techs/go group metadata. Its description explains the subject; whenToRead tells the agent when to open it.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
printf 'Working directory: %s\n\n' "$PWD"
"$CODE_RULES_DEMO_BINARY" local add group techs/go --name Go --description 'Go conventions for this project.' --when-to-read 'When writing or reviewing Go code.'
printf '\nFile: .code-rules/local/techs/go/_group.json\n'
cat .code-rules/local/techs/go/_group.json
mkdir -p "$GENERATED_FILES/project"
cp -R .code-rules "$GENERATED_FILES/project/"
printf '\nSnapshot available in Generated Files: project/.code-rules/\n'
