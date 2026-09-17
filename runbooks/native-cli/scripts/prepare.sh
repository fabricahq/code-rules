#!/bin/bash
# Build a fresh native CLI and print a copyable terminal setup for an isolated playground; retain earlier runs for inspection.
set -euo pipefail
: "${CODE_RULES_DEMO_SOURCE_DIR:?Start this runbook with open.sh first.}"
command -v go >/dev/null
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/code-rules-runbook-bin.XXXXXX")
CODE_RULES_DEMO_BINARY="$build_dir/code-rules"
export CODE_RULES_DEMO_BINARY
printf 'Building Code Rules from %s\n' "$CODE_RULES_DEMO_SOURCE_DIR"
(cd "$CODE_RULES_DEMO_SOURCE_DIR" && go build -o "$CODE_RULES_DEMO_BINARY" ./cmd/code-rules)
CODE_RULES_DEMO_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/code-rules-runbook.XXXXXX")
export CODE_RULES_DEMO_ROOT
mkdir "$CODE_RULES_DEMO_ROOT/project" "$CODE_RULES_DEMO_ROOT/library" "$CODE_RULES_DEMO_ROOT/playground"
"$CODE_RULES_DEMO_BINARY" --help
printf '\nBinary: %s\nGuided workspace: %s\n' "$CODE_RULES_DEMO_BINARY" "$CODE_RULES_DEMO_ROOT"
printf '\nCopy these commands into your terminal (bash or zsh):\n\n'
printf 'export CODE_RULES_DEMO_BINARY=%q\n' "$CODE_RULES_DEMO_BINARY"
printf 'cd %q\n' "$CODE_RULES_DEMO_ROOT/playground"
cat <<'COMMANDS'

# Explore the CLI.
"$CODE_RULES_DEMO_BINARY" --help
"$CODE_RULES_DEMO_BINARY" local add rule --help

# Missing project: expect exit 1 and an error in JSON.
"$CODE_RULES_DEMO_BINARY" check --json
echo "Exit status: $?"

# Unknown option: expect exit 2 and a readable error.
"$CODE_RULES_DEMO_BINARY" --not-a-flag
echo "Exit status: $?"

# Initialize this playground, then follow its README to add groups and rules.
"$CODE_RULES_DEMO_BINARY" init
cat .code-rules/README.md
COMMANDS
printf '\nThe binary and playground remain on disk after this runbook stops.\nRe-run Prepare to build current source into a new temporary directory.\n'
