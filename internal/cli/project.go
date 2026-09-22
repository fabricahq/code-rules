// Register the supported commands for consuming projects.

package cli

import (
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/spf13/cobra"
)

// newProjectCommand groups project operations with their flags and help sections.
func newProjectCommand(options Options, output *commandOutput) *cobra.Command {
	command := &cobra.Command{Use: "project", Short: "Manage rules for this project"}
	command.Long = command.Short + "\n\nCode Rules stores this project's configuration and rules in the .code-rules/\ndirectory at its root. Rules can be project-only, imported from libraries, or both.\nRun project commands from the project root." + documentationHelp
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
			projectOptions := project.Options{Directory: options.Directory, ToolVersion: options.Version}
			var changes project.FileChanges
			var err error
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
	addProjectAuthoringCommands(command, options, output)
	for _, child := range command.Commands() {
		child.GroupID = "main"
		if child.Name() == "build" || child.Name() == "check" {
			child.GroupID = "utility"
		}
	}

	return command
}
