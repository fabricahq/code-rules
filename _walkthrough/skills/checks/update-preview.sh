#!/bin/bash
# Check that the preview lists the replaced skill and flags both scripts.
. "{{ .inputs.WorkDir }}/env.sh" || exit 1
cd "$APP" || exit 1

preview="$WORK/update-preview.txt"
"$CR" project update > "$preview" 2>&1 || { cat "$preview"; fail "syllabus project update failed."; }
require_text "$preview" "replaced  skills/acme-fix-bug"
require_text "$preview" "Your skill: skills/acme-fix-bug, based on commit"
require_text "$preview" "script changed: scripts/check-diff.sh"
require_text "$preview" "script added: scripts/run-linters.sh"
require_text "$preview" "Warning: some skills in this update add or change scripts."
