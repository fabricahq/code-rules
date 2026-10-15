// Determine complete project freshness without applying repairs or exposing storage details.

package project

import (
	"context"
	"os"
	"path"

	"github.com/fabricahq/code-rules/internal/rules"
)

// ProblemKind distinguishes the observed mismatch without relying on presentation text.
type ProblemKind string

const (
	MissingFile    ProblemKind = "missing-file"
	StaleContents  ProblemKind = "stale-contents"
	UnexpectedFile ProblemKind = "unexpected-file"
	OutdatedGuide  ProblemKind = "outdated-readme"
	// OutdatedAgents means the repository's AGENTS.md lacks the Syllabus section, or holds an older one.
	OutdatedAgents ProblemKind = "outdated-agents-section"
)

// RepairAction identifies the project operation that can repair an observed mismatch.
type RepairAction string

const (
	Rebuild      RepairAction = "rebuild"
	RefreshGuide RepairAction = "refresh-guide"
)

// Problem identifies a path relative to the configuration directory, such as ../AGENTS.md, and its intended repair.
// Refreshing a guide still requires preserving manual edits; Check never authorizes overwriting them.
type Problem struct {
	Kind   ProblemKind
	Path   string
	Repair RepairAction
}

// CheckResult lists all freshness problems in deterministic order; an empty list means current.
// Invalid, unreadable, or concurrently changing inputs return an error instead of this report.
type CheckResult struct{ Problems []Problem }

// Current reports whether generated output, the managed guide, and the AGENTS.md section match the selected inputs.
func (r CheckResult) Current() bool { return len(r.Problems) == 0 }

// Check compares generated output, the managed guide, and the Syllabus section of the repository's AGENTS.md in one
// optimistic snapshot. It performs no writes, locks, recovery, Git, or network access. Staleness is a result, not an
// error; an edited section or broken markers in AGENTS.md are errors, as they are for build.
func Check(ctx context.Context, options Options) (CheckResult, error) {
	guideName, guideBytes := projectGuide()
	var agents agentsPlan
	defer func() { agents.close() }()
	changes, guideChanges, err := checkProjectFiles(ctx, options, map[string][]byte{guideName: guideBytes}, func(root *os.Root, config rules.Configuration) error {
		var err error
		agents, err = planAgentsSection(ctx, root, config, options)
		return err
	})
	if err != nil {
		return CheckResult{}, err
	}
	// The snapshot covers the Syllabus directory; AGENTS.md, outside it, must also be as planned when it ends.
	if err := agents.requireUnchanged(ctx); err != nil {
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
	if agents.content != nil {
		result.Problems = append(result.Problems, Problem{Kind: OutdatedAgents, Path: path.Join("..", agentsFile), Repair: Rebuild})
	}
	return result, nil
}
