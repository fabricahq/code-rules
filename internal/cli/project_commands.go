// Register the supported commands for consuming projects.

package cli

import (
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/spf13/cobra"
)

// newProjectCommand groups project operations with their flags and help sections.
func newProjectCommand(options Options, output *commandOutput) *cobra.Command {
	command := &cobra.Command{Use: "project", Short: "Manage rules for this project"}
	command.Long = command.Short + "\n\nCode Rules stores this project's configuration and rules in the .code-rules/\ndirectory at its root. Rules can be project-only, imported from libraries, or both.\nRun init from the Git repository root. Other project commands can run from any\nsubdirectory. Outside Git, run commands from the project root." + documentationHelp
	command.AddGroup(
		&cobra.Group{ID: "main", Title: "Main commands:"},
		&cobra.Group{ID: "utility", Title: "Utility commands:"},
	)
	for _, name := range []string{"sync", "build", "check"} {
		descriptions := map[string]string{"sync": "Fetch configured libraries and build guidance", "build": "Rebuild guidance using local library snapshots", "check": "Check configuration and generated guidance"}
		cmd := &cobra.Command{Use: name, Short: descriptions[name], Args: cobra.NoArgs}
		if name == "check" {
			cmd.Long = "Check project configuration, generated guidance, and the Code Rules guide without changing files or fetching libraries. This checks file consistency, not whether application code follows the rules."
		}
		cmd.RunE = func(cmd *cobra.Command, _ []string) error {
			directory, err := commandDirectory(cmd.Context(), options.Directory, "project", false)
			if err != nil {
				return err
			}
			projectOptions := project.Options{Directory: directory, ToolVersion: options.Version}
			var changes project.FileChanges
			switch name {
			case "sync":
				changes, err = project.Sync(cmd.Context(), projectOptions, options.Git)
			case "build":
				changes, err = project.Build(cmd.Context(), projectOptions)
			case "check":
				report, checkErr := checkProject(cmd.Context(), projectOptions)
				if checkErr == nil {
					output.report = projectCheckedReport(report)
				}
				return checkErr
			}
			if err != nil {
				return err
			}
			output.report = projectChangesReport(name, changes)
			return nil
		}
		command.AddCommand(cmd)
	}
	command.AddCommand(projectInitCommand(options, output))
	add := &cobra.Command{Use: "add", Short: "Add a project-only rule, project-only group, or library"}
	add.AddCommand(projectLibraryCommand(options, output), projectGroupCommand(options, output), projectRuleCommand(options, output))
	command.AddCommand(add)
	for _, child := range command.Commands() {
		child.GroupID = "main"
		if child.Name() == "build" || child.Name() == "check" {
			child.GroupID = "utility"
		}
	}

	return command
}

func projectInitCommand(options Options, output *commandOutput) *cobra.Command {
	initialize, f := newAuthoringCommand("init", "Set up Code Rules in this project", cobra.NoArgs, options.Directory)
	initialize.RunE = func(cmd *cobra.Command, _ []string) error {
		target, err := f.options(cmd.Context(), true)
		if err != nil {
			return err
		}
		result, err := project.Initialize(cmd.Context(), target)
		if err != nil {
			return err
		}
		output.report = projectInitializedReport(result)
		return nil
	}
	return initialize
}

func projectLibraryCommand(options Options, output *commandOutput) *cobra.Command {
	source, sf := newAuthoringCommand("library ALIAS", "Configure a shared library to use (without fetching)", requiredArgument("library alias", "team", "The alias is a short name for this library in your project configuration."), options.Directory)
	for name, description := range map[string]string{"repository": "Git repository URL", "ref": "Exact tag, full commit SHA, or version range (e.g. >= 1.2.0, < 2.0.0)"} {
		sf.add(source, name, description)
	}
	source.Long = librarySelectionHelp + documentationHelp
	sf.prompts = map[string]string{"repository": "Git repository URL", "ref": "Ref (tag, full commit SHA, or version range)"}
	var groups []string
	source.Flags().StringArrayVar(&groups, "groups", nil, "Library group `path` (repeat), or *, practices/*, techs/*")
	source.RunE = func(cmd *cobra.Command, args []string) error {
		target, err := sf.options(cmd.Context(), false)
		if err != nil {
			return err
		}
		plan, err := project.PlanSource(cmd.Context(), args[0], target)
		if err != nil {
			return err
		}
		sf.introduction = librarySelectionIntroduction(args[0])
		if err := sf.collectSource(&groups); err != nil {
			return err
		}
		result, err := plan.Commit(cmd.Context(), project.SourceInput{Repository: sf.value("repository"), Ref: sf.value("ref"), Groups: groups})
		if err != nil {
			return err
		}
		output.report = sourceAddedReport(result)
		return nil
	}
	return source
}

func projectGroupCommand(options Options, output *commandOutput) *cobra.Command {
	group, gf := newAuthoringCommand("group GROUP_PATH", "Create a project-only rule group", requiredArgument("group path", "practices/testing", "Use a category and group slug, such as practices/testing or techs/go."), options.Directory)
	gf.addGroupFlags(group, "")
	group.RunE = func(cmd *cobra.Command, args []string) error {
		target, err := gf.options(cmd.Context(), false)
		if err != nil {
			return err
		}
		plan, err := project.PlanLocalGroup(cmd.Context(), args[0], target)
		if err != nil {
			return err
		}
		gf.introduction = groupIntroduction(args[0], false)
		if err := gf.require("name", "description", "when-to-read"); err != nil {
			return err
		}
		result, err := plan.Commit(cmd.Context(), gf.group(""))
		if err != nil {
			return err
		}
		output.report = groupCreatedReport(result.Files, result.Warnings, args[0], authoringScope{})
		return nil
	}
	return group
}

func projectRuleCommand(options Options, output *commandOutput) *cobra.Command {
	rule, rf := newAuthoringCommand("rule RULE_PATH", "Create a project-only rule or unfinished draft", requiredArgument("rule path", "practices/testing/my-rule", "Include the group path and rule slug, without .md."), options.Directory)
	rf.addRuleFlags(rule)
	rule.RunE = func(cmd *cobra.Command, args []string) error {
		target, err := rf.options(cmd.Context(), false)
		if err != nil {
			return err
		}
		plan, err := project.PlanLocalRule(cmd.Context(), args[0], target)
		if err != nil {
			return err
		}
		metadata, body, err := rf.collectRule(cmd.Context(), args[0], false)
		if err != nil {
			return err
		}
		result, err := plan.Commit(cmd.Context(), metadata, body)
		if err != nil {
			return err
		}
		output.report = ruleCreatedReport(result.Files, result.Warnings, body == nil, authoringScope{})
		return nil
	}
	return rule
}
