# Native CLI runbook

Executable introduction to project and library authoring, read-only checks, and repair.

## Open

Install Go and [Gruntwork Runbooks](https://runbooks.gruntwork.io/intro/installation/), then run from this checkout:

```sh
bash runbooks/native-cli/open.sh
```

The launcher opens Runbooks at <http://localhost:4392/>. Tested with Runbooks `beta-v0.9.0`.

Choose **Prepare** first. It builds this checkout into a fresh temporary directory and prints the binary’s absolute path. Copy the printed terminal setup to try commands in a separate `playground/` directory. The output includes missing-project and invalid-option failures with their expected exit codes.

Then run the guided blocks in order. They use separate temporary project and library directories. The scripts do not install globally or fetch repositories. Re-run **Prepare** after changing Go code to build a fresh binary.

The stale-output check intentionally fails. Continue to **Repair**. Initialize and each file-changing step capture files in Runbooks' file viewer, including hidden `.code-rules` files. Each script prints its working directory and labels displayed file contents. CLI output is human-readable by default; the Check step also demonstrates `--json`. Those copies are snapshots from the last successful block, not live filesystem views.

Re-running Prepare starts a fresh workspace. Earlier temporary workspaces remain on disk for inspection. Stop the server with Ctrl+C when finished. The launcher and runbook print the temporary paths so you can remove them when no longer needed.

## Test

Pass the source checkout to Gruntwork's block-level test runner. Prepare builds the binary:

```sh
CODE_RULES_DEMO_SOURCE_DIR="$PWD" \
  runbooks test runbooks/native-cli --no-telemetry -v
```

Expect one test with all 12 steps passing, including `command:detect: fail` as an expected outcome. Check the summary: Runbooks beta-v0.9.0 can return exit 0 even when configuration loading runs zero tests.

The test uses the same scripts as the browser. Production CLI behavior is covered separately by Go tests and the real Git lifecycle tests.
