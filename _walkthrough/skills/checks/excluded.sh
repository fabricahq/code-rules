#!/bin/bash
# Check that the excluded skill isn't installed, is listed with its reason, and stays in vendor/.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

[ ! -e .claude/skills/acme-release-notes ] || fail "acme-release-notes is installed, but the project excludes it."
pass "acme-release-notes isn't installed"
require_text .code-rules/generated/SKILLS.md '- `acme:skills/acme-release-notes`: Release Planner writes our release notes.'
[ -f .code-rules/vendor/acme/skills/acme-release-notes/SKILL.md ] || fail "The excluded skill isn't in vendor/."
pass "the excluded skill stays in vendor/, so updates can compare it"
