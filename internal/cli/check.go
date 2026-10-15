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
	// Warnings never change Status.
	Warnings []string `json:"warnings"`
}

// checkProblem identifies an observed mismatch and the steps that can repair it, in the same form as authoring
// results' next steps.
type checkProblem struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
	// Skill is the project ID of the skill an installed skill folder problem is about; it is omitted for other
	// problems.
	Skill     string     `json:"skill,omitempty"`
	Message   string     `json:"message"`
	NextSteps []nextStep `json:"nextSteps"`
}

// rebuild is the repair of every problem project check reports about generated files and the managed guide:
// regenerating them.
var rebuild = nextStep{Instruction: "Regenerate the project's generated files and Code Rules guide:", Commands: []string{"code-rules project build"}}

// reinstallSkills is the repair of every problem project check reports about an installed skill folder: installing
// fresh copies from each skill's source.
var reinstallSkills = nextStep{Instruction: "Reinstall the project's skills from their sources, replacing any edits to the installed copies:", Commands: []string{"code-rules project build"}}

// checkProject checks generated files, installed skill folders, and the managed README, keeping unreadable or invalid
// inputs as operational errors.
func checkProject(ctx context.Context, options project.Options) (projectCheckResult, error) {
	report, err := project.Check(ctx, options)
	if err != nil {
		return projectCheckResult{}, err
	}
	result := projectCheckResult{Status: "up-to-date", Problems: []checkProblem{}, Warnings: nonNil(report.Warnings)}
	for _, problem := range report.Problems {
		message := ""
		repair := rebuild
		switch problem.Kind {
		case project.MissingFile:
			message = "Missing generated file"
		case project.StaleContents:
			message = "Stale generated contents"
		case project.UnexpectedFile:
			message = "Unexpected generated file"
		case project.OutdatedGuide:
			message = "Code Rules guide is missing or outdated; preserve any manual edits before refreshing"
		case project.MissingSkill:
			message, repair = "Installed skill "+problem.Skill+" is missing", reinstallSkills
		case project.ModifiedSkill:
			message, repair = "Installed skill "+problem.Skill+" differs from its source; don't edit installed copies, since build replaces them", reinstallSkills
		case project.UnexpectedSkill:
			message, repair = "Installed skill "+problem.Skill+" is no longer installed by this project; build removes it", reinstallSkills
		}
		result.Problems = append(result.Problems, checkProblem{Kind: string(problem.Kind), Path: problem.Path, Skill: problem.Skill, Message: message, NextSteps: []nextStep{repair}})
	}
	if !report.Current() {
		result.Status = "out-of-date"
	}
	return result, nil
}
