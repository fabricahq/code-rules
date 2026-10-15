#!/bin/bash
# Show the library's manifest: the library section of .syllabus/config.yaml.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$LIB"

show cat .syllabus/config.yaml
show syllabus library check
