#!/bin/bash
# Show the version and top-level help of the CLI the walkthrough built.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

show syllabus --version
show syllabus --help
show syllabus project --help
