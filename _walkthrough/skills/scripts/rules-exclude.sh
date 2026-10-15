#!/bin/bash
# Exclude one imported rule with a reason, using the version 2 list form.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

cat > .code-rules/config.yaml <<'YAML'
schemaVersion: 2
skillTargets: [claude, agents]
sources:
  acme:
    repository: git@demo.invalid:acme-library.git
    rules:
      include:
        - practices/*
        - techs/go/wrap-errors
      exclude:
        - id: practices/testing/name-tests-by-behavior
          reason: We name tests after the function they cover.
YAML
show cat .code-rules/config.yaml

# check notices that generated guidance no longer matches the configuration, and build fixes it offline.
expect_failure code-rules project check
show code-rules project build
show code-rules project check
