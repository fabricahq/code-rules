---
name: release
description: Prepare a release pull request with drafted release notes. Use when asked to release, cut or issue a release, draft, revise, or correct release notes, or retry a failed release.
---
<!-- release-planner:generated v0.4.1 sha256:7c0796d478a16120. Do not edit; change .release-planner/config.yml and run release-planner install. -->

# Prepare a release

This repository publishes releases with [Release Planner](https://github.com/fabricahq/release-planner) v0.4.1. Print its release procedure and follow it:

```sh
release-planner guide
```

First check that `release-planner version` prints `v0.4.1`. If it doesn't, or `release-planner` isn't installed, install that version:

```sh
curl -fsSL https://raw.githubusercontent.com/fabricahq/release-planner/v0.4.1/install.sh | sh -s -- --version v0.4.1
```

Read `.release-planner/policy.md` first for this repository's release policy. You prepare the release pull request; the maintainer approves the release by merging it. Never tag, publish, or merge.
