// Collect a library change note's options and prompts, then record it through library.PlanChange.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/spf13/cobra"
)

// changeLevels explains each version change the way the rule versions reference defines it.
const changeLevels = `  major  Work that complied with the previous version could fail this one.
  minor  Work that complied still complies, and this version adds new guidance.
  patch  Work that complied still complies, and this version adds no new guidance.`

// libraryChangeCommand writes a new change note, prompting in a terminal for a missing change level or summary.
func libraryChangeCommand(options Options, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("change RULE_ID...", "Record rule changes in a new change note", requiredArguments("rule ID", "practices/testing/my-rule --bump minor --summary 'Add a Python example.'", "Name each rule by its path without .md, such as practices/testing/my-rule."), options)
	cmd.Long = cmd.Short + "\n\nAfter the first library release, every rule change needs a change note in changes/.\nThe next library release publishes the notes added since the previous one.\n\nFor rules that have a version, choose --bump:\n" + changeLevels + "\nLeave out --bump for a new rule; it starts at version 1.0.0. To retire a rule,\ndelete its Markdown file and asset directory, then pass --retire.\n\nWrite the summary for project maintainers deciding whether to update." + documentationHelp
	f.add(cmd, "bump", "Change for rules that have a version: major, minor, or patch")
	f.add(cmd, "summary", "One line describing the change for project maintainers")
	cmd.Flags().Bool("retire", false, "Record that the rules are retired; delete their Markdown files and asset directories first")
	f.add(cmd, "replaced-by", "With --retire and one rule: the rule that replaces it")
	f.prompts = map[string]string{"bump": "Change (major, minor, or patch)", "summary": "Summary"}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		target, err := f.libraryOptions(cmd.Context(), false)
		if err != nil {
			return err
		}
		retire, _ := cmd.Flags().GetBool("retire")
		if err := changeFlagsUsage(f, args, retire); err != nil {
			return usage(err)
		}
		request := library.ChangeRequest{IDs: args, Bump: rules.Change(f.value("bump")), Summary: f.value("summary"), Retire: retire, ReplacedBy: f.value("replaced-by")}
		plan, err := library.PlanChange(cmd.Context(), request, target)
		if err != nil {
			return err
		}
		f.introduction = changeIntroduction(args, plan.Versioned, retire)
		if plan.Versioned {
			if err := f.require("bump"); err != nil {
				return err
			}
		}
		if err := f.require("summary"); err != nil {
			return err
		}
		result, err := plan.Commit(cmd.Context(), rules.Change(f.value("bump")), f.value("summary"), time.Now())
		if err != nil {
			return err
		}
		output.report = changeRecordedReport(result, authoringScope{library: true, directory: f.value("directory"), workdir: f.directory})
		return nil
	}
	return cmd
}

// changeFlagsUsage rejects option combinations the command never accepts, whatever the library holds.
func changeFlagsUsage(f *authoringFlags, ids []string, retire bool) error {
	switch bump := f.value("bump"); {
	case bump != "" && validateAnswer("bump", bump) != nil:
		return validateAnswer("bump", bump)
	case bump != "" && retire:
		return fmt.Errorf("--bump isn't accepted with --retire")
	case f.value("replaced-by") != "" && !retire:
		return fmt.Errorf("--replaced-by requires --retire")
	case f.value("replaced-by") != "" && len(ids) != 1:
		return fmt.Errorf("--replaced-by requires a single rule ID")
	case strings.ContainsAny(f.value("summary"), "\r\n"):
		return fmt.Errorf("--summary must be one line")
	}
	return nil
}

// changeIntroduction orients the author before the first prompt for a change level or summary.
func changeIntroduction(ids []string, versioned, retire bool) string {
	switch {
	case retire:
		return fmt.Sprintf("Recording the retirement of: %s\n\nExplain in the summary why the rule is retired.\n\n", strings.Join(ids, ", "))
	case versioned:
		return fmt.Sprintf("Recording a change to: %s\n\nChoose how much the rule changed:\n%s\nWhen unsure, choose the larger change.\n\nWrite the summary for project maintainers deciding whether to update:\nsay what changed in the obligation or guidance.\n\n", strings.Join(ids, ", "), changeLevels)
	}
	return fmt.Sprintf("Recording a new rule: %s\n\nNew rules start at version 1.0.0. Write the summary for project maintainers\ndeciding whether to update: say what the rule adds.\n\n", strings.Join(ids, ", "))
}

// changeRecordedReport lists the new note and how to validate it with the rule change.
func changeRecordedReport(result library.AuthoringResult, scope authoringScope) commandReport {
	var out strings.Builder
	out.WriteString("Change note created.\n")
	formatAuthored(&out, result.Added, result.Changed, result.Warnings, scope.workdir)
	steps := []nextStep{{Instruction: "Next: Commit the note with the rule change, then validate the library:", Commands: []string{scope.command("check")}}}
	return authoredReport(&out, result.Added, result.Changed, result.Warnings, steps)
}
