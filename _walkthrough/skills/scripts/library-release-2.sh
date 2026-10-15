#!/bin/bash
# Change the library and publish library release 2: a rule change, a changed skill the project forked, and a
# changed and an added script in a skill the project installs.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$LIB"

# acme-fix-bug: the project forked it, so the update lists this as a replaced skill.
printf '4. Write the root cause in the pull request.\n' >> .code-rules/skills/acme-fix-bug/SKILL.md
# acme-review: one changed script and one new one.
printf '#!/bin/sh\n# List the files the current branch changes, with their status.\ngit diff --name-status main...HEAD\n' \
  > .code-rules/skills/acme-review/scripts/check-diff.sh
printf '#!/bin/sh\n# Run the linters.\ngolangci-lint run ./...\n' > .code-rules/skills/acme-review/scripts/run-linters.sh
chmod +x .code-rules/skills/acme-review/scripts/run-linters.sh
printf '3. Run scripts/run-linters.sh.\n' >> .code-rules/skills/acme-review/SKILL.md
# A rule change, which needs a change note now that the library has a release.
printf '\nPrefer a fake over a mock when the real dependency is slow.\n' >> .code-rules/rules/practices/testing/test-behavior.md
show code-rules library change practices/testing/test-behavior --bump minor --summary "Prefer fakes over mocks for slow dependencies."

show git status --short
commit_all "Improve the review and fix-bug skills"
git push -q origin main
show code-rules library check
show code-rules library release --no-github-release --non-interactive
