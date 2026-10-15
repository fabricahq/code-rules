#!/bin/bash
# Show what library check does with a rule-library.yaml manifest from before version 2.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

SCRATCH="$WORK/scratch/legacy-library"
if [ ! -d "$SCRATCH" ]; then
  mkdir -p "$WORK/scratch"
  cp -R "$LIB" "$SCRATCH"
fi
cd "$SCRATCH"
printf 'formatVersion: 1\nlicense:\n  spdxExpression: MIT\n  file: LICENSE.md\n' > rule-library.yaml
show cat rule-library.yaml
expect_failure code-rules library check
