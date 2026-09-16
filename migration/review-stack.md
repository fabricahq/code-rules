# Review PRs 22-26 together

PRs 17-21 have merged into `go-migration`. The user authorized preparing the following five dependent PRs together. They remain open for human review. Passing CI or automated reviews does not authorize merging these five.

| PR | Capability | Branch | Review base | Walkthrough |
| --- | --- | --- | --- | --- |
| #22 | Library assets | codex/go-library-assets | go-migration | /walkthrough/pr22 |
| #23 | Effective rules | codex/go-rule-resolution | codex/go-library-assets | /walkthrough/pr23 |
| #24 | Standalone rendering | codex/go-rule-rendering | codex/go-rule-resolution | /walkthrough/pr24 |
| #25 | Discovery indexes | codex/go-discovery-indexes | codex/go-rule-rendering | /walkthrough/pr25 |
| #26 | Licensed output | codex/go-licensed-output | codex/go-discovery-indexes | /walkthrough/pr26 |

Run `go run ./cmd/rules-lab -serve -port 4391` from the top branch. The root page links to every new walkthrough, with earlier walkthroughs under a separate disclosure. Each page invokes real Go and shows only its PR's capability. Libraries use temporary fixtures; generated output stays in memory.

Fix defects on the earliest owning branch and merge forward through dependents. Re-run affected checks and refresh reviews after changes. Once a predecessor receives human approval and merges, retarget its successor to `go-migration` and verify the diff remains focused.

Every PR receives the independent validation prompt in [independent-validation.md](independent-validation.md), plus CodeRabbit or Devin review. Source snapshot authenticity, persistence, inline combined group delivery, and the production CLI remain later work.
