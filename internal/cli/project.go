// Group consuming-project commands and preserve the original command paths for existing scripts.

package cli

import (
	"errors"
	"path/filepath"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/spf13/cobra"
)

// newProjectCommand gives each command tree its own flags and closures.
func newProjectCommand(options Options, started *bool, output *commandOutput) *cobra.Command {
	command := &cobra.Command{Use: "project", Short: "Configure and manage rules for this project"}
	for _, name := range []string{"sync", "build", "check"} {
		config := &singleString{}
		descriptions := map[string]string{"sync": "Fetch configured libraries and build guidance", "build": "Rebuild guidance using local library snapshots", "check": "Check configuration and generated guidance"}
		cmd := &cobra.Command{Use: name, Short: descriptions[name], Args: cobra.NoArgs}
		if name == "check" {
			cmd.Long = "Check configuration, generated guidance, and the project guide without changing files or fetching libraries. This checks file consistency, not whether application code follows the rules."
		}
		cmd.Flags().Var(config, "config", "Configuration file (default .code-rules/config.json)")
		cmd.RunE = func(cmd *cobra.Command, _ []string) error {
			*started = true
			path := config.value
			if path == "" {
				path = filepath.Join(".code-rules", "config.json")
			}
			if !filepath.IsAbs(path) {
				path = filepath.Join(options.Directory, path)
			}
			projectOptions := project.Options{ConfigPath: path, ToolVersion: options.Version}
			var changes project.FileChanges
			var err error
			switch name {
			case "sync":
				changes, err = project.Sync(cmd.Context(), projectOptions, options.Git)
			case "build":
				changes, err = project.Build(cmd.Context(), projectOptions)
			case "check":
				report, checkErr := checkProject(cmd.Context(), projectOptions, config.value)
				if checkErr == nil || errors.Is(checkErr, errCheckOutOfDate) {
					output.value = report
				}
				return checkErr
			}
			if err != nil {
				return err
			}
			output.value = changes
			return nil
		}
		command.AddCommand(cmd)
	}
	addProjectAuthoringCommands(command, options, started, output)

	return command
}

// addLegacyProjectCommands uses fresh command instances so compatibility paths cannot
// change the canonical tree's parents or flag state. Only the old root groups are hidden.
func addLegacyProjectCommands(root *cobra.Command, options Options, started *bool, output *commandOutput) {
	legacy := newProjectCommand(options, started, output)
	local := &cobra.Command{Use: "local", Short: "Author local engineering rules", Hidden: true}
	localAdd := &cobra.Command{Use: "add", Short: "Add a local group or rule"}
	local.AddCommand(localAdd)
	for _, command := range legacy.Commands() {
		legacy.RemoveCommand(command)
		command.Hidden = true
		if command.Name() == "add" {
			for _, child := range command.Commands() {
				if child.Name() == "library" {
					child.Use = "source ALIAS"
				} else {
					command.RemoveCommand(child)
					localAdd.AddCommand(child)
				}
			}
		}
		root.AddCommand(command)
	}
	root.AddCommand(local)
}
