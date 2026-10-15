#!/bin/bash
# Show the library's manifest: the library section of .code-rules/config.yaml.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$LIB"

show cat .code-rules/config.yaml
show code-rules library check
