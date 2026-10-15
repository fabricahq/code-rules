// Register project add skill, which forks a skill a source installs through project.PlanSkillFork.

package cli

import (
	"fmt"
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/spf13/cobra"
)

// projectSkillCommand registers project add skill. It only forks: an authored skill is written by hand.
func projectSkillCommand(options Options, output *commandOutput) *cobra.Command {
	skill, sf := newAuthoringCommand("skill --from SOURCE:SKILL", "Fork a skill a source installs, to replace it with your own", skillForkArguments, options.Directory)
	skill.Long = skill.Short + "\n\nCopies the skill from vendor/, exactly as the source's last sync recorded it,\nwith every file's executable bit, into .code-rules/skills/<name>/, where <name>\nis the name agents already know the skill by. config.yaml gets an entry in the\nsource's skills.exclude that names the fork as the skill's replacement, records\nthe commit it copied as basedOn, and records your reason, so the fork installs\nin place of the source's skill and code-rules project update lists the source's\nlater changes to it.\n\nTo write a new skill instead, create .code-rules/skills/<name>/SKILL.md." + documentationHelp
	sf.add(skill, "from", "The skill to fork, as `SOURCE:SKILL`: a configured source and the skill's ID in it, such as fabrica:skills/fix-bug")
	sf.add(skill, "reason", "Why the project uses the fork instead of the source's skill")
	skill.RunE = func(cmd *cobra.Command, _ []string) error {
		if !cmd.Flags().Changed("from") {
			return usage(fmt.Errorf("--from is required: name the skill to fork as SOURCE:SKILL, such as fabrica:skills/fix-bug. To write a new skill instead, create .code-rules/skills/<name>/SKILL.md"))
		}
		from, err := project.ParseSkillForkSource(sf.value("from"))
		if err != nil {
			return usage(err)
		}
		target, err := sf.options(cmd.Context(), false)
		if err != nil {
			return err
		}
		plan, err := project.PlanSkillFork(cmd.Context(), from, target)
		if err != nil {
			return err
		}
		reason := sf.value("reason")
		if !cmd.Flags().Changed("reason") {
			if !sf.interactive() {
				return usage(fmt.Errorf("--reason is required: the fork replaces %s's %s, and the exclusion that makes it the replacement records why", from.Source, from.Skill))
			}
			sf.introduction = fmt.Sprintf("Forking %s into:\n  .code-rules/skills/%s/\n\nThe fork replaces the source's skill: config.yaml gets an exclusion that names\nthe fork and records your reason.\n\n", from, plan.Name())
			if reason, err = sf.askValidated("Why use the fork instead of the source's skill?", func(value string) error { return validateAnswer("reason", value) }); err != nil {
				return err
			}
		}
		result, err := plan.Commit(cmd.Context(), reason)
		if err != nil {
			return err
		}
		output.report = skillForkedReport(result, from, plan, sf.directory)
		return nil
	}
	return skill
}

// skillForkArguments refuses a skill name, because project add skill only forks.
func skillForkArguments(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fmt.Errorf("project add skill takes no arguments: it forks the skill --from names, keeping its name. To write a new skill instead, create .code-rules/skills/%s/SKILL.md.\n\nUsage:\n  %s", args[0], cmd.UseLine())
}

// skillForkedReport lists the fork's files and explains what the fork replaced and how to install it.
func skillForkedReport(result project.AuthoringResult, from project.SkillForkSource, plan *project.SkillForkPlan, workdir string) commandReport {
	var out strings.Builder
	fmt.Fprintf(&out, "Skill forked from %s at commit %s.\n", from, shortCommit(plan.BasedOn()))
	formatAuthored(&out, result.Added, result.Changed, result.Warnings, workdir)
	instruction := "Next: Edit the forked skill to change its steps. A fork has no version: it changes\nonly when you edit it. The source's license still applies to the copied files." +
		"\nconfig.yaml now excludes " + from.String() + " and names skills/" + plan.Name() + " as its replacement,\nbased on commit " + shortCommit(plan.BasedOn()) + "; code-rules project update lists the source's later changes to it."
	steps := []nextStep{{Instruction: instruction}, {Instruction: "Then install the fork in place of the source's skill:", Commands: []string{"code-rules project build", "code-rules project check"}}}
	return authoredReport(&out, result.Added, result.Changed, result.Warnings, steps)
}
