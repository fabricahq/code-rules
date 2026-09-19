# Fabrica Homebrew tap

Install Code Rules on macOS or Linux with Homebrew:

```sh
brew install fabricahq/tap/code-rules
brew upgrade code-rules
brew uninstall code-rules
```

The formula installs the official executable for your operating system and processor, verifies the release archive's SHA-256, and retains its MIT license. Go, Node.js, and Bun are not required. Git is needed for `code-rules sync`.

The **Update Code Rules** workflow checks for a published stable release hourly, or when run manually. GitHub can delay scheduled runs. Until the first stable release is published and the workflow succeeds, the formula is unavailable. Prereleases and drafts are never selected; failed updates leave the existing formula intact.

To prepare an update locally:

```sh
python3 scripts/update_code_rules.py
```

Review `Formula/code-rules.rb`, then commit it. The updater refuses downgrades and changed checksums at the same version. Formula downloads come directly from `fabricahq/code-rules` release assets.

This repository can hold other Fabrica tools under `Formula/`. The Code Rules updater changes only `Formula/code-rules.rb`.
