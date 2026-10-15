#!/bin/bash
# Get the syllabus source at the preview branch and build the CLI into the work folder.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

BRANCH="{{ .inputs.Branch }}"
SRC="$WORK/syllabus"
if [ -d "$SRC/.git" ]; then
  show git -C "$SRC" fetch -q origin "$BRANCH"
  show git -C "$SRC" checkout -q -B "$BRANCH" FETCH_HEAD
else
  show git clone -q --branch "$BRANCH" --single-branch https://github.com/fabricahq/code-rules.git "$SRC"
fi
git --no-pager -C "$SRC" log -1 --format='Building %h %s (%cr)'

cd "$SRC"
# go version downloads Go 1.27.2 first when your Go is older, and says so. Building with -v then prints each package
# as it compiles, so a first build that downloads modules and compiles everything shows its progress.
show go version
show go build -v -o "$CR" ./cmd/syllabus
echo
echo "Built $CR"
