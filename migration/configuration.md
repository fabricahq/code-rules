# Project configuration

PR #18 builds on #17 (`codex/go-version-ranges`) and targets that branch for an isolated review diff. The user approved preparing multiple dependent PRs together; merge approval is still separate. After its predecessor merges, retarget this PR to `go-migration`.

`rules.ParseConfiguration(json.RawMessage) (Configuration, error)` combines the existing repository, ref, constraint, and group parsers. It accepts schema version 1, sorts source aliases and explicit group IDs, preserves authored reasons, rejects unknown fields, and rejects duplicate repository identities and conflicting exclusions/replacements. Replacement paths must be contained beneath `local/`. Missing optionality is deliberate: `groups`, `exclude`, and `replace` are required, including empty values.

The configuration owns source declarations only. It does not contain output settings. No files are read, refs resolved, or group patterns expanded by this function. Errors retain the precise configuration field and return a zero configuration. Source version selection is represented directly by `Version`; exact refs have `ParsedRef`.

29 independent complete-document fixtures run against Go and the unchanged TypeScript reference, with JSON projections for TypeScript maps. All 669 accumulated comparisons pass. Native HashiCorp constraint behavior remains the approved difference from PR #17. Existing group diagnostic improvements continue to apply. Unknown fields are checked in deterministic key order.

The walkthrough is at `/walkthrough/pr18`; `/walkthrough/pr17` remains available. Each invokes the same native binary and covers only its own capability. Run `go run ./cmd/rules-lab -serve -port 4391`.

Validation: Go race tests, vet, and the 669-case shared comparison suite. Stack-wide Staticcheck, repository checks, and installed-package checks run before handoff. CI and independent CodeRabbit review are requested on the ready PR. The source reference remains `7013d3d374a33a5cf65a2a48ff6870e46f9d7209`; the engineering corpus remains `e2166f90333157fd3e14c24d3e43287ece858e4b`.
