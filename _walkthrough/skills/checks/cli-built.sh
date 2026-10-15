#!/bin/bash
# Check that the walkthrough's CLI was built and runs.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1

[ -x "$CR" ] || fail "$CR doesn't exist; run the build step above."
version=$("$CR" --version) || fail "$CR --version failed."
pass "syllabus $version is at $CR"
"$CR" project add skill --help >/dev/null || fail "This build has no project add skill command; check that you built the claude/skills-preview branch."
pass "This build has the skills commands."
