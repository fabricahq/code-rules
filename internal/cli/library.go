// Expose native library initialization, authoring, and read-only validation independently of consuming projects.

package cli

import (
	"fmt"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/spf13/cobra"
)

// newLibraryCommand registers library-specific location and explicit-input flags.
func newLibraryCommand(use, description string, args cobra.PositionalArgs, directory string) (*cobra.Command, *authoringFlags) {
	cmd := &cobra.Command{Use: use, Short: description, Args: args}
	flags := &authoringFlags{command: cmd, values: map[string]*singleString{}, directory: directory}
	cmd.PostRunE = flags.finishPrompts
	flags.add(cmd, "directory", "Library directory (default current directory)")
	cmd.Flags().Bool("non-interactive", false, "Require explicit flags; never prompt")
	return cmd, flags
}

// libraryOptions anchors the library path to the embedding caller's working directory.
func (f *authoringFlags) libraryOptions() library.Options {
	if f.value("directory") == "" {
		directory := f.directory
		if directory == "" {
			directory = "."
		}
		return library.Options{Directory: directory}
	}
	return library.Options{Directory: f.file("directory")}
}

// addLibraryCommands installs a separate command tree that never reads consumer configuration.
func addLibraryCommands(root *cobra.Command, options Options, started *bool, output *commandOutput) {
	library := &cobra.Command{Use: "library", Short: "Create and maintain a shared rule library"}
	library.Long = library.Short + "\n\nLibraries are maintained separately from projects. Projects can define their own rule groups and import groups from libraries." + documentationHelp
	root.AddCommand(library)
	library.AddCommand(libraryInitCommand(options, started, output), libraryCheckCommand(options, started, output))
	add := &cobra.Command{Use: "add", Short: "Add a library group or rule"}
	library.AddCommand(add)
	add.AddCommand(libraryGroupCommand(options, started, output), libraryRuleCommand(options, started, output))
}

// libraryInitCommand reads explicit publisher terms before creating library-owned files.
func libraryInitCommand(options Options, started *bool, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("init", "Initialize a rule library without overwriting authored files", cobra.NoArgs, options.Directory)
	for name, description := range map[string]string{"spdx": "Library SPDX expression", "license-file": "Existing UTF-8 license text", "notice-file": "Existing UTF-8 notice text"} {
		f.add(cmd, name, description)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if (f.value("spdx") == "") != (f.value("license-file") == "") {
			return fmt.Errorf("supply --spdx and --license-file together")
		}
		if f.value("notice-file") != "" && f.value("spdx") == "" {
			return fmt.Errorf("--notice-file requires --spdx and --license-file")
		}
		*started = true
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
		result, err := library.Initialize(cmd.Context(), f.libraryOptions(), terms)
		if err != nil {
			return err
		}
		output.value = result
		return nil
	}
	return cmd
}

// libraryCheckCommand reports counts and licensing caveats without changing library files.
func libraryCheckCommand(options Options, started *bool, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("check", "Validate every library group, rule, asset, and declared term", cobra.NoArgs, options.Directory)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		*started = true
		result, err := library.Check(cmd.Context(), f.libraryOptions())
		if err != nil {
			return err
		}
		output.value = result
		return nil
	}
	return cmd
}

// libraryGroupCommand requires all group metadata before attempting exclusive publication.
func libraryGroupCommand(options Options, started *bool, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("group GROUP_PATH", "Create library group metadata", requiredArgument("group path", "practices/testing", "Use a category and group slug, such as practices/testing or techs/go."), options.Directory)
	f.addGroupFlags(cmd, "")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		f.introduction = groupIntroduction(args[0])
		if err := f.require("name", "description", "when-to-read"); err != nil {
			return err
		}
		*started = true
		result, err := library.AddGroup(cmd.Context(), args[0], f.group(""), f.libraryOptions())
		if err != nil {
			return err
		}
		output.value = result
		return nil
	}
	return cmd
}

// libraryRuleCommand creates supplied guidance or a marked canonical draft in an existing group.
func libraryRuleCommand(options Options, started *bool, output *commandOutput) *cobra.Command {
	cmd, f := newLibraryCommand("rule RULE_PATH", "Create a complete library rule or marked draft", requiredArgument("rule path", "practices/testing/my-rule", "Include the group path and rule slug, without .md."), options.Directory)
	f.addRuleFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		metadata, body, err := f.collectRule(cmd.Context(), args[0], true, started)
		if err != nil {
			return err
		}
		ro := library.RuleOptions{Options: f.libraryOptions(), Body: body}
		result, err := library.AddRule(cmd.Context(), args[0], metadata, ro)
		if err != nil {
			return err
		}
		output.value = result
		return nil
	}
	return cmd
}
