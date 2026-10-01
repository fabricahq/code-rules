// Collect a project update's scope, decisions, and confirmation, then preview or apply it through project.PlanUpdate.

package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/spf13/cobra"
)

// projectUpdateCommand previews newer rule versions and applies them after confirmation or with --yes.
func projectUpdateCommand(options Options, output *commandOutput) *cobra.Command {
	cmd := &cobra.Command{Use: "update [SOURCE | SOURCE:RULE ...]", Short: "Preview and apply newer rule versions from libraries", Args: cobra.ArbitraryArgs}
	cmd.Long = "Preview newer rule versions, new rules, and retirements from your libraries, then\napply them after you confirm. With no arguments, every source is updated.\nSOURCE, such as team, updates one library. SOURCE:RULE, such as\nteam:techs/go/errors, moves only that rule and adds no new rules. Sources that\nuse ref don't move.\n\nIn a terminal, update asks about each major change and retirement (adopt it, or\nkeep the current version with a pin) and each new rule (add it, or exclude it),\nthen asks you to confirm. It also asks whether each local rule that replaces a\nlibrary rule now incorporates the library's changes it lists (mark them\nincorporated, or review them later). Without a terminal, or with --json, it\nonly shows the preview unless you pass --yes. The update applies exactly the\nversions the preview showed." + documentationHelp
	f := &authoringFlags{command: cmd, values: map[string]*singleString{}, directory: options.Directory}
	cmd.PostRunE = f.finishPrompts
	cmd.Flags().Bool("yes", false, "Apply the previewed changes without asking; required without a terminal or with --json")
	var keep, exclude, incorporated []string
	cmd.Flags().StringArrayVar(&keep, "keep", nil, "Pin `SOURCE:RULE` at its current version instead of updating it (repeat); requires --reason")
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, "Exclude the new rule `SOURCE:RULE` instead of adding it (repeat); requires --reason")
	cmd.Flags().StringArrayVar(&incorporated, "incorporated", nil, "Record that your local rule replacing `SOURCE:RULE` incorporates the listed changes, setting its basedOn to the newest version (repeat)")
	f.add(cmd, "reason", "Reason recorded with each pin and exclusion the update writes")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		targets, err := updateTargets(args)
		if err != nil {
			return usage(err)
		}
		decisions, err := flagDecisions(keep, exclude, incorporated, f.value("reason"))
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
			output.report = updateReport(preview, false, false, directory, options.Directory)
			return nil
		}
		// A terminal shows the preview once, before the questions, so the report after them doesn't repeat it.
		asked := interactive && preview.Moves()
		if asked {
			flagged := len(decisions)
			if decisions, err = askUpdateDecisions(f, preview, decisions); err != nil {
				return err
			}
			// Answers change the update, so people confirm what they'll get.
			if len(decisions) > flagged {
				if preview, err = plan.Preview(decisions); err != nil {
					return err
				}
				f.introduction += "\n" + answersSummary(preview.Sources)
			}
			f.introduction += "\n"
			answer, err := askChoice(f, "Apply the update?", [2]string{"yes", "no"})
			if err != nil {
				return err
			}
			if answer != "yes" {
				output.report = updateReport(preview, true, true, directory, options.Directory)
				return nil
			}
		}
		result, err := plan.Apply(cmd.Context(), decisions)
		if err != nil {
			return err
		}
		output.report = updateReport(result, false, asked, directory, options.Directory)
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

// flagDecisions turns --keep and --exclude into decisions with --reason, and --incorporated into decisions without
// one, ignoring repeated values. --reason is required with --keep or --exclude and accepted only with one of them,
// and two of the flags naming one rule fail.
func flagDecisions(keep, exclude, incorporated []string, reason string) ([]project.UpdateDecision, error) {
	if len(keep)+len(exclude) > 0 && reason == "" {
		return nil, fmt.Errorf("--keep and --exclude require --reason, which is recorded with each pin and exclusion")
	}
	if reason != "" && len(keep)+len(exclude) == 0 {
		return nil, fmt.Errorf("--reason requires --keep or --exclude")
	}
	decisions := []project.UpdateDecision{}
	seen := map[string]bool{}
	// decided maps each SOURCE:RULE to the flag that decided it, so two flags can't decide one rule.
	decided := map[string]string{}
	for _, group := range []struct {
		flag   string
		kind   project.UpdateDecisionKind
		values []string
	}{{"--keep", project.DecisionKeep, keep}, {"--exclude", project.DecisionExclude, exclude}, {"--incorporated", project.DecisionIncorporated, incorporated}} {
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
			if other, ok := decided[value]; ok {
				return nil, fmt.Errorf("%s and %s both name %s; decide each rule once", other, group.flag, value)
			}
			decided[value] = group.flag
			seen[group.flag+value] = true
			decision := project.UpdateDecision{Source: source, Rule: rule, Kind: group.kind, Reason: reason}
			if group.kind == project.DecisionIncorporated {
				decision.Reason = ""
			}
			decisions = append(decisions, decision)
		}
	}
	return decisions, nil
}

