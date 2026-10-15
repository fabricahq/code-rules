#!/bin/bash
# Check that the selectors took every practice rule and only one Go rule.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

rules=.code-rules/generated/rules/acme
for rule in practices/testing/test-behavior practices/testing/name-tests-by-behavior practices/releases/write-release-notes techs/go/wrap-errors; do
  [ -f "$rules/$rule.md" ] || fail "$rule wasn't imported."
  pass "$rule imported"
done
[ ! -f "$rules/techs/go/table-tests.md" ] || fail "techs/go/table-tests was imported, but no selector names it."
pass "techs/go/table-tests not imported: no selector covers it"
