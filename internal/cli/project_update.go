// Collect a project update's scope, decisions, and confirmation, then preview or apply it through project.PlanUpdate.

package cli

import (
	"fmt"
	"strings"

	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/spf13/cobra"
)

// projectUpdateCommand previews newer rule versions and applies them after confirmation or with --yes.
func projectUpdateCommand(options Options, output *commandOutput) *cobra.Command {
	cmd := &cobra.Command{Use: "update [SOURCE | SOURCE:RULE ...]", Short: "Preview and apply newer rule versions from libraries", Args: cobra.ArbitraryArgs}
	cmd.Long = "Preview newer rule versions, new rules, and retirements from your libraries, then\napply them after you confirm. With no arguments, every source is updated.\nSOURCE, such as team, updates one library. SOURCE:RULE, such as\nteam:techs/go/errors, moves only that rule and adds no new rules. Sources that\nuse ref don't move.\n\nIn a terminal, update asks about each major change and retirement (adopt it, or\nkeep the current version with a pin) and each new rule (add it, or exclude it),\nthen asks you to confirm. Without a terminal, or with --json, it only shows the\npreview unless you pass --yes. The update applies exactly the versions the\npreview showed." + documentationHelp
	f := &authoringFlags{command: cmd, values: map[string]*singleString{}, directory: options.Directory}
	cmd.PostRunE = f.finishPrompts
	cmd.Flags().Bool("yes", false, "Apply the previewed changes without asking; required without a terminal or with --json")
	var keep, exclude []string
	cmd.Flags().StringArrayVar(&keep, "keep", nil, "Pin `SOURCE:RULE` at its current version instead of updating it (repeat); requires --reason")
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, "Exclude the new rule `SOURCE:RULE` instead of adding it (repeat); requires --reason")
	f.add(cmd, "reason", "Reason recorded with each pin and exclusion the update writes")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		targets, err := updateTargets(args)
		if err != nil {
			return usage(err)
		}
		decisions, err := flagDecisions(keep, exclude, f.value("reason"))
		if err != nil {
			return usage(err)
		}
		directory, err := commandDirectory(cmd.Context(), options.Directory, "project", false)
		if err != nil {
			return err
		}
		plan, err := project.PlanUpdate(cmd.Context(), project.Options{Directory: directory, ToolVersion: options.Version}, options.Git, targets)
		if err != nil {
			return err
		}
		preview, err := plan.Preview(decisions)
		if err != nil {
			return err
		}
		yes, _ := cmd.Flags().GetBool("yes")
		interactive := !yes && f.interactive()
		if !yes && !interactive {
			output.report = updateReport(preview, false)
			return nil
		}
		if interactive && preview.Moves() {
			var confirmed bool
			if decisions, confirmed, err = askUpdateDecisions(f, preview, decisions); err != nil {
				return err
			}
			if !confirmed {
				if preview, err = plan.Preview(decisions); err != nil {
					return err
				}
				output.report = updateReport(preview, true)
				return nil
			}
		}
		result, err := plan.Apply(cmd.Context(), decisions)
		if err != nil {
			return err
		}
		output.report = updateReport(result, false)
		return nil
	}
	return cmd
}

// updateTargets parses SOURCE and SOURCE:RULE arguments; whether each names a configured source and an imported
// rule is checked against the project.
func updateTargets(args []string) ([]imports.UpdateTarget, error) {
	targets := []imports.UpdateTarget{}
	for _, arg := range args {
		source, rule, scoped := strings.Cut(arg, ":")
		if source == "" {
			return nil, fmt.Errorf("%q: expected SOURCE or SOURCE:RULE, such as team or team:techs/go/errors", arg)
		}
		if scoped {
			if err := rules.ValidateRuleID(rule, arg); err != nil {
				return nil, err
			}
		}
		targets = append(targets, imports.UpdateTarget{Source: source, Rule: rule})
	}
	return targets, nil
}

