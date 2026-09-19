// Determine complete project freshness without applying repairs or exposing storage details.

package project

import (
	"context"
	"path"
)

// ProblemKind distinguishes the observed mismatch without relying on presentation text.
type ProblemKind string

const (
	MissingFile    ProblemKind = "missing_file"
	StaleContents  ProblemKind = "stale_contents"
	UnexpectedFile ProblemKind = "unexpected_file"
	OutdatedGuide  ProblemKind = "outdated_readme"
)

// RepairAction identifies the project operation that can repair an observed mismatch.
type RepairAction string

const (
	Rebuild      RepairAction = "rebuild"
	RefreshGuide RepairAction = "refresh-guide"
)

// Problem identifies a path relative to the configuration directory and its intended repair.
// Refreshing a guide still requires preserving manual edits; Check never authorizes overwriting them.
type Problem struct {
	Kind   ProblemKind
	Path   string
	Repair RepairAction
}

// CheckResult lists all freshness problems in deterministic order; an empty list means current.
// Invalid, unreadable, or concurrently changing inputs return an error instead of this report.
type CheckResult struct{ Problems []Problem }

// Current reports whether both generated output and the managed guide match the selected inputs.
func (r CheckResult) Current() bool { return len(r.Problems) == 0 }

// Check compares generated output and the managed guide in one optimistic snapshot.
// It performs no writes, locks, recovery, Git, or network access. Staleness is a result, not an error.
func Check(ctx context.Context, options Options) (CheckResult, error) {
	guideName, guideBytes := projectGuide(options.ConfigPath)
	changes, guideChanges, err := checkWithFiles(ctx, options, map[string][]byte{guideName: guideBytes})
	if err != nil {
		return CheckResult{}, err
	}
	result := CheckResult{Problems: []Problem{}}
	for _, group := range []struct {
		kind  ProblemKind
		paths []string
	}{
		{MissingFile, changes.Added}, {StaleContents, changes.Changed}, {UnexpectedFile, changes.Removed},
	} {
		for _, name := range group.paths {
			result.Problems = append(result.Problems, Problem{Kind: group.kind, Path: path.Join("generated", name), Repair: Rebuild})
		}
	}
	if len(guideChanges.Added)+len(guideChanges.Changed) > 0 {
		result.Problems = append(result.Problems, Problem{Kind: OutdatedGuide, Path: guideName, Repair: RefreshGuide})
	}
	return result, nil
}
