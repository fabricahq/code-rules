// Build command reports while scope and inputs are known; both output modes share next steps.

package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/project"
)

// commandReport owns a completed command's data, human rendering, and optional reported failure.
// A failed check retains its report; execution errors are handled separately and never imply success.
type commandReport struct {
	value   any
	human   string
	failure *responseError
}

// nextStep keeps recovery or follow-up commands separate from their explanation for automation.
type nextStep struct {
	Instruction string   `json:"instruction"`
	Commands    []string `json:"commands,omitempty"`
}

type authoringValue struct {
	Files     []string   `json:"files"`
	Warnings  []string   `json:"warnings,omitempty"`
	Next      string     `json:"next"`
	NextSteps []nextStep `json:"nextSteps"`
}

// authoringScope retains explicit invocation context without reading Cobra flags during rendering.
type authoringScope struct {
	library   bool
	directory string
}

func (s authoringScope) command(action string) string {
	if !s.library {
		return "code-rules project " + action
	}
	command := "code-rules library " + action
	if s.directory != "" {
		command += " --directory='" + strings.ReplaceAll(s.directory, "'", "'\"'\"'") + "'"
	}
	return command
}

// authoredReport renders each step once and uses the same text in the legacy next field.
func authoredReport(out *strings.Builder, files, warnings []string, steps []nextStep) commandReport {
	var next strings.Builder
	for _, step := range steps {
		if next.Len() > 0 {
			next.WriteByte('\n')
		}
		next.WriteString(step.Instruction + "\n")
		for _, command := range step.Commands {
			fmt.Fprintf(&next, "  %s\n", command)
		}
	}
	if next.Len() > 0 {
		out.WriteByte('\n')
		out.WriteString(next.String())
	}
	return commandReport{value: authoringValue{files, warnings, strings.TrimSpace(next.String()), steps}, human: out.String()}
}

func projectInitializedReport(result project.AuthoringResult) commandReport {
	var out strings.Builder
	if len(result.Files) == 0 {
		out.WriteString("Code Rules is already initialized.\nNo files changed.\n")
	} else {
		out.WriteString("Code Rules initialized!\n")
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning)
	}
	out.WriteString("\nCode Rules directory: .code-rules\nProject configuration: .code-rules/config.yaml\n")
	steps := []nextStep{{Instruction: "Run code-rules project --help to manage this project's rules.", Commands: []string{"code-rules project --help"}}}
	if len(result.Files) > 0 {
		steps = []nextStep{
			{Instruction: "Start with a project-only rule (example):", Commands: []string{"code-rules project add group practices/testing", "code-rules project add rule practices/testing/my-rule"}},
			{Instruction: "Or use a shared library (example):", Commands: []string{"code-rules project add library team", "code-rules project sync"}},
		}
	}
	return authoredReport(&out, result.Files, result.Warnings, steps)
}

func sourceAddedReport(result project.AuthoringResult) commandReport {
	var out strings.Builder
	formatAuthored(&out, result.Files, result.Warnings)
	return authoredReport(&out, result.Files, result.Warnings, []nextStep{{Instruction: "Next: Fetch the library rules and build guidance:", Commands: []string{"code-rules project sync"}}})
}

func libraryInitializedReport(result library.AuthoringResult, scope authoringScope) commandReport {
	var out strings.Builder
	formatAuthored(&out, result.Files, result.Warnings)
	steps := []nextStep{}
	if !result.LicenseDeclared {
		steps = append(steps, nextStep{Instruction: "License is undeclared. Decide terms before sharing."})
	}
	steps = append(steps, nextStep{Instruction: "Next: Add a group and rule, then validate the library:", Commands: []string{scope.command("add group practices/testing"), scope.command("add rule practices/testing/my-rule"), scope.command("check")}})
	return authoredReport(&out, result.Files, result.Warnings, steps)
}

