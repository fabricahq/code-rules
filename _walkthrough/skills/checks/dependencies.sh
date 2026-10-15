#!/bin/bash
# Check that the used skills are installed, and that SKILLS.md says which skill uses them.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

for skill in acme-tdd team-security-review; do
  [ -f ".claude/skills/$skill/SKILL.md" ] || fail "$skill isn't installed."
  pass "$skill installed"
done
require_text .syllabus/generated/SKILLS.md '- **Installed because:** used by `acme-build-feature`'
require_text .syllabus/generated/SKILLS.md '- **Installed because:** used by `team-pr-review`'
