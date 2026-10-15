#!/bin/bash
# Create a personal skill named like an installed one, in a stand-in home folder, and see sync and check warn.
# HOME points at the stand-in only for these commands, so nothing reads or writes your real home folder.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

FAKE_HOME="$WORK/fake-home"
mkdir -p "$FAKE_HOME/.claude/skills/acme-tdd"
printf -- '---\nname: acme-tdd\ndescription: My own TDD routine.\n---\n\nMy own steps.\n' > "$FAKE_HOME/.claude/skills/acme-tdd/SKILL.md"
show find "$FAKE_HOME" -type f

printf '\n$ HOME=%s code-rules project check\n' "$FAKE_HOME"
HOME="$FAKE_HOME" "$CR" project check
printf '\n$ HOME=%s code-rules project sync\n' "$FAKE_HOME"
HOME="$FAKE_HOME" "$CR" project sync
