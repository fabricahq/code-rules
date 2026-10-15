#!/bin/bash
# Check that the demo library, skills repository, and project exist.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1

git ls-remote --tags git@demo.invalid:acme-library.git | grep -q 'refs/tags/release/1$' || fail "acme-library has no release/1 tag."
pass "acme-library has library release 1 (tag release/1)."
git ls-remote --tags git@demo.invalid:team-skills.git | grep -q 'refs/tags/v1.0.0$' || fail "team-skills has no v1.0.0 tag."
pass "team-skills has tag v1.0.0."
require_text "$APP/.code-rules/config.yaml" "skillTargets: [claude, agents]"
