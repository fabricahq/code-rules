#!/bin/bash
# Prepare a disposable workspace: Creates a fresh workspace and prints CLI help. Re-running this step starts a new demo; previous files remain available at the printed path.
set -euo pipefail
test -x "${CODE_RULES_DEMO_BINARY:?Start this runbook with open.sh first.}"
CODE_RULES_DEMO_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/code-rules-runbook.XXXXXX")
export CODE_RULES_DEMO_ROOT
mkdir "$CODE_RULES_DEMO_ROOT/project" "$CODE_RULES_DEMO_ROOT/library"
printf 'Workspace: %s\nNative CLI: %s\n' "$CODE_RULES_DEMO_ROOT" "$CODE_RULES_DEMO_BINARY"
"$CODE_RULES_DEMO_BINARY" --help
