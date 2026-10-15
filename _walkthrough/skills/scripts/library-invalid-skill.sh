#!/bin/bash
# Add a broken skill to the library and see library check report every problem with it, then remove it.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$LIB"

mkdir -p .code-rules/skills/acme-deploy
cat > .code-rules/skills/acme-deploy/SKILL.md <<'MD'
---
name: deploy
description: Deploy a service.
metadata:
  syllabus-skill-type: runbook
  syllabus-rules-used: |
    practices/deploys/canary-first
---

1. Deploy.
MD
show cat .code-rules/skills/acme-deploy/SKILL.md
expect_failure code-rules library check
echo
echo "The same failure as JSON, with its error code:"
"$CR" library check --json || true

rm -r .code-rules/skills/acme-deploy
show code-rules library check
