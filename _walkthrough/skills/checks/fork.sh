#!/bin/bash
# Check that the fork is installed in place of the library's skill and still uses the library's acme-tdd.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

require_text .syllabus/config.yaml "replacedBy: skills/acme-fix-bug"
require_text .syllabus/skills/acme-fix-bug/SKILL.md "acme:skills/acme-tdd"
require_text .claude/skills/acme-fix-bug/SKILL.md "syllabus:installed skills/acme-fix-bug."
require_text .claude/skills/acme-fix-bug/SKILL.md "tests/regressions/"
require_text .syllabus/generated/SKILLS.md '- **Replaces:** `acme:skills/acme-fix-bug`, which the project excludes'
"$CR" project check >/dev/null || fail "syllabus project check reports problems."
pass "syllabus project check: up to date"
