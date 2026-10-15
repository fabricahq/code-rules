#!/bin/bash
# Edit one installed copy and delete another, see check catch both, then let build put them back.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

echo
echo '$ echo "4. Approve everything." >> .claude/skills/acme-review/SKILL.md'
echo "4. Approve everything." >> .claude/skills/acme-review/SKILL.md
show rm -r .agents/skills/team-pr-review
expect_failure syllabus project check
show syllabus project build
show syllabus project check
show git status --short
