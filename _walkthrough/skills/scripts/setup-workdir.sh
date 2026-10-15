#!/bin/bash
# Create the work folder and env.sh, which every later step sources. env.sh keeps Git and Go away from your
# home folder and your global Git configuration, and serves the demo repositories over a local SSH stand-in.
set -eo pipefail

# Runbooks starts the first blocks in the runbook's folder, where it keeps the Files panel's Generated files. Keep that
# folder, so show_files can remove stale copies there; a rerun after an earlier block changed directory keeps the one
# recorded before.
WORK="{{ .inputs.WorkDir }}"
RUNBOOK_DIR=""
if [ -f "$PWD/runbook.mdx" ]; then
  RUNBOOK_DIR="$PWD"
elif [ -f "$WORK/env.sh" ]; then
  RUNBOOK_DIR=$(sed -n 's/^export RUNBOOK_DIR="\(.*\)"$/\1/p' "$WORK/env.sh")
fi
case "$WORK" in
  /*) ;;
  *) echo "WorkDir must be an absolute path, such as /tmp/syllabus-skills-walkthrough."; exit 1 ;;
esac
if [ -e "$WORK" ] && [ ! -f "$WORK/.syllabus-walkthrough" ]; then
  echo "$WORK already exists and isn't a walkthrough work folder. Choose another WorkDir."
  exit 1
fi
mkdir -p "$WORK/bin" "$WORK/remotes"
touch "$WORK/.syllabus-walkthrough"

# Git configuration for the walkthrough only. GIT_CONFIG_GLOBAL in env.sh points here, so your ~/.gitconfig, its
# signing settings, and its credential helpers are never read or changed.
cat > "$WORK/gitconfig" <<'GITCONFIG'
[init]
	defaultBranch = main
[commit]
	gpgsign = false
[tag]
	gpgsign = false
[advice]
	detachedHead = false
GITCONFIG

# Syllabus fetches sources only over HTTPS or SSH. This stand-in for ssh serves git@demo.invalid:<name> from
# $WORK/remotes/<name>, so the demo library and skills repository behave like hosted repositories without a
# network or an SSH server. The CLI's own tests use the same technique.
cat > "$WORK/bin/demo-ssh" <<'SSH'
#!/bin/sh
for last; do :; done
service=${last%% *}
name=$(printf '%s' "${last#* }" | tr -d "'")
repo="$(cd "$(dirname "$0")/.." && pwd)/remotes/$name"
case "$service" in
  git-receive-pack) exec git receive-pack "$repo" ;;
  *) exec git -c uploadpack.allowFilter=true upload-pack "$repo" ;;
esac
SSH
chmod +x "$WORK/bin/demo-ssh"

cat > "$WORK/env.sh" <<ENV
# Sourced by every step of the walkthrough.
export WORK="$WORK"
export RUNBOOK_DIR="$RUNBOOK_DIR"
ENV
cat >> "$WORK/env.sh" <<'ENV'
export CR="$WORK/bin/syllabus"
export LIB="$WORK/acme-library" TEAM="$WORK/team-skills" APP="$WORK/demo-app"

# Git: ignore system and global configuration, and commit as a demo author.
export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL="$WORK/gitconfig"
export GIT_AUTHOR_NAME="Walkthrough" GIT_AUTHOR_EMAIL="walkthrough@example.invalid"
export GIT_COMMITTER_NAME="Walkthrough" GIT_COMMITTER_EMAIL="walkthrough@example.invalid"
export GIT_SSH_COMMAND="$WORK/bin/demo-ssh" GIT_SSH_VARIANT=ssh
# Runbooks runs blocks in a terminal, so keep Git from opening a pager that waits for a key.
export GIT_PAGER=cat PAGER=cat

# Go: use your own module and build caches, so a build reuses what you've already downloaded, and fetch the Go
# version syllabus needs when yours is older.
export GOTOOLCHAIN=auto

# syllabus runs the CLI this walkthrough built, so the commands below read like the ones you'd type.
syllabus() {
  if [ ! -x "$CR" ]; then
    echo "The CLI isn't built yet. Run \"Clone and build syllabus\" and wait for \"The CLI is built\" first." >&2
    return 1
  fi
  "$CR" "$@"
}

# quote_command prints its arguments as a command you could paste into a shell.
quote_command() {
  local out="" arg
  for arg in "$@"; do
    case "$arg" in
      "" | *[!A-Za-z0-9_./:=@%+,-]*) arg="'$(printf '%s' "$arg" | sed "s/'/'\\\\''/g")'" ;;
    esac
    out="$out${out:+ }$arg"
  done
  printf '\n$ %s\n' "$out"
}

# show_files copies the files of the repository at $1, without .git, to the Files panel's Generated files, under the
# folder name $2. Each block that changes a repository calls it last. Runbooks adds captured files to Generated files
# without removing any, so it first deletes the earlier copy there, or files the repository no longer has would stay.
show_files() {
  [ -n "${GENERATED_FILES:-}" ] || return 0
  if [ -n "$RUNBOOK_DIR" ] && [ -d "$RUNBOOK_DIR/generated/$2" ]; then
    rm -rf "${RUNBOOK_DIR:?}/generated/${2:?}"
  fi
  mkdir -p "$GENERATED_FILES/$2"
  tar -C "$1" --exclude=.git -cf - . | tar -C "$GENERATED_FILES/$2" -xf -
}

# show prints a command, then runs it.
show() { quote_command "$@"; "$@"; }

# expect_failure prints a command, runs it, and fails unless the command fails.
expect_failure() {
  quote_command "$@"
  if "$@"; then
    echo "Expected that command to fail, but it succeeded."
    return 1
  fi
  echo "(It failed, as expected.)"
}

# commit_all commits every change in the current repository.
commit_all() { git add -A && git commit -q -m "$1"; }

# pass and fail report the result of a check; fail stops the script.
pass() { echo "OK: $*"; }
fail() { echo "FAILED: $*"; exit 1; }

# require_text FILE TEXT fails unless FILE contains TEXT.
require_text() { grep -qF -- "$2" "$1" || fail "$1 doesn't contain: $2"; pass "$1 contains: $2"; }
ENV

echo "Work folder: $WORK"
ls -A "$WORK"