func ruleCreatedReport(files, warnings []string, draft bool, scope authoringScope) commandReport {
	var out strings.Builder
	if draft {
		out.WriteString("Rule draft created:\n")
	} else {
		out.WriteString("Rule created from --body-file:\n")
	}
	for _, file := range files {
		fmt.Fprintf(&out, "  %s\n", file)
	}
	for _, warning := range warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning)
	}
	instruction := "Review the Markdown file above. Make future edits directly in that file."
	if draft {
		instruction = "Next: Open the Markdown file above in your editor.\nKeep the metadata between the --- lines at the top. Below it:\n  - State the instructions and explain why they matter.\n  - Add correct and incorrect examples, then describe how to check compliance.\n  - Replace <...> placeholders and remove unused template sections."
		if scope.library {
			instruction += "\n  - Remove the <!-- code-rules:draft --> marker when the rule is complete."
		}
	}
	commands := []string{scope.command("build"), scope.command("check")}
	if scope.library {
		commands = []string{scope.command("check")}
	}
	return authoredReport(&out, files, warnings, []nextStep{{Instruction: instruction}, {Instruction: ruleReadyHeading(scope.library), Commands: commands}})
}

func groupCreatedReport(files, warnings []string, groupPath string, scope authoringScope) commandReport {
	var out strings.Builder
	formatAuthored(&out, files, warnings)
	action := "build"
	if scope.library {
		action = "check"
	}
	steps := []nextStep{
		{Instruction: "Next: Add a rule to this group (replace my-rule with your rule's slug):", Commands: []string{scope.command("add rule " + groupPath + "/my-rule")}},
		{Instruction: ruleReadyHeading(scope.library), Commands: []string{scope.command(action)}},
	}
	return authoredReport(&out, files, warnings, steps)
}

func projectChangesReport(action string, result project.FileChanges) commandReport {
	var out strings.Builder
	if result.Guide != nil {
		status := "updated"
		if result.Guide.Created {
			status = "created"
		}
		fmt.Fprintf(&out, "Code Rules guide %s: %s\n", status, filepath.Join(".code-rules", result.Guide.Path))
	}
	fmt.Fprintf(&out, "%s complete: %d added, %d changed, %d removed.\n", strings.ToUpper(action[:1])+action[1:], len(result.Added), len(result.Changed), len(result.Removed))
	if len(result.Added)+len(result.Changed)+len(result.Removed) > 0 {
		if action == "build" {
			out.WriteString("Paths relative to .code-rules/generated:\n")
		} else {
			out.WriteString("Paths relative to the Code Rules directory: .code-rules\n")
		}
	}
	for _, group := range []struct {
		label string
		paths []string
	}{{"Add", result.Added}, {"Change", result.Changed}, {"Remove", result.Removed}} {
		for _, path := range group.paths {
			fmt.Fprintf(&out, "  %s: %s\n", group.label, path)
		}
	}
	return commandReport{value: result, human: out.String()}
}

func libraryCheckedReport(result library.CheckResult) commandReport {
	var out strings.Builder
	fmt.Fprintf(&out, "Library is valid: %d group(s), %d rule(s).\n", result.Groups, result.Rules)
	for _, warning := range result.Warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning)
	}
	return commandReport{value: result, human: out.String()}
}

func projectCheckedReport(result projectCheckResult) commandReport {
	var out strings.Builder
	report := commandReport{value: result}
	if result.Status == "up_to_date" {
		out.WriteString("Status: up to date.\nGenerated guidance and the Code Rules guide are current.\nNo files were changed.\n")
	} else {
		report.failure = &responseError{Kind: "out_of_date", Message: "this project's Code Rules files are out of date; see the reported problems and next steps"}
		out.WriteString("Status: out of date.\nNo files were changed.\nPaths are relative to the Code Rules directory: .code-rules\n\nProblems:\n")
		for _, problem := range result.Problems {
			fmt.Fprintf(&out, "  %s: %s\n    Next: %s\n", problem.Message, problem.Path, problem.NextStep)
		}
	}
	report.human = out.String()
	return report
}
