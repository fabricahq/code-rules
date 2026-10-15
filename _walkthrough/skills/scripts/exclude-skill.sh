#!/bin/bash
# Take every skill the library offers except one, which the project excludes with a reason.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

cat > .syllabus/config.yaml <<'YAML'
schemaVersion: 2
skillTargets: [claude, agents]
sources:
  acme:
    repository: git@demo.invalid:acme-library.git
    rules:
      include:
        - practices/*
        - techs/go/wrap-errors
      exclude:
        - id: practices/testing/name-tests-by-behavior
          reason: We name tests after the function they cover.
    skills:
      include: ["*"]
      exclude:
        - id: skills/acme-release-notes
          reason: Release Planner writes our release notes.
  team:
    repository: git@demo.invalid:team-skills.git
    ref: v1.0.0
    skills:
      include:
        - review/pr-review
      uses:
        review/pr-review: [review/security-review]
YAML
show cat .syllabus/config.yaml
show syllabus project sync
show ls .claude/skills
show sed -n '/^## Excluded skills/,$p' .syllabus/generated/SKILLS.md
commit_all "Take every Acme skill but one"
show_files "$APP" demo-app