// askUpdateDecisions shows the preview and asks about each major change, retirement, new rule, and replaced rule the
// flags didn't decide. It returns the flags' decisions followed by one for each rule kept, excluded, or marked
// incorporated. The preview stays in f.introduction when there is nothing to ask.
func askUpdateDecisions(f *authoringFlags, preview project.UpdateResult, decisions []project.UpdateDecision) ([]project.UpdateDecision, error) {
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
			case imports.UpdateReplaced:
				choices = [2]string{"later", "incorporated"}
				context = fmt.Sprintf("%s: replaced by %s, with library changes up to %s.", name, row.LocalRule, row.ReviewedVersion())
				question = "Review them later, or mark them incorporated?"
			default:
				continue
			}
			f.introduction += "\n" + context + "\n"
			answer, err := askChoice(f, question, choices)
			if err != nil {
				return nil, err
			}
			if answer == choices[0] {
				continue
			}
			if row.Change == imports.UpdateReplaced {
				decisions = append(decisions, project.UpdateDecision{Source: source.Name, Rule: row.ID, Kind: project.DecisionIncorporated})
				continue
			}
			reason, err := f.askValidated(reasonLabel, func(value string) error {
				if value == "" {
					return fmt.Errorf("give a reason to record with the decision")
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			kind := project.DecisionKeep
			if row.Change == imports.UpdateNew {
				kind = project.DecisionExclude
			}
			decisions = append(decisions, project.UpdateDecision{Source: source.Name, Rule: row.ID, Kind: kind, Reason: reason})
		}
	}
	return decisions, nil
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
		return fmt.Errorf("%q isn't one of the answers; answer %s or %s, or %s or %s for short", value, choices[0], choices[1], choices[0][:1], choices[1][:1])
	})
	return chosen, err
}

// answersSummary lists the rows the user's answers decided: each kept rule with the version a new pin keeps it
// at, and each excluded new rule, with the reasons, and each replacement marked incorporated, with its new basedOn.
func answersSummary(sources []imports.SourceUpdate) string {
	var out strings.Builder
	out.WriteString("Your answers:\n")
	for _, source := range sources {
		for _, row := range source.Rules {
			switch row.Decision {
			case "keep":
				fmt.Fprintf(&out, "  Keep %s:%s at %s.\n    Reason: %s\n", source.Name, row.ID, row.From, row.Reason)
			case "exclude":
				fmt.Fprintf(&out, "  Exclude %s:%s.\n    Reason: %s\n", source.Name, row.ID, row.Reason)
			case "incorporated":
				fmt.Fprintf(&out, "  Mark %s:%s incorporated: your rule is based on %s.\n", source.Name, row.ID, row.ReviewedVersion())
			}
		}
	}
	return out.String()
}

// updateReport shows the preview, unless shown says a terminal already showed it, and, once applied, the files
// the update changed in the project at root, relative to workdir. A preview explains how to apply it; cancelled
// says the user declined to.
func updateReport(result project.UpdateResult, cancelled, shown bool, root, workdir string) commandReport {
	var out strings.Builder
	if !shown {
		formatUpdatePreview(&out, result.Sources)
	}
	// An update that applied nothing reads like a preview with nothing to apply.
	nothing := result.Applied && !result.Moves() && len(result.Added)+len(result.Changed)+len(result.Removed) == 0
	if result.Applied && !nothing {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(projectChangesReport("update", result.FileChanges, root, workdir).human)
		return commandReport{value: result, human: out.String()}
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning)
	}
	switch {
	case cancelled:
		out.WriteString("Update cancelled. No files were written.\n")
	case result.Moves() && !nothing:
		out.WriteString("\nThis is a preview; no files were written. To apply it, run the command again\nwith --yes, or in a terminal to answer each question and confirm.\n")
	case out.Len() > 0:
		out.WriteString("\nNo rule updates are available. No files were written.\n")
	default:
		out.WriteString("No rule updates are available. No files were written.\n")
	}
	return commandReport{value: result, human: out.String()}
}

// updateKindWidth fits the longest change name, replaced.
const updateKindWidth = len(imports.UpdateReplaced)

