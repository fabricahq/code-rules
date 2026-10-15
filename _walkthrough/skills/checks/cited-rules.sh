#!/bin/bash
# Check that the installed release-checklist links to each rule it cites.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

file=.claude/skills/release-checklist/SKILL.md
require_text "$file" "## Rules this skill applies"
require_text "$file" '`acme:practices/releases/write-release-notes` 1.0.0: [.syllabus/generated/rules/acme/practices/releases/write-release-notes.md](../../../.syllabus/generated/rules/acme/practices/releases/write-release-notes.md)'
require_text "$file" '`acme:practices/testing/name-tests-by-behavior`: excluded by this project: We name tests after the function they cover.'
require_text "$file" '`local:practices/security/no-secrets-in-logs`: [.syllabus/generated/rules/local/practices/security/no-secrets-in-logs.md]'
