#!/bin/bash
# Create demo-app, a small Git repository, and set up Syllabus in it.
set -eo pipefail
. "{{ .inputs.WorkDir }}/env.sh"

if [ -d "$APP" ]; then
  echo "The demo project already exists. Run the clean-up step at the end of the runbook to start over."
  exit 1
fi
mkdir -p "$APP"
cd "$APP"
git init -q
printf 'module example.com/demo-app\n\ngo 1.22\n' > go.mod
printf 'package main\n\nfunc main() {}\n' > main.go
commit_all "Start demo-app"

show syllabus project init --non-interactive
echo
show cat .syllabus/config.yaml
show_files "$APP" demo-app
