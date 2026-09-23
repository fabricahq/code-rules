---
title: "Install Code Rules"
description: "Build the Go CLI from source, then upgrade or roll back deliberately."
---

Code Rules is a standalone Go executable for macOS and Linux on amd64 and arm64. Running it requires neither Node.js nor Bun. Sync also needs Git and your existing repository credentials. Windows is not supported.

Public release publication remains separate work. Build the CLI from source using the instructions below.

## Build from source

Install the Go version declared in the repository's `go.mod`, then run from the Code Rules checkout:

```sh
go build -o ./dist/code-rules ./cmd/code-rules
./dist/code-rules --version
./dist/code-rules --help
```

Use the executable's absolute path, or add its directory to your `PATH`. A source build reports a development version unless built with an explicit version. Installing the executable does not initialize projects or fetch libraries.

Follow [Set up a project](/guides/set-up-project/) or [Create a rule library](/guides/create-library/).

## Upgrade or roll back

Keep executables in separate versioned directories. Test the new executable against a copy of your project before selecting it on `PATH`; retain the old directory for rollback.

After selecting a new version, run from your project:

```sh
code-rules project build
code-rules project check
```

Build automatically refreshes the managed project README and regenerates guidance and provenance. It preserves configuration and local rules and refuses to overwrite manual edits to the guide. Review the changes before committing. Sync also refreshes the guide; use it when you want to resolve remote library revisions again.

To roll back, select the previous executable and review regeneration with that version. The `code-rules update` command is not implemented.


## Versioning

Code Rules uses semantic versions: `MAJOR.MINOR.PATCH`. Before `1.0.0`, breaking changes require a new minor version. From `1.0.0`, they require a new major version.

Changes to the managed project README format are breaking changes. Release notes describe the format change. Running build or sync applies the template bundled in the selected CLI version automatically when the existing guide has no manual edits.

Keep project-specific notes in a separate file. If a guide was edited, preserve those notes and move the guide aside before rerunning build or sync.
