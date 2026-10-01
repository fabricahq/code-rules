// Build command reports while scope and inputs are known; both output modes share next steps.

package cli

import (
	"fmt"
	"path/filepath"

	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
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
	Commands    []string `json:"commands"`
}

// authoringValue is an authoring command's JSON value: the files it created and changed, as absolute paths.
type authoringValue struct {
	Added     []string   `json:"added"`
	Changed   []string   `json:"changed"`
	Warnings  []string   `json:"warnings"`
	NextSteps []nextStep `json:"nextSteps"`
}

// authoringScope retains explicit invocation context without reading Cobra flags during rendering.
type authoringScope struct {
	library bool
	// directory is the --directory option, and workdir the working directory that report paths are relative to.
	directory string
	workdir   string
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

// authoredReport renders each next step once, after the report, and returns the steps in its value. Every list
// in the value is present, empty when there's nothing to report.
func authoredReport(out *strings.Builder, added, changed, warnings []string, steps []nextStep) commandReport {
	var next strings.Builder
	for i, step := range steps {
		if step.Commands == nil {
			steps[i].Commands = []string{}
		}
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
	return commandReport{value: authoringValue{nonNil(added), nonNil(changed), nonNil(warnings), steps}, human: out.String()}
}

// nonNil returns list, or an empty list for nil, so JSON output shows [] rather than null.
func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func projectInitializedReport(result project.AuthoringResult) commandReport {
	var out strings.Builder
	if len(result.Written()) == 0 {
		out.WriteString("Code Rules is already initialized.\nNo files changed.\n")
	} else {
		out.WriteString("Code Rules initialized!\n")
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning)
	}
	out.WriteString("\nCode Rules directory: .code-rules\nProject configuration: .code-rules/config.yaml\n")
	steps := []nextStep{{Instruction: "Run code-rules project --help to manage this project's rules.", Commands: []string{"code-rules project --help"}}}
	if len(result.Written()) > 0 {
		steps = []nextStep{
			{Instruction: "Start with a project-only rule (example):", Commands: []string{"code-rules project add group practices/testing", "code-rules project add rule practices/testing/my-rule"}},
			{Instruction: "Or use a shared library (example):", Commands: []string{"code-rules project add library team", "code-rules project sync"}},
			{Instruction: "Then connect your coding agent to the rules; see .code-rules/README.md."},
		}
	}
	return authoredReport(&out, result.Added, result.Changed, result.Warnings, steps)
}

func sourceAddedReport(result project.AuthoringResult, scope authoringScope) commandReport {
	var out strings.Builder
	formatAuthored(&out, result.Added, result.Changed, result.Warnings, scope.workdir)
	return authoredReport(&out, result.Added, result.Changed, result.Warnings, []nextStep{{Instruction: "Next: Fetch the library rules and build guidance:", Commands: []string{"code-rules project sync"}}})
}

func libraryInitializedReport(result library.AuthoringResult, scope authoringScope) commandReport {
	var out strings.Builder
	formatAuthored(&out, result.Added, result.Changed, result.Warnings, scope.workdir)
	steps := []nextStep{}
	if !result.LicenseDeclared {
		steps = append(steps, nextStep{Instruction: "License is undeclared. Decide terms before sharing."})
	}
	if result.HasGroups {
		steps = append(steps, nextStep{Instruction: "Next: Validate the library:", Commands: []string{scope.command("check")}})
	} else {
		steps = append(steps, nextStep{Instruction: "Next: Add a group and rule, then validate the library:", Commands: []string{scope.command("add group practices/testing"), scope.command("add rule practices/testing/my-rule"), scope.command("check")}})
	}
	return authoredReport(&out, result.Added, result.Changed, result.Warnings, steps)
}

// ruleCreatedReport explains how to finish the rule; changeNote adds recording id's change note in a released library.
func ruleCreatedReport(added, changed, warnings []string, draft bool, scope authoringScope, id string, changeNote bool) commandReport {
	var out strings.Builder
	if draft {
		out.WriteString("Rule draft created.\n")
	} else {
		out.WriteString("Rule created from --body-file.\n")
	}
	formatAuthored(&out, added, changed, warnings, scope.workdir)
	instruction := "Review the Markdown file above. Make future edits directly in that file."
	if draft {
		instruction = "Next: Open the Markdown file above in your editor.\nKeep the metadata between the --- lines at the top. Below it:\n  - State the instructions and explain why they matter.\n  - Add correct and incorrect examples, then describe how to check compliance.\n  - Replace <...> placeholders and remove unused template sections."
		instruction += "\n  - Remove the " + rules.DraftMarker + " marker when the rule is complete."
	}
	commands := []string{scope.command("build"), scope.command("check")}
	if scope.library {
		commands = []string{scope.command("check")}
	}
	steps := []nextStep{{Instruction: instruction}, {Instruction: ruleReadyHeading(scope.library), Commands: commands}}
	if changeNote {
		steps = []nextStep{
			{Instruction: instruction},
			{Instruction: "After the first library release, every new rule needs a change note. After writing the rule text, record it with a summary for project maintainers:", Commands: []string{scope.command("change " + id + " --summary '<what the rule adds>'")}},
			{Instruction: "Then validate the library:", Commands: commands},
		}
	}
	return authoredReport(&out, added, changed, warnings, steps)
}

func groupCreatedReport(added, changed, warnings []string, groupPath string, scope authoringScope) commandReport {
	var out strings.Builder
	formatAuthored(&out, added, changed, warnings, scope.workdir)
	action := "build"
	if scope.library {
		action = "check"
	}
	steps := []nextStep{
		{Instruction: "Next: Add a rule to this group (replace my-rule with your rule's slug):", Commands: []string{scope.command("add rule " + groupPath + "/my-rule")}},
		{Instruction: ruleReadyHeading(scope.library), Commands: []string{scope.command(action)}},
	}
	return authoredReport(&out, added, changed, warnings, steps)
}

// projectChangesReport lists the files sync, build, or update changed in the project at root, naming their directory
// relative to workdir, as authoring reports name paths.
func projectChangesReport(action string, result project.FileChanges, root, workdir string) commandReport {
	var out strings.Builder
	codeRules := relativeDirectory(filepath.Join(root, ".code-rules"), workdir)
	if result.Guide != nil {
		status := "updated"
		if result.Guide.Created {
			status = "created"
		}
		fmt.Fprintf(&out, "Code Rules guide %s: %s\n", status, filepath.Join(codeRules, result.Guide.Path))
	}
	fmt.Fprintf(&out, "%s complete: %d added, %d changed, %d removed.\n", strings.ToUpper(action[:1])+action[1:], len(result.Added), len(result.Changed), len(result.Removed))
	if len(result.Added)+len(result.Changed)+len(result.Removed) > 0 {
		if action == "build" {
			fmt.Fprintf(&out, "Paths relative to %s:\n", filepath.Join(codeRules, "generated"))
		} else {
			fmt.Fprintf(&out, "Paths relative to %s:\n", codeRules)
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
	for _, warning := range result.Warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning)
	}
	return commandReport{value: result, human: out.String()}
}

func libraryCheckedReport(result library.CheckResult) commandReport {
	var out strings.Builder
	fmt.Fprintf(&out, "Library is valid: %d group(s), %d rule(s).\n", result.GroupCount, result.RuleCount)
	for _, warning := range result.Warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning)
	}
	// With nothing pending, check says what library release would, rather than describe an empty release.
	if len(result.PendingRelease.Rules)+len(result.PendingRelease.LibraryFiles) == 0 {
		fmt.Fprintf(&out, "\nNothing to publish: no pending change notes and no library-wide changes since release/%d.\n", result.PendingRelease.Release-1)
		return commandReport{value: result, human: out.String()}
	}
	fmt.Fprintf(&out, "\nPending library release %d\n", result.PendingRelease.Release)
	if len(result.PendingRelease.Rules) == 0 {
		out.WriteString("  No rule changes are pending.\n")
	}
	formatPendingRules(&out, result.PendingRelease.Rules)
	if files := result.PendingRelease.LibraryFiles; len(files) > 0 {
		fmt.Fprintf(&out, "Library-wide files changed since release/%d:\n  %s\n", result.PendingRelease.Release-1, strings.Join(files, "\n  "))
	}
	return commandReport{value: result, human: out.String()}
}

