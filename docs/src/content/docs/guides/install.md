---
title: "Install Code Rules"
description: "Install the packaged CLI, try a release candidate, and upgrade without changing project rules."
---

Code Rules packages a `code-rules` executable for **Node.js 22.18 or later on macOS and Linux**. Imports also require Git 2.30 or later and your normal Git credentials.
Bun is needed to develop and package this repository; users of the installed CLI need Node.js and Git.
Windows is not supported in the first release.

## Try an unpublished release candidate

No public npm release has shipped yet. From the Code Rules checkout:

```sh
bun install --frozen-lockfile
npm pack
```

`npm pack` builds the executable and includes the canonical authoring template.
Install the resulting tarball into an isolated prefix:

```sh
npm install --global --prefix /tmp/code-rules-cli ./fabricahq-code-rules-0.1.0-rc.1.tgz
/tmp/code-rules-cli/bin/code-rules --version
/tmp/code-rules-cli/bin/code-rules --help
```

Add `/tmp/code-rules-cli/bin` to your terminal's `PATH` to use `code-rules` directly.
The prefix only contains the tool. Project configuration and rule files live in the consuming repository's `.code-rules/` directory.
Follow [Set up a project](/guides/set-up-project/) or [Create a rule library](/guides/create-library/).

## Install a published release

Once a release is available, install the scoped package. The executable name remains `code-rules`:

```sh
npm install --global @fabricahq/code-rules
code-rules --version
```

Use an explicit package version to pin the tool for your team or CI.
Prereleases use the `next` npm tag; stable releases use `latest`.
Installing the package does not create project files, download rule libraries, or edit agent instructions.

## Upgrade the tool

Once stable releases are available:

```sh
npm install --global @fabricahq/code-rules@latest
code-rules --version
code-rules build
code-rules check
```

The package manager replaces the executable and its dependencies. It leaves authored rules and committed project files alone.
Build regenerates with the installed version, including that version in provenance. Review and commit those changes.
Run `sync` separately when you also want to resolve library versions again.
To return to an earlier tool version, install that exact npm version and regenerate with it.

The separate `code-rules update` command remains a follow-up. Use the package manager that owns your installation.
