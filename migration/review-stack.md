# Review PRs 17-21 together

The user authorized preparing this dependent series before human review. Each PR is ready, each receives an independent bot review, and none is approved to merge merely because CI passes.

| PR | Branch | Review base | Walkthrough |
| --- | --- | --- | --- |
| #17 | codex/go-version-ranges | go-migration | /walkthrough/pr17 |
| #18 | codex/go-configuration | codex/go-version-ranges | /walkthrough/pr18 |
| #19 | codex/go-library-manifest | codex/go-configuration | /walkthrough/pr19 |
| #20 | codex/go-version-selection | codex/go-library-manifest | /walkthrough/pr20 |
| #21 | codex/go-local-library | codex/go-version-selection | /walkthrough/pr21 |

Run `go run ./cmd/rules-lab -serve -port 4391` from the top branch. The root page is the review collection. Later implementations retain earlier walkthroughs and API operations. The new filesystem lab uses only temporary fixtures.

Fix a defect on the earliest owning branch, then merge that branch forward through its dependents. Use merge commits while the stack is under review so parent fixes disappear from child diffs; do not leave separately cherry-picked parent changes in a child diff. Re-run affected checks and request fresh reviews after changed heads. After an approved predecessor merges, retarget its successor to `go-migration` and verify the diff still contains only that successor's work.

Scope documents: [constraints](version-constraints.md), [configuration](configuration.md), [license declarations](licenses.md), [release selection](version-selection.md), and [local catalog](local-library.md).
