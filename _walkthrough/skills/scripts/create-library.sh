#!/bin/bash
# Create the demo library, acme-library: a few rules and five skills, published as library release 1.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

if [ -d "$WORK/remotes/acme-library.git" ]; then
  echo "The demo library already exists. Run the clean-up step at the end of the runbook to start over."
  exit 1
fi

# The "hosted" copy, reached as git@demo.invalid:acme-library.git, and your working copy.
git init -q --bare "$WORK/remotes/acme-library.git"
git clone -q git@demo.invalid:acme-library.git "$LIB" 2>/dev/null
cd "$LIB"

mkdir -p .syllabus
cat > .syllabus/config.yaml <<'YAML'
schemaVersion: 2
sources: {}
library:
  license:
    spdxExpression: MIT
    file: LICENSE.md
    notices: []
YAML
cat > LICENSE.md <<'TEXT'
MIT License

Copyright (c) Acme

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
documentation files, to deal in the software without restriction, subject to the above copyright notice being
included in all copies. THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.
TEXT

# write_group GROUP NAME DESCRIPTION WHEN_TO_READ
write_group() {
  mkdir -p ".syllabus/rules/$1"
  printf 'name: %s\ndescription: %s\nwhenToRead: %s\n' "$2" "$3" "$4" > ".syllabus/rules/$1/_group.yaml"
}
# write_rule RULE TITLE WHEN_TO_READ BODY
write_rule() {
  printf -- '---\ntitle: %s\nwhenToRead: %s\nimpact: HIGH\nimpactDescription: Agents repeat this mistake without the rule.\n---\n\n%s\n' "$2" "$3" "$4" > ".syllabus/rules/$1.md"
}

write_group practices/testing "Testing" "How Acme tests code" "Before writing or changing tests"
write_rule practices/testing/test-behavior "Test behavior, not implementation" "When writing or reviewing tests" \
  "Test what code does through its public interface. Evidence: tests call exported functions and assert on results."
write_rule practices/testing/name-tests-by-behavior "Name tests after the behavior they check" "When naming a test" \
  "Name each test for the behavior it proves, such as TestRetryStopsAfterThreeAttempts."
write_group practices/releases "Releases" "How Acme ships releases" "Before preparing a release"
write_rule practices/releases/write-release-notes "Write release notes for people" "When writing release notes" \
  "Describe what changed for the people who use the software, not the commits."
write_group techs/go "Go" "How Acme writes Go" "Before writing or reviewing Go code"
write_rule techs/go/wrap-errors "Wrap errors with context" "When returning an error in Go" \
  "Wrap each returned error with what the code was doing, using fmt.Errorf with %w."
write_rule techs/go/table-tests "Use table-driven tests" "When writing Go tests" \
  "Write related Go test cases as a table of inputs and expected results."

# write_skill NAME DESCRIPTION METADATA BODY
write_skill() {
  mkdir -p ".syllabus/skills/$1"
  printf -- '---\nname: %s\ndescription: %s\nmetadata:\n%s---\n\n%s\n' "$1" "$2" "$3" "$4" > ".syllabus/skills/$1/SKILL.md"
}

write_skill acme-build-feature "Build a feature one slice at a time. Use when starting a feature." \
'  syllabus-skill-type: playbook
  syllabus-rules-used: |
    practices/testing/test-behavior
    techs/go/wrap-errors
  syllabus-skills-used: |
    skills/acme-tdd
' '1. Name the slice you are building.
2. Use the acme-tdd skill to write a failing test, then make it pass.
3. Wrap every error you return.'

write_skill acme-tdd "Drive a change with a failing test first. Use when changing behavior." \
'  syllabus-skill-type: playbook
  syllabus-rules-used: |
    practices/testing/test-behavior
' '1. Write a test that fails for the behavior you want.
2. Make it pass with the smallest change.
3. Refactor.'

write_skill acme-review "Review a change against Acme rules. Use when asked for a review." \
'  syllabus-skill-type: review
' '1. Run scripts/check-diff.sh to list the changed files.
2. Check each change against the rules the project includes.'
mkdir -p .syllabus/skills/acme-review/scripts
printf '#!/bin/sh\n# List the files the current branch changes.\ngit diff --name-only main...HEAD\n' > .syllabus/skills/acme-review/scripts/check-diff.sh
chmod +x .syllabus/skills/acme-review/scripts/check-diff.sh

write_skill acme-release-notes "Draft release notes. Use when preparing a release." \
'  syllabus-skill-type: playbook
  syllabus-rules-used: |
    practices/releases/write-release-notes
' '1. List the changes since the last release.
2. Write one line per change, for the people who use the software.'

write_skill acme-fix-bug "Fix a reported bug. Use when asked to fix a bug." \
'  syllabus-skill-type: playbook
  syllabus-skills-used: |
    skills/acme-tdd
' '1. Reproduce the bug.
2. Use the acme-tdd skill to capture it in a failing test.
3. Fix it.'

commit_all "Add Acme rules and skills"
git push -q origin main

echo "The library's files:"
find .syllabus LICENSE.md -type f | sort
echo
show syllabus library check
show syllabus library release --no-github-release --non-interactive
echo
echo "Tags in the hosted library:"
git ls-remote --tags origin
