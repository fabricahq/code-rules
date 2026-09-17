# Review the migration stack

PRs 17-26 have merged into `go-migration` after user approval, resolved feedback, final review, and passing checks. PR #27 remains open for human review. Passing CI or automated reviews does not authorize merging it.

| PR | Capability | Branch | Review base | Walkthrough |
| --- | --- | --- | --- | --- |
| #22 (merged) | Library assets | codex/go-library-assets | go-migration | /walkthrough/pr22 |
| #23 (merged) | Effective rules | codex/go-rule-resolution | go-migration | /walkthrough/pr23 |
| #24 (merged) | Standalone rendering | codex/go-rule-rendering | go-migration | /walkthrough/pr24 |
| #25 (merged) | Discovery indexes | codex/go-discovery-indexes | go-migration | /walkthrough/pr25 |
| #26 (merged) | Licensed output | codex/go-licensed-output | go-migration | /walkthrough/pr26 |
| #27 | Full-rule groups | codex/go-inline-groups | go-migration | /walkthrough/pr27 |

Run `go run ./cmd/rules-lab -serve -port 4391` from the top branch. The root page links to every new walkthrough, with earlier walkthroughs under a separate disclosure. Each page invokes real Go and shows only its PR's capability. Libraries use temporary fixtures; generated output stays in memory.

Fix defects on the earliest owning branch and merge forward through dependents. Re-run affected checks and refresh reviews after changes. Once a predecessor receives human approval and merges, retarget its successor to `go-migration` and verify the diff remains focused.

Every PR receives the independent validation prompt in [independent-validation.md](independent-validation.md), plus CodeRabbit or Devin review. Source snapshot authenticity, persistence, and the production CLI remain later work.
