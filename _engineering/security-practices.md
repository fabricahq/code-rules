# Security practices

Users install Code Rules to download rule libraries into their repositories. Those files become instructions for their coding agents.
**Assume any agent may read imported rules on every invocation.** An unauthorized change could repeatedly influence generated code, tool use, and access to the user's repository.
Security is therefore a core product requirement.

## What we must protect

1. **Rule-file integrity.** Preserve the selected library revision's original rule bytes. Detect unauthorized changes to retained files and agent-readable output.
   Generated output may intentionally transform Markdown or apply explicit project overrides; those changes must remain attributable to the selected rules and the user's configuration.
2. **Library-source authenticity.** Fetch the repository and revision the user selected. Never silently substitute another library or revision, or present an unverified origin as authenticated.
   Users must be able to understand which source and commit supplied their rules.
3. **Our repository and release integrity.** Protect our source, repository access, dependencies, workflows, and published executables against unauthorized changes.
   A compromised importer or release could undermine both rule integrity and source verification across many consuming repositories.

These are requirements to verify, not a claim that an audit has established every guarantee.
A checksum proves consistency with its reference record, not authenticity if an attacker can replace both the content and the record.
Even authentic, unchanged rules can contain harmful instructions. Users must deliberately choose which publishers and rule changes they trust.

## Security audits

Track evidence, confirmed findings, and remaining limitations in these focused audits:

- [CR-3: Rule-file integrity](https://linear.app/ohmygoshjosh/issue/CR-3/audit-rule-file-integrity-from-import-through-agent-readable-output), from imported bytes through the output agents read.
- [CR-4: Library-source authenticity](https://linear.app/ohmygoshjosh/issue/CR-4/audit-rule-library-source-authenticity-and-revision-resolution), including repository identity, Git routing, revision selection, and provenance.
- [CR-5: Repository and release security](https://linear.app/ohmygoshjosh/issue/CR-5/audit-repository-and-release-pipeline-against-supply-chain-compromise), including live access controls and source-to-artifact verification.

The audits must exercise the real CLI or workflow boundaries in controlled fixtures and record the exact commit reviewed.
Dependency scans alone do not establish rule integrity or library authenticity. Treat open audits as unverified work, not completed protection.

## Protecting this repository

We pin dependencies, review updates through Renovate PRs, and scan Go code for known vulnerabilities. Maintainers approve merges; the bot never merges updates automatically.
The controls below reduce the risk of compromising Code Rules itself. Live GitHub settings must enforce the access and review policies alongside the repository configuration.

### Dependency pins

- Pin every external GitHub Action and reusable workflow to its full commit SHA. Keep the release version in a trailing comment for readability.
- Resolve each SHA from the upstream repository, including the commit behind an annotated tag. A version tag alone can move.
- Commit `go.mod` and `go.sum`. Keep Go's checksum verification enabled for public modules.
- Commit `bun.lock` and install website dependencies with `bun install --frozen-lockfile`. Version ranges in `package.json` do not authorize CI to regenerate the lockfile.
- Pin executable tools, including actionlint, Staticcheck, and govulncheck. Avoid `@latest` in CI and release commands.

Pins prevent unexpected changes to selected dependencies. Checksums verify downloaded content. Neither proves that the selected code is safe.

### Renovate update policy

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
A custom manager finds versioned `go run` tools in workflows and CONTRIBUTING.md commands so those pins receive update PRs too.
Renovate includes indirect Go requirements. We disable broad lockfile-maintenance runs; dependency PRs regenerate the affected lockfile through the package manager.
Review transitive changes in each lockfile diff: the cooldown does not establish the safety or age of every dependency a package manager resolves.

We constrain TypeScript below 6.1 because Astro check and typescript-eslint require the TypeScript 6 compiler API.
Revisit that constraint when both tools support a newer API. If the constraint blocks a security fix, resolve compatibility as part of the security PR.

### Vulnerability detection

[Dependency security](../.github/workflows/security.yml) runs `govulncheck` on pushes, pull requests, daily at 14:23 UTC, and manual dispatch.
The scan includes tests on Linux and macOS and uses the current Go vulnerability database. Findings and scan errors fail the job.
The release workflow repeats the scan against the exact release source before publication, including retries.
[CONTRIBUTING.md](../CONTRIBUTING.md#validate-changes) owns the local command and pinned scanner version.

Scheduled scans run from the default branch after a maintainer merges the workflow. Maintainers must monitor failed runs and GitHub vulnerability alerts.
Scans identify known vulnerabilities in the code they analyze. Passing scans and tests cannot prove that a dependency is free of malicious code or undisclosed vulnerabilities.

Renovate consumes GitHub vulnerability alerts and enables OSV alerts for direct dependencies supported by Renovate.
The experimental OSV integration supplements transitive dependency alerts and govulncheck.
Website dependencies rely on those advisory feeds; govulncheck only analyzes Go code.

### Review and release access

Before merging an update, review its source, release notes, changed permissions or install scripts, and manifest and lockfile diffs.
Confirm all applicable CI checks passed on the current commit. A patch version is not evidence that an update is safe.
For security fixes, verify that the selected version addresses the advisory. Do not bypass failed checks to accelerate a merge.

Workflow permissions default to read-only. Pull request jobs do not receive publication credentials.
The release build compiles and tests the publisher without write access. Only the final publication job receives permission to write release data.
Actions in that job can access its job token; step-level environment variables do not isolate the token from other actions in the job.
SHA pins and minimal permissions protect against compromised actions as well as compromised publication code.

Dependency updates do not publish releases. A maintainer approves a release by merging its release-note PR, as described in [the release procedure](releasing.md).

### PR preview downloads

The packaging workflow builds PR code with read-only repository permissions. Successful builds do not establish that the code is safe.
The separate **CLI preview downloads** workflow runs from the default branch and only reads GitHub metadata and posts comments. It never checks out PR code or downloads or runs an artifact.

A preview receives automatic download links only if its PR author currently has repository write or admin permission and the PR branch belongs to this repository.
Forks, including maintainer-owned forks, and other contributors require explicit approval:

1. Review the current PR commit, including source, dependencies, tests, and build workflow changes, for isolated testing.
2. Find its successful **Package CLI binaries** run. Copy the numeric run ID from its URL and the PR's full 40-character commit SHA.
3. In Actions, select **CLI preview downloads**, then **Run workflow** from the default branch. Enter that run ID and commit SHA.

The workflow checks the approver's current write access, the build's repository, workflow, event, completion and success, and the PR's current head.
A new commit requires a new approval for external previews. Approval advertises a particular preview; it does not approve a release or certify that the code is safe.
Artifacts remain accessible in Actions before promotion; this gate controls the bot's download recommendation, not access to the build output.

Preview commands print a warning with the packaged source commit to stderr, preserving stdout for command results and JSON.
The warning and version string are self-reported metadata, not authentication. Preview executables are not publisher-signed or attested by Code Rules.
Run them only in disposable environments without credentials or private files. GitHub artifact digests can detect changed bytes against a trusted reference, but do not establish benign behavior.
Official release publication remains a separate reviewed process.

### GitHub setup and activation

Repository configuration does not install the Renovate GitHub App or enforce branch protection.

1. Grant the [Mend Renovate GitHub App](https://github.com/apps/renovate) access to `fabricahq/code-rules`. Limit its installation to repositories the organization intends it to manage.
2. Keep the dependency graph and GitHub's **Dependabot alerts** enabled. Renovate reads these alerts; Dependabot does not need to create PRs.
3. Leave **Dependabot security updates** disabled and do not add a Dependabot version-update configuration. Renovate owns update PRs.
4. Give Renovate read access to vulnerability alerts and complete its onboarding after this configuration reaches `main`.
5. Verify the Dependency Dashboard lists Go modules, Bun dependencies, actions, runtimes, and all three Go tools. Investigate extraction errors before relying on automation.
6. Use main-branch protection or a ruleset to require PRs and the applicable CI checks. Grant Renovate no merge bypass. Review protection separately from the bot's `automerge: false` policy.

Verify live installation permissions, alert settings, and branch protections in GitHub before claiming they are active.
Record dated evidence in the repository security audit instead of relying on a setup snapshot in this document.

## Policy references

- [GitHub: secure use of actions](https://docs.github.com/en/actions/reference/security/secure-use#using-third-party-actions)
- [Go: module authentication](https://go.dev/ref/mod#authenticating)
- [Go: vulnerability management](https://go.dev/doc/security/vuln/)
- [Renovate: minimum release age](https://docs.renovatebot.com/configuration-options/#minimumreleaseage)
- [Renovate: vulnerability alerts](https://docs.renovatebot.com/configuration-options/#vulnerabilityalerts)
- [Renovate: OSV alerts and coverage limits](https://docs.renovatebot.com/configuration-options/#osvvulnerabilityalerts)

When changing this policy, validate `renovate.json` with Renovate's `renovate-config-validator --strict` command and run the [validation commands](../CONTRIBUTING.md#validate-changes).