// formatUpdatePreview lists each source's rows with the change, rule ID, and versions in aligned columns, each
// row's details indented beneath them, and any move of its shared files. It writes nothing when no source has
// anything to show, so the caller's summary stands alone.
func formatUpdatePreview(out *strings.Builder, sources []imports.SourceUpdate) {
	idWidth := 0
	listed := false
	for _, source := range sources {
		for _, row := range source.Rules {
			idWidth = max(idWidth, len(row.ID))
		}
		listed = listed || len(source.Rules) > 0 || source.Ref != "" || source.SharedFiles != nil
	}
	if !listed {
		return
	}
	indent := strings.Repeat(" ", 2+updateKindWidth+2)
	for _, source := range sources {
		out.WriteString(source.Name + "\n")
		switch {
		case source.Ref != "":
			fmt.Fprintf(out, "  Imports %s with ref, so update doesn't move it.\n", source.Ref)
		case len(source.Rules) == 0 && source.SharedFiles == nil:
			out.WriteString("  No rule updates.\n")
		}
		for _, row := range source.Rules {
			fmt.Fprintf(out, "  %-*s  %-*s  %s\n", updateKindWidth, row.Change, idWidth, row.ID, updateVersions(row))
			for _, line := range updateDetails(source.Name, row) {
				fmt.Fprintf(out, "%s%s\n", indent, line)
			}
		}
		if shared := source.SharedFiles; shared != nil {
			fmt.Fprintf(out, "  Shared files: release %d -> %d\n", shared.From, shared.To)
		}
	}
}

// updateVersions shows where a row's rule moves, or the one version it has.
func updateVersions(row imports.RuleUpdate) string {
	switch {
	case row.From != nil && row.To != nil && *row.To == *row.From:
		return row.From.String()
	case row.Change == imports.UpdateNew:
		return row.To.String()
	case row.Change == imports.UpdateRetired && *row.From != *row.LastVersion:
		return row.From.String() + ", last version " + row.LastVersion.String()
	case row.To == nil:
		return row.From.String()
	}
	return row.From.String() + " -> " + row.To.String()
}

// updateDetails returns the lines shown under a row of source: its replacement, summaries, local rule, pin, and
// decision.
func updateDetails(source string, row imports.RuleUpdate) []string {
	lines := []string{}
	switch {
	case row.ReplacementLocalRule != "" && row.ReplacementRetired:
		lines = append(lines, "Replaced by "+row.ReplacedBy+", which the library also retired, in favor of "+row.CurrentReplacement+", which your project already replaces with "+row.ReplacementLocalRule+".")
	case row.ReplacementLocalRule != "":
		lines = append(lines, "Replaced by "+row.ReplacedBy+", which your project already replaces with "+row.ReplacementLocalRule+".")
	case row.ReplacementRetired && row.CurrentReplacement != "":
		lines = append(lines, "Replaced by "+row.ReplacedBy+", which the library also retired, in favor of "+row.CurrentReplacement+".")
	case row.ReplacementRetired:
		lines = append(lines, "Replaced by "+row.ReplacedBy+", which the library also retired without a replacement.")
	case row.ReplacedBy != "":
		lines = append(lines, "Replaced by "+row.ReplacedBy+".")
	}
	// A row spanning several versions labels each summary with its version, so people can tell which change is which.
	spans := slices.ContainsFunc(row.SummaryVersions, func(version rules.RuleVersion) bool { return version != row.SummaryVersions[0] })
	for i, summary := range row.Summaries {
		if spans && i < len(row.SummaryVersions) {
			summary = row.SummaryVersions[i].String() + ": " + summary
		}
		lines = append(lines, summary)
	}
	switch {
	case row.BasedOn != nil && row.Decision == "incorporated":
		lines = append(lines, "Your rule: "+row.LocalRule+", based on "+row.BasedOn.String()+".")
	case row.BasedOn != nil:
		lines = append(lines, "Your rule: "+row.LocalRule+", based on "+row.BasedOn.String()+".",
			"Changes since "+row.BasedOn.String()+", the version your rule is based on. When it has them,", "record that with --incorporated "+source+":"+row.ID+".")
	case row.LocalRule != "":
		// Without basedOn, the row can only compare with the imported version.
		lines = append(lines, "Your rule: "+row.LocalRule+".", "Changes since the imported version "+row.From.String()+"; your rule may already have some.")
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
	case row.Decision == "incorporated":
		lines = append(lines, "Marked incorporated: your rule is now based on "+row.ReviewedVersion().String()+".")
	}
	return lines
}
