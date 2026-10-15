#!/bin/bash
# Fork the library's acme-fix-bug into the project, change it, and install it in place of the library's skill.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

show code-rules project add skill --from acme:skills/acme-fix-bug \
  --reason "We file a regression test with every fix." --non-interactive
echo
echo "The exclusion the command added to config.yaml:"
grep -A3 'id: skills/acme-fix-bug' .code-rules/config.yaml
show cat .code-rules/skills/acme-fix-bug/SKILL.md

printf '4. Add the failing test to tests/regressions/ and link the bug.\n' >> .code-rules/skills/acme-fix-bug/SKILL.md
show code-rules project build
show sed -n '/^### acme-fix-bug/,/Installed at/p' .code-rules/generated/SKILLS.md
commit_all "Fork acme-fix-bug"
