# Native CLI runbook

Supplementary, executable CLI review for PRs 34, 35, 36, and 39. The function and scenario labs remain on port 4391.

## Open

Install Go and [Gruntwork Runbooks](https://runbooks.gruntwork.io/intro/installation/), then run from this checkout:

```sh
bash runbooks/native-cli/open.sh
```

The launcher builds this checkout's native CLI into a temporary directory and opens Runbooks at <http://localhost:4392/>. Tested with Runbooks `beta-v0.9.0`.

Choose **Prepare** first, then run blocks in order. The scripts create a fresh temporary project and library; they print that path for inspection. They do not install globally or fetch repositories. Each launch builds a fresh binary; restart the launcher after changing Go code.

The stale-output check intentionally fails. Continue to **Repair**. Build, Repair, and Validate capture files in Runbooks' file viewer. Those copies are snapshots from the last successful block, not live filesystem views.

Re-running Prepare starts a fresh workspace. Earlier temporary workspaces remain on disk for inspection. Stop the server with Ctrl+C when finished. The launcher and runbook print the temporary paths so you can remove them when no longer needed.

## Test

Build a binary, then run Gruntwork's block-level test runner:

```sh
go build -o /tmp/code-rules-runbook-cli ./cmd/code-rules
CODE_RULES_DEMO_BINARY=/tmp/code-rules-runbook-cli \
  runbooks test runbooks/native-cli --no-telemetry -v
```

Expect one test with all 12 steps passing, including `command:detect: fail` as an expected outcome. Check the summary: Runbooks beta-v0.9.0 can return exit 0 even when configuration loading runs zero tests.

The test uses the same scripts as the browser. Production CLI behavior is covered separately by Go tests and the PR39 integration pilots.
