#!/bin/bash
# Check that check warns about the personal skill, and still passes.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

out="$WORK/personal-skill-check.txt"
HOME="$WORK/fake-home" "$CR" project check > "$out" 2>&1 || { cat "$out"; fail "check failed; a personal skill should only warn."; }
pass "check passes"
require_text "$out" "Warning: your personal skill ~/.claude/skills/acme-tdd has the same name as this project's acme-tdd"
require_text "$out" "Claude Code uses your personal skill."
