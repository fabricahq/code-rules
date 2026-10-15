#!/bin/bash
# Preview the update to library release 2. Without a terminal, update only shows the preview.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"
cd "$APP"

show code-rules project update
