#!/bin/bash
# Install skills that other skills use: one declared by the skill itself, one declared by the project.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

echo "acme-build-feature declares the skill it uses in its own metadata:"
show head -11 "$LIB/.code-rules/skills/acme-build-feature/SKILL.md"

cat > .code-rules/config.yaml <<'YAML'
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
      include:
        - skills/acme-review
        - skills/acme-build-feature
  team:
    repository: git@demo.invalid:team-skills.git
    ref: v1.0.0
    skills:
      include:
        - review/pr-review
      # team-pr-review calls team-security-review without declaring it, so the project declares it.
      uses:
        review/pr-review: [review/security-review]
YAML
show cat .code-rules/config.yaml

# build works offline, from vendor/, so it asks for a sync when the selection changes.
expect_failure code-rules project build
show code-rules project sync
echo
echo "What SKILLS.md says about why each skill is installed:"
grep -E '^### |Uses:|Installed because:' .code-rules/generated/SKILLS.md
commit_all "Install the skills our skills use"
