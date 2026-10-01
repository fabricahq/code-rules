// Collect a fork's library, version, and reason, then fork the rule through project.PlanFork.

package cli

import (
	"fmt"
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/spf13/cobra"
)

// forkRuleOptions are the options of project add rule that write a new rule, which a fork copies from the library.
var forkRuleOptions = []string{"title", "when-to-read", "impact", "impact-description", "body-file"}

// forkRule forks rule id from the library and version that --from names. It asks for the reason in a terminal when
// the project imports the rule from that library and --reason is missing.
func forkRule(cmd *cobra.Command, id string, f *authoringFlags, options Options, output *commandOutput) error {
	for _, name := range forkRuleOptions {
		if cmd.Flags().Changed(name) {
			return usage(fmt.Errorf("--%s doesn't apply to a fork, which copies the library rule's metadata and text; omit it, or omit --from to write a new rule", name))
		}
	}
	from, err := project.ParseForkSource(f.value("from"))
	if err != nil {
		return usage(err)
	}
	target, err := f.options(cmd.Context(), false)
	if err != nil {
		return err
	}
	plan, err := project.PlanFork(cmd.Context(), id, from, target, options.Git)
	if err != nil {
		return err
	}
	reason, given := f.value("reason"), cmd.Flags().Changed("reason")
	switch {
	case plan.Replaces() == "" && given:
		return usage(fmt.Errorf("--reason applies only when the project imports the rule from the library it forks, and no source imports %s from %s; omit --reason", id, from.Library))
	case plan.Replaces() != "" && !given:
		if !f.interactive() {
			return usage(fmt.Errorf("--reason is required: the project imports %s from %s, so the fork replaces it, and the exclusion that makes it the replacement records why", id, plan.Replaces()))
		}
		f.introduction = fmt.Sprintf("Forking %s %s from %s into:\n  .code-rules/local/%s.md\n\nThe project imports this rule from %s, so the fork replaces it: config.yaml\ngets an exclusion that names the fork and records your reason.\n\n", id, from.Version, from.Library, id, plan.Replaces())
		if reason, err = f.askValidated("Why use the fork instead of the imported rule?", func(value string) error { return validateAnswer("reason", value) }); err != nil {
			return err
		}
	}
	result, err := plan.Commit(cmd.Context(), reason)
	if err != nil {
		return err
	}
	output.report = ruleForkedReport(result, id, from, plan, f.directory)
	return nil
}

// ruleForkedReport lists the fork's files and explains what the fork replaced and how to build it.
func ruleForkedReport(result project.AuthoringResult, id string, from project.ForkSource, plan *project.ForkPlan, workdir string) commandReport {
	var out strings.Builder
	fmt.Fprintf(&out, "Rule forked from %s %s, published in library release %d.\n", from.Library, from.Version, plan.Release())
	formatAuthored(&out, result.Added, result.Changed, result.Warnings, workdir)
	instruction := "Next: Edit the forked rule to change what it says. A fork has no version: it changes\nonly when you edit it. The library's license still applies to the copied text."
	if source := plan.Replaces(); source != "" {
		instruction += "\nconfig.yaml now excludes " + source + "'s " + id + " and names the fork as its replacement,\nbased on " + from.Version.String() + "; code-rules project update lists the library's later changes to it."
	}
	steps := []nextStep{{Instruction: instruction}, {Instruction: "Then rebuild this project's guidance:", Commands: []string{"code-rules project build", "code-rules project check"}}}
	return authoredReport(&out, result.Added, result.Changed, result.Warnings, steps)
}
