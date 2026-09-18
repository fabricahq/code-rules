// Report project freshness without presenting proposed repairs as completed file changes.

package cli

import (
	"context"
	"errors"
	"strings"

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
	report, err := project.Check(ctx, options)
	if err != nil {
		return projectCheckResult{}, err
	}
	result := projectCheckResult{Status: "up_to_date", Problems: []checkProblem{}}
	for _, problem := range report.Problems {
		message := ""
		switch problem.Kind {
		case project.MissingFile:
			message = "Missing generated file"
		case project.StaleContents:
			message = "Stale generated contents"
		case project.UnexpectedFile:
			message = "Unexpected generated file"
		case project.OutdatedGuide:
			message = "Project guide is missing or outdated; preserve any manual edits before refreshing"
		}
		command := "build"
		if problem.Repair == project.RefreshGuide {
			command = "init"
		}
		result.Problems = append(result.Problems, checkProblem{Kind: string(problem.Kind), Path: problem.Path, Message: message, NextStep: checkRepairCommand(command, configArgument)})
	}
	if !report.Current() {
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
