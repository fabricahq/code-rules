#!/bin/bash
# Author a skill in the project, see build reject bad syllabus- metadata, then see the cited-rule links it writes.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"
SKILL=.code-rules/skills/release-checklist/SKILL.md
mkdir -p "$(dirname "$SKILL")"

echo "1. A skill type that isn't playbook or review:"
cat > "$SKILL" <<'MD'
---
name: release-checklist
description: Prepare a release of demo-app. Use when asked to cut a release.
metadata:
  syllabus-skill-type: checklist
---

1. Check that main is green.
MD
expect_failure code-rules project build

echo
echo "2. A syllabus- key Code Rules doesn't define:"
cat > "$SKILL" <<'MD'
---
name: release-checklist
description: Prepare a release of demo-app. Use when asked to cut a release.
metadata:
  syllabus-skill-type: playbook
  syllabus-owner: platform-team
---

1. Check that main is green.
MD
expect_failure code-rules project build

echo
echo "3. A bare rule ID that isn't one of the project's own rules:"
cat > "$SKILL" <<'MD'
---
name: release-checklist
description: Prepare a release of demo-app. Use when asked to cut a release.
metadata:
  syllabus-skill-type: playbook
  syllabus-rules-used: |
    practices/security/no-passwords-in-logs
---

1. Check that main is green.
MD
expect_failure code-rules project build

echo
echo "4. A valid skill. It cites a library rule, a library rule the project excludes, and an authored rule:"
cat > "$SKILL" <<'MD'
---
name: release-checklist
description: Prepare a release of demo-app. Use when asked to cut a release.
metadata:
  syllabus-skill-type: playbook
  syllabus-rules-used: |
    acme:practices/releases/write-release-notes
    acme:practices/testing/name-tests-by-behavior
    practices/security/no-secrets-in-logs
---

1. Check that main is green.
2. Draft the release notes.
3. Check new log lines for secrets.
MD
show code-rules project build
show cat .claude/skills/release-checklist/SKILL.md
commit_all "Add the release-checklist skill"
