#!/bin/bash
# Check that both skills are installed in both skill targets, with the script still executable.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

for target in .claude/skills .agents/skills; do
  for skill in acme-review team-pr-review; do
    [ -f "$target/$skill/SKILL.md" ] || fail "$target/$skill/SKILL.md is missing."
    pass "$target/$skill installed"
  done
  [ -x "$target/acme-review/scripts/check-diff.sh" ] || fail "$target/acme-review/scripts/check-diff.sh isn't executable."
  pass "$target/acme-review/scripts/check-diff.sh is executable"
done
"$CR" project check >/dev/null || fail "syllabus project check reports problems."
pass "syllabus project check: up to date"
