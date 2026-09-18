# Review the migration stack

PRs #17-#32 are merged into `go-migration`. The remaining implementation batch is
ready for review as a stack. Human approval of these new PRs is still required.

| PR  | Capability                      | Branch                         | Review base                    | Walkthrough       |
| --- | ------------------------------- | ------------------------------ | ------------------------------ | ----------------- |
| #33 | Sync complete projects          | codex/go-project-sync          | go-migration                   | /walkthrough/pr33 |
| #34 | Native project CLI              | codex/go-cli                   | codex/go-project-sync          | /walkthrough/pr34 |
| #35 | Project authoring               | codex/go-project-authoring     | codex/go-cli                   | /walkthrough/pr35 |
| #36 | Library authoring/check         | codex/go-library-authoring     | codex/go-project-authoring     | /walkthrough/pr36 |
| #37 | Interactive prompts             | codex/go-interactive-authoring | codex/go-library-authoring     | /walkthrough/pr37 |
| #38 | Native artifact candidates      | codex/go-native-artifacts      | codex/go-interactive-authoring | /walkthrough/pr38 |
| #39 | Complete CLI pilot and evidence | codex/go-native-acceptance     | codex/go-native-artifacts      | /walkthrough/pr39 |

Build `cmd/code-rules` and `cmd/rules-lab` into the same output directory. For PR38,
place committed candidate artifacts in `native-artifacts` beside the lab. Run
`rules-lab -serve -port 4391`. Every walkthrough executes native Go in isolated
fixtures and covers its PR's new capability. CLI pages show real command output,
errors, exit status, and project files; PR37 uses actual pseudo-terminals.

Fix defects on the earliest owning branch and propagate changes through dependents.
Refresh affected tests and reviews. After an approved predecessor merges, retarget
its successor to `go-migration` and verify the diff stays focused.

Every PR receives the full [independent validation prompt](independent-validation.md)
and a Devin review. CodeRabbit exhausted its included review quota after #28.
Check live feedback and nonempty green CI before an authorized merge. A ready PR,
completed review, or passing test is not human merge approval.

See [native acceptance](native-acceptance.md) for evidence, approved differences,
and remaining release decisions. The TypeScript npm entrypoint stays intact.
