# Release selection

PR #20 targets `go-migration` after #19 merged. It remains unmerged for the user's review.

`SelectVersion(advertisement string, constraint VersionConstraint)` reads the bounded `git ls-remote --tags` format without running Git. Ordinary non-version tags are skipped through `errors.As` on the existing validation error. Matching and precedence use HashiCorp go-version. The highest matching release wins; equivalent versions at different peeled commits produce `VersionAmbiguous`. Safe aliases use deterministic spelling order. The returned object is the original tag object, preserving evidence needed for later fetch-time movement checks.

`VersionSelectionError.Kind` distinguishes no match, ambiguity, invalid Git output, and limits. Zero constraints produce a validation error. Failures return a zero result. Discovery is limited to 20,000 distinct records and 8 MiB, including annotated-tag peel records. Exact ref fetching, subprocess cancellation, authentication, and moved-tag verification remain outside this pure function.

20 independent advertisement fixtures cover empty/single/multiple candidates, unsorted numeric precedence, ordinary tags, stable/prerelease matching, aliases, metadata, annotated tags, ambiguity below the winning version, and malformed protocol. Direct tests cover the exact record threshold, the next record, zero constraints, and the byte limit.

`tests/migration/compare-version-selection.ts` creates a real local Git repository and compares the native result to the pinned TypeScript importer for annotated aliases. It then creates an equal-version alias at a different commit and requires both implementations to reject it. Its temporary repository is removed afterward. The Go workflow runs this check after the existing shared suite. No network repository is accessed.

Walkthrough: `/walkthrough/pr20`. Edit the advertisement and constraint, then invoke real Go. Earlier PRs retain their own walkthrough pages. All 661 existing shared cases continue to run, with 202 explicitly approved differences. Go race/vet and real-Git comparisons run for this slice; final stack-wide checks and independent review precede any merge. Reference and engineering corpus pins are unchanged.