// flagDecisions turns --keep and --exclude into decisions with --reason, ignoring repeated values. --reason is
// required with either and accepted only with one of them.
func flagDecisions(keep, exclude []string, reason string) ([]project.UpdateDecision, error) {
	if len(keep)+len(exclude) > 0 && reason == "" {
		return nil, fmt.Errorf("--keep and --exclude require --reason, which is recorded with each pin and exclusion")
	}
	if reason != "" && len(keep)+len(exclude) == 0 {
		return nil, fmt.Errorf("--reason requires --keep or --exclude")
	}
	decisions := []project.UpdateDecision{}
	seen := map[string]bool{}
	for _, group := range []struct {
		flag   string
		values []string
	}{{"--keep", keep}, {"--exclude", exclude}} {
		for _, value := range group.values {
			source, rule, ok := strings.Cut(value, ":")
			if !ok || source == "" {
				return nil, fmt.Errorf("%s %q: expected SOURCE:RULE, such as team:techs/go/errors", group.flag, value)
			}
			if err := rules.ValidateRuleID(rule, group.flag+" "+value); err != nil {
				return nil, err
			}
			if seen[group.flag+value] {
				continue
			}
			seen[group.flag+value] = true
			decisions = append(decisions, project.UpdateDecision{Source: source, Rule: rule, Keep: group.flag == "--keep", Reason: reason})
		}
	}
	return decisions, nil
}

// askUpdateDecisions shows the preview, asks about each major change, retirement, and new rule the flags didn't
// decide, and then asks for confirmation. It returns the flags' decisions followed by the answers, and whether
// the update was confirmed.
func askUpdateDecisions(f *authoringFlags, preview project.UpdateResult, decisions []project.UpdateDecision) ([]project.UpdateDecision, bool, error) {
	var intro strings.Builder
	formatUpdatePreview(&intro, preview.Sources)
	f.introduction = intro.String()
	for _, source := range preview.Sources {
		for _, row := range source.Rules {
			if row.Decision != "" || row.Pin != nil {
				continue
			}
			// Each question names its rule on a line of its own, so prompts stay short enough not to wrap.
			name := source.Name + ":" + row.ID
			var context, question, reasonLabel string
			var choices [2]string
			switch row.Change {
			case imports.UpdateMajor:
				choices = [2]string{"adopt", "keep"}
				context = fmt.Sprintf("%s: major change, %s -> %s.", name, row.From, row.To)
				question = fmt.Sprintf("Adopt it, or keep %s?", row.From)
				reasonLabel = "Reason for keeping it:"
			case imports.UpdateRetired:
				choices = [2]string{"drop", "keep"}
				context = fmt.Sprintf("%s: retired.", name)
				question = fmt.Sprintf("Drop it, or keep %s?", row.From)
				reasonLabel = "Reason for keeping it:"
			case imports.UpdateNew:
				choices = [2]string{"add", "exclude"}
				context = fmt.Sprintf("%s: new rule, %s.", name, row.To)
				question = "Add it, or exclude it?"
				reasonLabel = "Reason for excluding it:"
			default:
				continue
			}
			f.introduction += "\n" + context + "\n"
			answer, err := askChoice(f, question, choices)
			if err != nil {
				return nil, false, err
			}
			if answer == choices[0] {
				continue
			}
			reason, err := f.askValidated(reasonLabel, func(value string) error {
				if value == "" {
					return fmt.Errorf("give a reason to record with the decision")
				}
				return nil
			})
			if err != nil {
				return nil, false, err
			}
			decisions = append(decisions, project.UpdateDecision{Source: source.Name, Rule: row.ID, Keep: row.Change != imports.UpdateNew, Reason: reason})
		}
	}
	f.introduction += "\n"
	answer, err := askChoice(f, "Apply the update?", [2]string{"yes", "no"})
	return decisions, answer == "yes", err
}

