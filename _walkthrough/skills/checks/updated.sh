#!/bin/bash
# Check that the project is on library release 2 and the fork kept its own steps.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

require_text .syllabus/generated/SKILLS.md "from release/2"
[ -x .claude/skills/acme-review/scripts/run-linters.sh ] || fail "The new script isn't installed, or isn't executable."
pass "the new script is installed and executable"
grep -q "Write the root cause" .claude/skills/acme-fix-bug/SKILL.md && fail "The update changed your fork."
pass "the update didn't change your fork"
