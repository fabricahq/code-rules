# Security practices

We pin dependencies, review updates through Renovate PRs, and scan Go code for known vulnerabilities. Maintainers approve merges; the bot never merges updates automatically.

## Dependency pins

- Pin every external GitHub Action and reusable workflow to its full commit SHA. Keep the release version in a trailing comment for readability.
- Resolve each SHA from the upstream repository, including the commit behind an annotated tag. A version tag alone can move.
- Commit `go.mod` and `go.sum`. Keep Go's checksum verification enabled for public modules.
- Commit `bun.lock` and install website dependencies with `bun install --frozen-lockfile`. Version ranges in `package.json` do not authorize CI to regenerate the lockfile.
- Pin executable tools, including actionlint, Staticcheck, and govulncheck. Avoid `@latest` in CI and release commands.

Pins prevent unexpected changes to selected dependencies. Checksums verify downloaded content. Neither proves that the selected code is safe.

## Renovate update policy

[renovate.json](../renovate.json) owns the executable policy.

| Update | Handling |
| --- | --- |
| Routine patch and minor releases | Open reviewed PRs during Monday's midnight-to-6 a.m. window in `America/Los_Angeles`, after a seven-day release cooldown. |
| Major releases | Propose separate PRs. A maintainer reviews compatibility and migration requirements. |
| Security fixes | Propose PRs on the next bot run, without the weekly window or cooldown. Prioritize review, but require passing checks and manual merging. |
| Action changes | Update the full SHA and its version comment together. Review changes even when the version label stays the same. |

The seven-day cooldown applies to each proposed version, not the time since a package last changed. Missing release timestamps hold routine updates for investigation.
The Dependency Dashboard shows pending updates. Renovate's schedule limits when it can create routine updates; it does not guarantee an exact run time.
Renovate may refresh existing update branches outside that window.

Renovate manages Go modules, Bun workspaces and their lockfile, workflow actions, and supported runtime inputs.
A custom manager finds versioned `go run` tools in workflows and README commands so those pins receive update PRs too.
Renovate includes indirect Go requirements. We disable broad lockfile-maintenance runs; dependency PRs regenerate the affected lockfile through the package manager.
Review transitive changes in each lockfile diff: the cooldown does not establish the safety or age of every dependency a package manager resolves.

We constrain TypeScript below 6.1 because Astro check and typescript-eslint require the TypeScript 6 compiler API.
Revisit that constraint when both tools support a newer API. If the constraint blocks a security fix, resolve compatibility as part of the security PR.

## Vulnerability detection

[Dependency security](../.github/workflows/security.yml) runs `govulncheck` on pushes, pull requests, daily at 14:23 UTC, and manual dispatch.
The scan includes tests on Linux and macOS and uses the current Go vulnerability database. Findings and scan errors fail the job.
The release workflow repeats the scan against the exact release source before publication, including retries.
The [README](../README.md#validate-changes) owns the local command and pinned scanner version.

Scheduled scans run from the default branch after a maintainer merges the workflow. Maintainers must monitor failed runs and GitHub vulnerability alerts.
Scans identify known vulnerabilities in the code they analyze. Passing scans and tests cannot prove that a dependency is free of malicious code or undisclosed vulnerabilities.

Renovate consumes GitHub vulnerability alerts and enables OSV alerts for direct dependencies supported by Renovate.
The experimental OSV integration supplements transitive dependency alerts and govulncheck.
Website dependencies rely on those advisory feeds; govulncheck only analyzes Go code.

## Review and release access

Before merging an update, review its source, release notes, changed permissions or install scripts, and manifest and lockfile diffs.
Confirm all applicable CI checks passed on the current commit. A patch version is not evidence that an update is safe.
For security fixes, verify that the selected version addresses the advisory. Do not bypass failed checks to accelerate a merge.

Workflow permissions default to read-only. Pull request jobs do not receive publication credentials.
The release build compiles and tests the publisher without write access. Only the final publication job receives permission to write release data.
Actions in that job can access its job token; step-level environment variables do not isolate the token from other actions in the job.
SHA pins and minimal permissions protect against compromised actions as well as compromised publication code.

Dependency updates do not publish releases. A maintainer approves a release by merging its release-note PR, as described in [the release procedure](releasing.md).

## GitHub setup and activation

Repository configuration does not install the Renovate GitHub App or enforce branch protection.

1. Install the [Mend Renovate GitHub App](https://github.com/apps/renovate) for `fabricahq`, selecting only `code-rules`.
2. Keep the dependency graph and GitHub's **Dependabot alerts** enabled. Renovate reads these alerts; Dependabot does not need to create PRs.
3. Leave **Dependabot security updates** disabled and do not add a Dependabot version-update configuration. Renovate owns update PRs.
4. Give Renovate read access to vulnerability alerts and complete its onboarding after this configuration reaches `main`.
5. Verify the Dependency Dashboard lists Go modules, Bun dependencies, actions, runtimes, and all three Go tools. Investigate extraction errors before relying on automation.
6. Use main-branch protection or a ruleset to require PRs and the applicable CI checks. Grant Renovate no merge bypass. Review protection separately from the bot's `automerge: false` policy.

During setup on 2026-09-18, GitHub alerts were enabled and Dependabot security updates were disabled.
Renovate was not installed for the organization, and `main` had no branch protection or repository ruleset.
Those observations are a setup snapshot, not a guarantee of the live settings. Verify activation in GitHub before claiming that updates are running.

## Policy references

- [GitHub: secure use of actions](https://docs.github.com/en/actions/reference/security/secure-use#using-third-party-actions)
- [Go: module authentication](https://go.dev/ref/mod#authenticating)
- [Go: vulnerability management](https://go.dev/doc/security/vuln/)
- [Renovate: minimum release age](https://docs.renovatebot.com/configuration-options/#minimumreleaseage)
- [Renovate: vulnerability alerts](https://docs.renovatebot.com/configuration-options/#vulnerabilityalerts)
- [Renovate: OSV alerts and coverage limits](https://docs.renovatebot.com/configuration-options/#osvvulnerabilityalerts)

When changing this policy, validate `renovate.json` with Renovate's `renovate-config-validator --strict` command and run the README validation commands.
