#!/bin/bash
# Check without changing files: Exit 0 and empty added, changed, and removed arrays mean generated output matches its inputs. This runs the CLI with no tools on PATH.
set -euo pipefail
cd "${CODE_RULES_DEMO_ROOT:?Run Prepare first.}/project"
env PATH= "$CODE_RULES_DEMO_BINARY" check
