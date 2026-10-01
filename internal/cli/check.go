// Report project freshness without presenting proposed repairs as completed file changes.

package cli

import (
	"context"

	"github.com/fabricahq/code-rules/internal/project"
)

// projectCheckResult describes the current project; checking never applies the suggested repairs.
type projectCheckResult struct {
	Status   string         `json:"status"`
	Problems []checkProblem `json:"problems"`
}

// checkProblem identifies an observed mismatch and the steps that can repair it, in the same form as authoring
// results' next steps.
type checkProblem struct {
	Kind      string     `json:"kind"`
	Path      string     `json:"path"`
	Message   string     `json:"message"`
	NextSteps []nextStep `json:"nextSteps"`
}

// rebuild is the repair of every problem project check reports: regenerating output and the managed guide.
var rebuild = nextStep{Instruction: "Regenerate the project's generated files and Code Rules guide:", Commands: []string{"code-rules project build"}}

// checkProject checks generated files and the managed README, keeping unreadable or invalid inputs as operational errors.
func checkProject(ctx context.Context, options project.Options) (projectCheckResult, error) {
	report, err := project.Check(ctx, options)
	if err != nil {
		return projectCheckResult{}, err
	}
	result := projectCheckResult{Status: "up-to-date", Problems: []checkProblem{}}
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
			message = "Code Rules guide is missing or outdated; preserve any manual edits before refreshing"
		}
		result.Problems = append(result.Problems, checkProblem{Kind: string(problem.Kind), Path: problem.Path, Message: message, NextSteps: []nextStep{rebuild}})
	}
	if !report.Current() {
		result.Status = "out-of-date"
	}
	return result, nil
}
