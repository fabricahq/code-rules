# Go migration: independent validation

Independently validate the current Go-migration PR in https://github.com/fabricahq/code-rules. Review correctness, compatibility, safety, testability, maintainability, and simplicity, and explicitly enforce applicable engineering rules from https://github.com/josh-padnick/code-rules.

You are the validator, not the implementation agent. Inspect source and run checks yourself. Treat the implementer's report as a set of claims to verify. Keep the submitted source unchanged; create isolated worktrees or temporary reproduction files as needed. Deliver findings to the human rather than committing fixes, posting external reviews, or merging the PR.

## Fix the review scope

Use the migration PR supplied with this prompt. If none is supplied, locate the open migration PR; ask for the target if there are multiple plausible candidates. Record exact base/head commits and the pinned TypeScript reference. Review the full diff, relevant surrounding code and contracts, and applicable project instructions. Check whether the head changes during review and qualify or refresh affected evidence before issuing a verdict.

Identify what this slice promises and what remains intentionally unimplemented. A deliberately incomplete Go candidate is not a defect by itself. Verify that it cannot masquerade as a complete replacement. For the initial harness PR, review the inventory and measurement system rather than demanding a finished Go CLI.

## Load and apply the engineering rules

1. Access https://github.com/josh-padnick/code-rules with available authorized GitHub access. Use a clean checkout or immutable files at a recorded commit. A local checkout is acceptable after verifying its remote, revision, and relevant contents. If access fails, continue correctness review but mark rules validation incomplete; do not substitute remembered or generic rules.
2. Read the corpus README and discover its current groups. This corpus has its own layout and is not necessarily a Code Rules-format library. Select technology groups for changed behavior and relevant practice groups, even when their examples use a different language or no test files changed. Start with Go for Go changes, TypeScript for harness/runtime changes, and code-design practices for function/module design. Discover additional applicable groups rather than inventing them.
3. Read every relevant or plausibly relevant rule completely, including exceptions. A `whenToRead` cue selects material to inspect; it is not itself proof of applicability or a violation. Complete truncated reads and reload needed rules after compaction. Read implementation guidance to understand the obligation and validation guidance to evaluate evidence.
4. Apply domain overlays only when the rule's stated domain actually matches. The product's GitHub organization alone does not make Fabrica application's internal architecture contracts applicable. Resolve project-specific exceptions explicitly. Surface incompatible obligations for human judgment rather than silently ignoring one.
5. Record the corpus commit and selected rule paths. For each rule-backed finding, cite the exact rule path/revision, explain applicability and exceptions, and connect its requirement to concrete code and consequences. Set finding severity from actual impact and likelihood, not the rule's impact label. Keep private rule bodies out of public artifacts; use references and minimal explanation.

## Verify the migration contract

Run the relevant commands from the actual checkout. Inspect required checks and test implementation; do not rely solely on CI status. Validate the following wherever the slice touches them:

- **Behavior:** independently reproduce promised success, failure, boundary, and degenerate cases. Compare both executables in separate workspaces at known revisions. Identify reference bugs rather than demanding they be copied.
- **Outputs:** verify identities, group selection, local additions, replacements/exclusions, deterministic ordering, links/assets, license and notice bytes, provenance, and generated layout. Inspect both missing output and unexpected extra output.
- **Filesystem safety:** confirm read-only behavior, preservation of user edits, path containment, symlink handling, stale-file cleanup, partial-failure behavior, locks, cancellation, rollback, and recovery as applicable. Keep destructive probes confined to disposable workspaces.
- **Interfaces:** check exit codes, contextual help, prompts, meaningful errors, version-range semantics, Git behavior, and the absence of a hidden Node/Bun runtime dependency in the Go candidate.
- **Measurement integrity:** inspect normalization, fixture completeness, required-case changes, skipped checks, baseline refreshes, and approved differences. Try a targeted counterexample or intentionally altered result to establish that the harness detects a relevant defect. A green harness that weakens the contract is a finding.

Separate check failures caused by the change from environment limitations and pre-existing defects. A missing prerequisite is unverified evidence, not a passing check. Re-run affected checks after any new head rather than carrying approval forward automatically.

## Evaluate engineering quality

Assess whether the Go design expresses the operation clearly, has cohesive boundaries and explicit ownership, and fits the repository's scale. Look for accidental complexity, unnecessary abstractions, hidden state, broad dependencies, fragile error handling, and literal TypeScript translations that fight Go conventions. Explain concrete maintenance or failure consequences instead of proposing stylistic rewrites.

Assess tests for realistic behavior coverage, useful failure signals, deterministic execution, and boundary cases. Check whether important paths are reachable without elaborate mocks. Prefer tests that survive harmless refactoring. Distinguish a testability defect in the design from a merely missing test.

Keep findings grounded: name the trigger, affected code, observed or well-supported consequence, and smallest useful correction. General correctness or maintainability findings need no invented rule citation. Separate optional improvements from defects requiring changes.

## Deliver the review

Provide a concise report containing:

1. **Verdict:** ready for human review, changes required, or validation incomplete. This is advisory and does not authorize merge.
2. **Scope:** PR and base/head commits, TypeScript reference, and rules-corpus revision.
3. **Findings:** severity, precise file/line, trigger, consequence, evidence or reproduction, and recommended correction. Include rule citations for rule-backed findings. State explicitly when there are no actionable findings.
4. **Evidence:** checks you executed and their results, compared capabilities, and reproduction artifacts. Distinguish your runs from inherited CI evidence.
5. **Rules coverage:** selected rules with brief applicability, material exceptions, and gaps. Avoid claiming exhaustive compliance beyond what you inspected.
6. **Human decisions:** contract changes, proposed exceptions, design tradeoffs, and residual risks that need judgment.

On re-review, fix the new head, verify previous findings are resolved, and inspect the intervening diff for new problems. Preserve independence: suggest durable lessons for the human and implementer to review rather than modifying their acceptance criteria or feedback file yourself.
