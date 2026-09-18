---
title: "Set up a project"
description: "Start with local rules, then add shared libraries without moving your authored files."
---

Run these commands from the repository where you want to use rules.
The examples assume `code-rules` is installed and available on your PATH. See [Install Code Rules](/guides/install/) if you need the executable.

## Start with local rules

```sh
code-rules project init
code-rules project add group practices/testing
code-rules project add rule practices/testing/retry-budget
```

`init` creates `.code-rules/config.json` with no imported sources and `.code-rules/local/README.md`.
No imported library or existing Git repository is required. Repeating `init` preserves valid configuration and local rules and refreshes the managed project README. It refuses to overwrite manually edited guides; keep project notes in a separate file.
Human output starts with a setup confirmation and example commands for project-only rules or a shared library. If setup is already current, it reports that no files changed. Use `--json` for the exact changed file paths.

