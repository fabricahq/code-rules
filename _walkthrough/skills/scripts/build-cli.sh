#!/bin/bash
# Get the code-rules source at the preview branch and build the CLI into the work folder.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

CHECKOUT="{{ .inputs.CodeRulesCheckout }}"
BRANCH="{{ .inputs.Branch }}"
if [ -n "$CHECKOUT" ]; then
  SRC="$CHECKOUT"
  echo "Using your checkout at $SRC, as it is:"
  git -C "$SRC" log -1 --format='%h %s (%cr)'
else
  SRC="$WORK/code-rules"
  if [ -d "$SRC/.git" ]; then
    show git -C "$SRC" fetch -q origin "$BRANCH"
    show git -C "$SRC" checkout -q -B "$BRANCH" FETCH_HEAD
  else
    show git clone -q --branch "$BRANCH" --single-branch https://github.com/fabricahq/code-rules.git "$SRC"
  fi
  git -C "$SRC" log -1 --format='Building %h %s (%cr)'
fi

cd "$SRC"
show go build -o "$CR" ./cmd/code-rules
echo
echo "Built $CR"
