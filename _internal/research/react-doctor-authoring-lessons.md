# Rule-authoring lessons from React Doctor

Research date: 2026-09-14. Scope: transferable authoring and maintenance practices, not importing its rule text or implementing enforcement in Code Rules.

Evidence uses public first-party GitHub documents and previously fetched PR bodies. Registry/source artifacts were pinned to `922616f08db7d48463b1495e4dbdb699fc3d7c34` (2026-09-13). The authoring guide was read from GitHub's indexed `main` page; an exact pinned fetch was unavailable. No claim below requires reproducing React Doctor's restricted rule corpus.

## 1. Teach the boundary with a valid near-miss, not only a bad/fixed pair

React Doctor's authoring guide asks authors to establish a mechanism, inspect real code, and test similar-looking valid patterns. Its reducer example distinguishes mutation followed by returning the original object from a no-op return and a clone-first update. It also recommends checking suspicious-looking real applications before deciding the rule's scope. [Authoring guide](https://github.com/millionco/react-doctor/blob/main/docs/HOW_TO_WRITE_A_RULE.md#define-the-rule)

**Transfer to Code Rules:** Where a rule invites overapplication, include a realistic valid counterexample and explain the decisive difference. This fits the existing flexible body. For the meaningful-steps rule, show an operation where extraction helps and a short cohesive function that should stay intact. Do not mandate three examples for every simple rule.

## 2. Distinguish an exception from insufficient evidence

The same guide separates detector limitations from actual valid behavior. An imported reducer is outside the first detector's analysis scope; that does not establish that its behavior is safe. Nested mutation with a fresh top-level return is treated as a different defect, not a universal endorsement. [Authoring guide](https://github.com/millionco/react-doctor/blob/main/docs/HOW_TO_WRITE_A_RULE.md#v1-scope)

UI-library detectors likewise abstain when a title or accessible name could be supplied through unresolved child components or render props. [PR #1654](https://github.com/millionco/react-doctor/pull/1654)

Their configuration docs also restrict unused-export, unused-dependency, and circular-dependency checks to full scans because partial scopes cannot prove whole-project reachability. [Configuration guide](https://www.react.doctor/docs/configuration/config-files#enable-optional-project-graph-rules)

**Transfer to Code Rules:** In Validation, name the evidence required to establish the claim and distinguish “valid exception” from “inspect the referenced implementation before deciding.” A detector's skipped case must not become an agent-facing permission. Identify the minimum context a check needs, such as caller contracts, wrapper components, or whole-project references. A diff alone may be insufficient. An agent may resolve a delegated implementation that a static detector cannot inspect. This improves guidance without choosing an enforcement workflow.

## 3. State the prerequisite that makes the advice true

React Router rules check installed version, routing mode, import identity, and package boundaries. For example, route-level lazy loading advice requires React Router 6.9+ in library/data mode and is disabled in framework mode; framework-specific checks require evidence of the framework package. [PR #1411](https://github.com/millionco/react-doctor/pull/1411)

**Transfer to Code Rules:** If advice depends on a version, runtime, framework mode, or component contract, state that condition in the full rule and cite the authoritative contract. The `whenToRead` cue can mention it briefly when it helps selection. Avoid assuming that a dependency somewhere in a monorepo proves applicability everywhere. No need for a universal machine-readable version schema unless a concrete selection feature needs it.

## 4. Treat wrong advice as a rule defect and retain the correction history

React Doctor retired 33 IDs in one cleanup, including recommendations against supported React Native behavior and an insufficiently justified manual-memoization prohibition. IDs still resolve but no longer emit findings; formerly firing examples now assert silence. The PR explicitly distinguishes limited probes from a measured false-positive rate. [PR #1802](https://github.com/millionco/react-doctor/pull/1802)

**Transfer to Code Rules:** A rule update should be able to narrow or withdraw a recommendation, not only add more rules. Record why the recommendation changed and provide a valid counterexample when that explains the correction. Adoption updates should eventually make withdrawals/replacements visible before a consumer chooses to update. Exact retirement mechanics require a separate design decision; do not silently resurrect stale advice from an old licensed snapshot.

## 5. Separate project policy choices from general correctness claims

React Doctor made all design-tagged rules opt-in after individual authors inconsistently set their defaults. Focused design scans and explicit configuration still enable them. [PR #1434](https://github.com/millionco/react-doctor/pull/1434)

**Transfer to Code Rules:** An adopted style or design convention can be legitimate without being a universal defect. Explain its goal, tradeoff, and conditions instead of inventing a runtime failure. Libraries can organize opinionated policies as deliberately adopted groups; Code Rules need not adopt React Doctor's scan presets, scores, or automatic enforcement behavior.

## Smallest useful next change

Refine the canonical authoring guide with four optional prompts: what mechanism supports this advice; what similar-looking example is valid; what additional context must be checked before concluding; and what version/runtime assumptions matter. Review the meaningful-steps rule against these prompts. Keep them as useful authoring guidance, not mandatory metadata or repeated boilerplate. Rule retirement/update visibility is a separate future product consideration.

## Comparison with current Code Rules

The canonical authoring guide already covers scoped obligations, meaningful examples, insufficient evidence, and honest claims. The suggestions sharpen those existing prompts rather than introduce a new required schema. The current update guide already proposes added/removed/changed rule reports and review of upstream changes hidden by replacements. Retirement reasons and migration guidance would extend that design; they are not existing runtime capabilities.

Local references: [authoring guide](../../docs/src/content/docs/reference/rule-authoring.md), [update guide](../../docs/src/content/docs/guides/update.md).
