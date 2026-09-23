// Expose native library initialization, authoring, and read-only validation independently of consuming projects.

package cli

import (
	"context"
	"fmt"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/spf13/cobra"
)

// newLibraryCommand registers library-specific location and explicit-input flags.
func newLibraryCommand(use, description string, args cobra.PositionalArgs, directory string) (*cobra.Command, *authoringFlags) {
	cmd := &cobra.Command{Use: use, Short: description, Args: args}
	flags := &authoringFlags{command: cmd, values: map[string]*singleString{}, directory: directory}
	cmd.PostRunE = flags.finishPrompts
	flags.add(cmd, "directory", "Library directory (default Git root, or current directory outside Git)")
	cmd.Flags().Bool("non-interactive", false, "Require explicit flags; never prompt")
	return cmd, flags
}

// libraryOptions resolves the selected target without changing relative body or license inputs.
func (f *authoringFlags) libraryOptions(ctx context.Context, initialize bool) (library.Options, error) {
	directory := f.directory
	if f.value("directory") != "" {
		directory = f.file("directory")
	}
	directory, err := commandDirectory(ctx, directory, "library", initialize)
	return library.Options{Directory: directory}, err
}

// addLibraryCommands installs a separate command tree that never reads consumer configuration.
func addLibraryCommands(root *cobra.Command, options Options, output *commandOutput) {
	library := &cobra.Command{Use: "library", Short: "Create and maintain a shared rule library"}
	library.Long = library.Short + "\n\nLibraries are maintained separately from projects. Projects can define their own rule groups and import groups from libraries.\n\nRun init from the Git repository root. Other library commands can run from any\nsubdirectory. Outside Git, run commands from the library root." + documentationHelp
	root.AddCommand(library)
	library.AddCommand(libraryInitCommand(options, output), libraryCheckCommand(options, output))
	add := &cobra.Command{Use: "add", Short: "Add a library group or rule"}
	library.AddCommand(add)
	add.AddCommand(libraryGroupCommand(options, output), libraryRuleCommand(options, output))
}

// libraryInitCommand reads explicit publisher terms before creating library-owned files.
func libraryInitCommand(options Options, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("init", "Initialize a rule library without overwriting authored files", cobra.NoArgs, options.Directory)
	for name, description := range map[string]string{"spdx": "Library SPDX expression", "license-file": "Existing UTF-8 license text", "notice-file": "Existing UTF-8 notice text"} {
		f.add(cmd, name, description)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		target, err := f.libraryOptions(cmd.Context(), true)
		if err != nil {
			return err
		}
		if (f.value("spdx") == "") != (f.value("license-file") == "") {
			return usage(fmt.Errorf("supply --spdx and --license-file together"))
		}
		if f.value("notice-file") != "" && f.value("spdx") == "" {
			return usage(fmt.Errorf("--notice-file requires --spdx and --license-file"))
		}
		var terms *library.Terms
		if f.value("spdx") != "" {
			license, err := readAuthoringText(cmd.Context(), f.file("license-file"), "license text")
			if err != nil {
				return err
			}
			terms = &library.Terms{SPDXExpression: f.value("spdx"), License: license}
			if f.value("notice-file") != "" {
				notice, err := readAuthoringText(cmd.Context(), f.file("notice-file"), "notice text")
				if err != nil {
					return err
				}
				terms.Notice = &notice
			}
		}
		result, err := library.Initialize(cmd.Context(), target, terms)
		if err != nil {
			return err
		}
		output.report = libraryInitializedReport(result, authoringScope{library: true, directory: f.value("directory")})
		return nil
	}
	return cmd
}

// libraryCheckCommand reports counts and licensing caveats without changing library files.
func libraryCheckCommand(options Options, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("check", "Validate every library group, rule, asset, and declared term", cobra.NoArgs, options.Directory)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		target, err := f.libraryOptions(cmd.Context(), false)
		if err != nil {
			return err
		}
		result, err := library.Check(cmd.Context(), target)
		if err != nil {
			return err
		}
		output.report = libraryCheckedReport(result)
		return nil
	}
	return cmd
}

// libraryGroupCommand requires all group metadata before attempting exclusive publication.
func libraryGroupCommand(options Options, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("group GROUP_PATH", "Create library group metadata", requiredArgument("group path", "practices/testing", "Use a category and group slug, such as practices/testing or techs/go."), options.Directory)
	f.addGroupFlags(cmd, "")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		target, err := f.libraryOptions(cmd.Context(), false)
		if err != nil {
			return err
		}
		plan, err := library.PlanGroup(cmd.Context(), args[0], target)
		if err != nil {
			return err
		}
		f.introduction = groupIntroduction(args[0], true)
		if err := f.require("name", "description", "when-to-read"); err != nil {
			return err
		}
		result, err := plan.Commit(cmd.Context(), f.group(""))
		if err != nil {
			return err
		}
		output.report = groupCreatedReport(result.Files, result.Warnings, args[0], authoringScope{library: true, directory: f.value("directory")})
		return nil
	}
	return cmd
}

// libraryRuleCommand creates supplied guidance or a marked canonical draft in an existing group.
func libraryRuleCommand(options Options, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("rule RULE_PATH", "Create a complete library rule or marked draft", requiredArgument("rule path", "practices/testing/my-rule", "Include the group path and rule slug, without .md."), options.Directory)
	f.addRuleFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		target, err := f.libraryOptions(cmd.Context(), false)
		if err != nil {
			return err
		}
		plan, err := library.PlanRule(cmd.Context(), args[0], target)
		if err != nil {
			return err
		}
		metadata, body, err := f.collectRule(cmd.Context(), args[0], true)
		if err != nil {
			return err
		}
		result, err := plan.Commit(cmd.Context(), metadata, body)
		if err != nil {
			return err
		}
		output.report = ruleCreatedReport(result.Files, result.Warnings, body == nil, authoringScope{library: true, directory: f.value("directory")})
		return nil
	}
	return cmd
}
