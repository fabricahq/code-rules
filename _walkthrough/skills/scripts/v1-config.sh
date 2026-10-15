#!/bin/bash
# Show what a version 1 configuration does now: every command refuses it and shows the version 2 shape.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

SCRATCH="$WORK/scratch/v1-config"
mkdir -p "$SCRATCH/.syllabus"
cd "$SCRATCH"
[ -d .git ] || git init -q
cat > .syllabus/config.yaml <<'YAML'
schemaVersion: 1
sources:
  acme:
    repository: git@demo.invalid:acme-library.git
    groups: [practices/testing]
    rules: [techs/go/wrap-errors]
    exclude:
      practices/testing/name-tests-by-behavior:
        reason: We name tests after the function they cover.
YAML
show cat .syllabus/config.yaml
expect_failure syllabus project check
