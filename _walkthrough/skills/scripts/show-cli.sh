#!/bin/bash
# Show the version and top-level help of the CLI the walkthrough built.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

show code-rules --version
show code-rules --help
show code-rules project --help
