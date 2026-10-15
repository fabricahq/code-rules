#!/bin/bash
# Author a project-only rule group and rule. They live in .code-rules/rules/, laid out like a library's.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

show code-rules project add group practices/security --name Security \
  --description "How demo-app handles secrets" --when-to-read "When code reads input or handles secrets" --non-interactive
printf 'Never write secrets, tokens, or passwords to logs. Evidence: log calls pass only non-secret fields.\n' > "$WORK/rule-body.md"
show code-rules project add rule practices/security/no-secrets-in-logs --title "Keep secrets out of logs" \
  --impact CRITICAL --impact-description "Logs are widely readable" --when-to-read "When adding a log line" \
  --body-file "$WORK/rule-body.md" --non-interactive
show code-rules project build
echo
echo "Authored rules:"
find .code-rules/rules -type f | sort
echo
echo "Generated copy, under the source name local:"
find .code-rules/generated/rules/local -type f | sort
