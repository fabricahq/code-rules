#!/bin/bash
# Apply the update, then confirm the fork kept your change.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

show syllabus project update --yes
show syllabus project check
show ls -l .claude/skills/acme-review/scripts
echo
echo "Your fork is unchanged; the library's new step 4 isn't in it:"
show tail -3 .claude/skills/acme-fix-bug/SKILL.md
commit_all "Update Acme to library release 2"
