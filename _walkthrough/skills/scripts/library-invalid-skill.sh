#!/bin/bash
# Add a broken skill to the library and see library check report every problem with it, then remove it.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$LIB"

mkdir -p .syllabus/skills/acme-deploy
cat > .syllabus/skills/acme-deploy/SKILL.md <<'MD'
---
name: deploy
description: Deploy a service.
metadata:
  syllabus-skill-type: runbook
  syllabus-rules-used: |
    practices/deploys/canary-first
  syllabus-skills-used: |
    skills/acme-rollback
---

1. Deploy.
MD
show cat .syllabus/skills/acme-deploy/SKILL.md
expect_failure syllabus library check
echo
echo "The same failure as JSON, with its error code:"
"$CR" library check --json || true

rm -r .syllabus/skills/acme-deploy
show syllabus library check
