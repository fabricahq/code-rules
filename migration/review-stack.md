# Review the migration stack

PRs #17-#27 have merged into `go-migration`. PRs #28-#32 are open for human review. Passing CI or automated review does not authorize merging them.

| PR | Capability | Branch | Review base | Walkthrough |
| --- | --- | --- | --- | --- |
| #28 | Vendor snapshots | codex/go-vendor-snapshots | go-migration | /walkthrough/pr28 |
| #29 | Recoverable project writes | codex/go-project-writes | codex/go-vendor-snapshots | /walkthrough/pr29 |
| #30 | Offline build and check | codex/go-offline-project | codex/go-project-writes | /walkthrough/pr30 |
| #31 | Verified Git revisions | codex/go-git-revisions | codex/go-offline-project | /walkthrough/pr31 |
| #32 | Complete Git library imports | codex/go-git-imports | codex/go-git-revisions | /walkthrough/pr32 |

Run `go run ./cmd/rules-lab -serve -port 4391` from the top branch. The root page links to all five walkthroughs. Each invokes native Go and covers only its PR's capability. Filesystem writes and Git repositories use isolated disposable fixtures.

Fix defects on the earliest owning branch and propagate changes through dependents. Re-run affected checks and refresh reviews. After a predecessor receives human approval and merges, retarget its successor to `go-migration` and verify the diff remains focused.

Every PR receives the [independent validation prompt](independent-validation.md), plus CodeRabbit or Devin review. CodeRabbit exhausted its included review quota after #28; subsequent PRs use Devin. Inspect live GitHub checks and review threads before merging.

The TypeScript CLI remains the production entry point. The Go executable is still a development lab. Next come sync orchestration, Cobra commands, project and library authoring, native packaging, and full migration acceptance. Estimate 5-8 further implementation PRs after this batch, followed by the separately approved integration PR to `main`.

The broad capability inventory remains pending until its full acceptance evidence is reconciled. Per-slice tests and reviews do not by themselves establish complete CLI parity or authorize TypeScript removal.
