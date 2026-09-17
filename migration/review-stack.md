# Review the migration stack

PRs #17-#32 have merged into `go-migration`. The user authorized preparing all remaining slices for review; passing CI or automated review does not authorize merging those new PRs.

| PR  | Capability             | Branch                | Review base           | Walkthrough       |
| --- | ---------------------- | --------------------- | --------------------- | ----------------- |
| #33 | Sync complete projects | codex/go-project-sync | go-migration          | /walkthrough/pr33 |
| #34 | Native project CLI     | codex/go-cli          | codex/go-project-sync | /walkthrough/pr34 |

| #35 | Project authoring | codex/go-project-authoring | codex/go-cli | /walkthrough/pr35 |

Build `cmd/code-rules` and `cmd/rules-lab` into the same output directory, then run `dist/rules-lab -serve -port 4391` from the top branch. The root page links to available walkthroughs. Each invokes native Go and covers only its PR's capability. Filesystem writes and Git repositories use isolated disposable fixtures.

Fix defects on the earliest owning branch and propagate changes through dependents. Re-run affected checks and refresh reviews. After a predecessor receives human approval and merges, retarget its successor to `go-migration` and verify the diff remains focused.

Every PR receives the [independent validation prompt](independent-validation.md), plus CodeRabbit or Devin review. CodeRabbit exhausted its included review quota after #28; subsequent PRs use Devin. Inspect live GitHub checks and review threads before merging.

## Remaining implementation batch

1. Sync orchestration (#33).
2. Cobra CLI commands for sync, build, and check.
3. Project initialization, source configuration, and local authoring.
4. Library initialization, authoring, and validation.
5. Interactive prompts and non-interactive behavior.
6. Native packaging and installation checks.
7. Full migration acceptance, documentation, and capability evidence.

Each PR gets a walkthrough covering only its new behavior. CLI walkthroughs will display arguments, standard output, standard error, exit status, and filesystem changes from isolated executions. Existing walkthrough URLs remain available.

The TypeScript CLI remains the production entry point until the native packaging slice is approved. The broad capability inventory remains pending until full acceptance evidence is reconciled. The final integration into `main`, release publication, and removal of the TypeScript implementation require separate approval.
