#!/bin/bash
# Show SKILLS.md, the pointer to it in RULES.md, and the marker line in an installed copy.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

show cat .code-rules/generated/SKILLS.md
show grep -n SKILLS.md .code-rules/generated/RULES.md
show head -7 .claude/skills/acme-review/SKILL.md