// pendingVersions shows a pending rule's current and next version, or the only one it has.
func pendingVersions(rule library.PendingRule) string {
	switch {
	case rule.LastVersion != nil && rule.ReplacedBy != "":
		return rule.LastVersion.String() + ", replaced by " + rule.ReplacedBy
	case rule.LastVersion != nil:
		return rule.LastVersion.String()
	case rule.From == nil:
		return rule.To.String()
	}
	return rule.From.String() + " -> " + rule.To.String()
}

func projectCheckedReport(result projectCheckResult) commandReport {
	var out strings.Builder
	report := commandReport{value: result}
	if result.Status == "up-to-date" {
		out.WriteString("Status: up to date.\nGenerated guidance and the Code Rules guide are current.\nNo files were changed.\n")
	} else {
		report.failure = &responseError{Kind: "out-of-date", Message: "this project's Code Rules files are out of date; see the reported problems and next steps"}
		out.WriteString("Status: out of date.\nNo files were changed.\nPaths relative to .code-rules:\n\nProblems:\n")
		for _, problem := range result.Problems {
			fmt.Fprintf(&out, "  %s: %s\n    Next: %s\n", problem.Message, problem.Path, strings.Join(problem.NextSteps[0].Commands, "; "))
		}
	}
	report.human = out.String()
	return report
}

// relativeDirectory returns directory, which is absolute, relative to workdir, resolving symbolic links so a working
// directory reached through one, such as macOS's /var for /private/var, still compares. It returns directory as is
// when no relative path leads there.
func relativeDirectory(directory, workdir string) string {
	if absolute, err := filepath.Abs(workdir); err == nil {
		workdir = absolute
	}
	if resolved, err := filepath.EvalSymlinks(workdir); err == nil {
		workdir = resolved
	}
	if resolved, err := filepath.EvalSymlinks(directory); err == nil {
		directory = resolved
	}
	relative, err := filepath.Rel(workdir, directory)
	if err != nil {
		return directory
	}
	return relative
}