// askChoice asks question until the answer is one of the two choices, or its first letter, ignoring case.
func askChoice(f *authoringFlags, question string, choices [2]string) (string, error) {
	var chosen string
	_, err := f.askValidated(fmt.Sprintf("%s [%s/%s]:", question, choices[0], choices[1]), func(value string) error {
		for _, choice := range choices {
			if strings.EqualFold(value, choice) || strings.EqualFold(value, choice[:1]) {
				chosen = choice
				return nil
			}
		}
		return fmt.Errorf("answer %s or %s", choices[0], choices[1])
	})
	return chosen, err
}

// updateReport shows the preview and, once applied, the files the update changed. A preview explains how to
// apply it; cancelled says the user declined to.
func updateReport(result project.UpdateResult, cancelled bool) commandReport {
	var out strings.Builder
	formatUpdatePreview(&out, result.Sources)
	if result.Applied {
		out.WriteByte('\n')
		out.WriteString(projectChangesReport("update", result.FileChanges).human)
		return commandReport{value: result, human: out.String()}
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning)
	}
	switch {
	case cancelled:
		out.WriteString("\nUpdate cancelled. No files were written.\n")
	case result.Moves():
		out.WriteString("\nThis is a preview; no files were written. To apply it, run the command again\nwith --yes, or in a terminal to answer each question and confirm.\n")
	default:
		out.WriteString("\nNo rule updates are available. No files were written.\n")
	}
	return commandReport{value: result, human: out.String()}
}

// updateKindWidth fits the longest change name, replaced.
const updateKindWidth = len(imports.UpdateReplaced)

// formatUpdatePreview lists each source's rows with the change, rule ID, and versions in aligned columns, and
// each row's details indented beneath them.
func formatUpdatePreview(out *strings.Builder, sources []imports.SourceUpdate) {
	idWidth := 0
	for _, source := range sources {
		for _, row := range source.Rules {
			idWidth = max(idWidth, len(row.ID))
		}
	}
	indent := strings.Repeat(" ", 2+updateKindWidth+2)
	for _, source := range sources {
		out.WriteString(source.Name + "\n")
		switch {
		case source.Ref != "":
			fmt.Fprintf(out, "  Imports %s with ref, so update doesn't move it.\n", source.Ref)
		case len(source.Rules) == 0:
			out.WriteString("  No rule updates.\n")
		}
		for _, row := range source.Rules {
			fmt.Fprintf(out, "  %-*s  %-*s  %s\n", updateKindWidth, row.Change, idWidth, row.ID, updateVersions(row))
			for _, line := range updateDetails(row) {
				fmt.Fprintf(out, "%s%s\n", indent, line)
			}
		}
	}
}

// updateVersions shows where a row's rule moves, or the one version it has.
func updateVersions(row imports.RuleUpdate) string {
	switch {
	case row.Change == imports.UpdateNew:
		return row.To.String()
	case row.Change == imports.UpdateRetired && *row.From != *row.LastVersion:
		return row.From.String() + ", last version " + row.LastVersion.String()
	case row.To == nil:
		return row.From.String()
	}
	return row.From.String() + " -> " + row.To.String()
}

// updateDetails returns the lines shown under a row: its replacement, summaries, local rule, pin, and decision.
func updateDetails(row imports.RuleUpdate) []string {
	lines := []string{}
	if row.ReplacedBy != "" {
		lines = append(lines, "Replaced by "+row.ReplacedBy+".")
	}
	lines = append(lines, row.Summaries...)
	if row.LocalRule != "" {
		lines = append(lines, "Your rule: "+row.LocalRule+".")
	}
	switch {
	case row.Change == imports.UpdatePinned:
		lines = append(lines, "Newest version: "+row.Newest.String()+".", "Reason: "+row.Pin.Reason)
	case row.Pin != nil:
		lines = append(lines, "Your pin keeps it at "+row.Pin.Version.String()+".", "Reason: "+row.Pin.Reason)
	case row.Decision == "keep":
		lines = append(lines, "Kept at "+row.From.String()+" by a new pin.", "Reason: "+row.Reason)
	case row.Decision == "exclude":
		lines = append(lines, "Excluded by a new exclusion.", "Reason: "+row.Reason)
	}
	return lines
}
