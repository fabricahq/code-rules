#!/bin/bash
# Create team-skills, a plain Git repository of skills that isn't a library, tagged v1.0.0.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

if [ -d "$WORK/remotes/team-skills.git" ]; then
  echo "The demo skills repository already exists. Run the clean-up step at the end of the runbook to start over."
  exit 1
fi
git init -q --bare "$WORK/remotes/team-skills.git"
git clone -q git@demo.invalid:team-skills.git "$TEAM" 2>/dev/null
cd "$TEAM"

mkdir -p review/pr-review review/security-review writing/commit-message
cat > review/pr-review/SKILL.md <<'MD'
---
name: team-pr-review
description: Review a pull request. Use when asked to review a PR.
---

1. Read the description and the diff.
2. Run the team-security-review skill on the diff.
3. Leave comments that name the problem and a fix.
MD
cat > review/security-review/SKILL.md <<'MD'
---
name: team-security-review
description: Look for security problems in a diff. Use when reviewing code that handles input or secrets.
---

1. Find every place the diff reads input or handles a secret.
2. Check each one for injection, leaks, and missing authorization.
MD
cat > writing/commit-message/SKILL.md <<'MD'
---
description: Write a commit message. Use when committing.
---

Write a short subject in the imperative mood, then a body that says why.
MD
printf '# Team skills\n\nSkills our team shares. This repository is not a Syllabus library.\n' > README.md

commit_all "Add team skills"
git tag v1.0.0
git push -q origin main v1.0.0

echo "Skill folders (any folder with a SKILL.md):"
find . -name SKILL.md -not -path './.git/*' | sort
echo
echo "Tags in the hosted repository:"
git ls-remote --tags origin
show_files "$TEAM" team-skills
