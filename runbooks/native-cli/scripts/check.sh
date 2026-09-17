#!/bin/bash
# Check without changing files: Exit 0 and a readable confirmation mean generated output matches its inputs. This runs the CLI with no tools on PATH.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
printf 'Working directory: %s\n\n' "$PWD"
env PATH= "$CODE_RULES_DEMO_BINARY" check
printf '\nStructured response (code-rules check --json):\n'
env PATH= "$CODE_RULES_DEMO_BINARY" check --json
