#!/bin/bash
# Delete the work folder and everything the walkthrough created in it.
set -eo pipefail
WORK="{{ .inputs.WorkDir }}"

if [ ! -f "$WORK/.code-rules-walkthrough" ]; then
  echo "$WORK isn't a walkthrough work folder, so it's left as it is."
  exit 1
fi
chmod -R u+w "$WORK"
rm -rf "$WORK"
echo "Deleted $WORK"
