#!/bin/bash
# Add skills from both sources: team-skills, a plain repository pinned to a tag, and one skill from the library.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

# A repository that isn't a library has no library releases, so it needs --ref: a tag or a full commit SHA.
show code-rules project add source team --repository git@demo.invalid:team-skills.git \
  --ref v1.0.0 --skills review/pr-review --non-interactive

# The library source already exists, so add its skills to config.yaml by hand.
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
  team:
    repository: git@demo.invalid:team-skills.git
    ref: v1.0.0
    skills:
      include:
        - review/pr-review
YAML
show cat .code-rules/config.yaml
show code-rules project sync
echo
echo "Installed copies, one per skill target:"
find .claude .agents -type f | sort
show ls -l .claude/skills/acme-review/scripts
show cat .code-rules/generated/installed-skills.json
commit_all "Install skills with Code Rules"
