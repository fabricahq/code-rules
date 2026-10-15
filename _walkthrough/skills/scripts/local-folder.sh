#!/bin/bash
# Show what happens to a project that still keeps authored rules in the old .syllabus/local/ folder.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

SCRATCH="$WORK/scratch/local-folder"
if [ ! -d "$SCRATCH" ]; then
  mkdir -p "$WORK/scratch"
  cp -R "$APP" "$SCRATCH"
fi
cd "$SCRATCH"
mkdir -p .syllabus/local/practices/legacy
printf 'name: Legacy\ndescription: Rules written before version 2\nwhenToRead: When touching old code\n' > .syllabus/local/practices/legacy/_group.yaml
printf -- '---\ntitle: An old rule\nwhenToRead: When touching old code\nimpact: LOW\nimpactDescription: Old habits\n---\n\nAn old rule.\n' > .syllabus/local/practices/legacy/old-rule.md
show find .syllabus/local -type f
expect_failure syllabus project build
