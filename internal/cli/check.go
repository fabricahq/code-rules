// Report project freshness without presenting proposed repairs as completed file changes.

package cli

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/authoring"
	"github.com/fabricahq/code-rules/internal/project"
)

var errCheckOutOfDate = errors.New("project is out of date; see the reported problems and next steps")

// projectCheckResult describes the current project; checking never applies the suggested repairs.
type projectCheckResult struct {
	Status   string         `json:"status"`
	Problems []checkProblem `json:"problems"`
}

// checkProblem identifies an observed mismatch and the command that can repair it.
type checkProblem struct {
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Message  string `json:"message"`
	NextStep string `json:"nextStep"`
}

// checkProject checks generated files and the managed README, keeping unreadable or invalid inputs as operational errors.
func checkProject(ctx context.Context, options project.Options, configArgument string) (projectCheckResult, error) {
	guideName, guideBytes := authoring.ProjectGuide(options.ConfigPath)
	changes, guideChanges, err := project.CheckWithFiles(ctx, options, map[string][]byte{guideName: guideBytes})
	if err != nil {
		return projectCheckResult{}, err
	}
	result := projectCheckResult{Status: "up_to_date", Problems: []checkProblem{}}
	for _, group := range []struct {
		kind, message string
		paths         []string
	}{
		{"missing_file", "Missing generated file", changes.Added},
		{"stale_contents", "Stale generated contents", changes.Changed},
		{"unexpected_file", "Unexpected generated file", changes.Removed},
	} {
		for _, path := range group.paths {
			result.Problems = append(result.Problems, checkProblem{Kind: group.kind, Path: filepath.ToSlash(filepath.Join("generated", filepath.FromSlash(path))), Message: group.message, NextStep: checkRepairCommand("build", configArgument)})
		}
	}
	if len(guideChanges.Added)+len(guideChanges.Changed) > 0 {
		result.Problems = append(result.Problems, checkProblem{Kind: "outdated_readme", Path: guideName, Message: "Project guide is missing or outdated; preserve any manual edits before refreshing", NextStep: checkRepairCommand("init", configArgument)})
	}
	if len(result.Problems) > 0 {
		result.Status = "out_of_date"
		return result, errCheckOutOfDate
	}
	return result, nil
}

// checkRepairCommand keeps custom configuration selections and quotes them for copying into a shell.
func checkRepairCommand(command, config string) string {
	result := "code-rules " + command
	if config != "" {
		result += " --config='" + strings.ReplaceAll(config, "'", "'\"'\"'") + "'"
	}
	return result
}
