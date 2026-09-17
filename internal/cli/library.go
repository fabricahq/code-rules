// Expose native library initialization, authoring, and read-only validation independently of consuming projects.

package cli

import (
	"fmt"
	"strings"

	"github.com/fabricahq/code-rules/internal/authoring"
	"github.com/spf13/cobra"
)

// newLibraryCommand registers library-specific location and explicit-input flags.
func newLibraryCommand(use, description string, arity int, directory string) (*cobra.Command, *authoringFlags) {
	cmd := &cobra.Command{Use: use, Short: description, Args: cobra.ExactArgs(arity)}
	flags := &authoringFlags{command: cmd, values: map[string]*singleString{}, directory: directory}
	flags.add(cmd, "directory", "Library directory (default current directory)")
	cmd.Flags().Bool("non-interactive", false, "Require explicit flags; never prompt")
	return cmd, flags
}

// libraryOptions anchors the library path to the embedding caller's working directory.
func (f *authoringFlags) libraryOptions() authoring.LibraryOptions {
	if f.value("directory") == "" {
		directory := f.directory
		if directory == "" {
			directory = "."
		}
		return authoring.LibraryOptions{Directory: directory}
	}
	return authoring.LibraryOptions{Directory: f.file("directory")}
}

// addLibraryCommands installs a separate command tree that never reads consumer configuration.
func addLibraryCommands(root *cobra.Command, options Options, started *bool) {
	library := &cobra.Command{Use: "library", Short: "Author and validate a shared rule library"}
	root.AddCommand(library)
	library.AddCommand(libraryInitCommand(options, started), libraryCheckCommand(options, started))
	add := &cobra.Command{Use: "add", Short: "Add a library group or rule"}
	library.AddCommand(add)
	add.AddCommand(libraryGroupCommand(options, started), libraryRuleCommand(options, started))
}

// libraryInitCommand reads explicit publisher terms before creating library-owned files.
func libraryInitCommand(options Options, started *bool) *cobra.Command {
	cmd, f := newLibraryCommand("init", "Initialize a rule library without overwriting authored files", 0, options.Directory)
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
		var terms *authoring.LibraryTerms
		if f.value("spdx") != "" {
			license, err := readAuthoringText(cmd.Context(), f.file("license-file"), "license text")
			if err != nil {
				return err
			}
			terms = &authoring.LibraryTerms{SPDXExpression: f.value("spdx"), License: license}
			if f.value("notice-file") != "" {
				notice, err := readAuthoringText(cmd.Context(), f.file("notice-file"), "notice text")
				if err != nil {
					return err
				}
				terms.Notice = &notice
			}
		}
		result, err := authoring.InitializeLibrary(cmd.Context(), f.libraryOptions(), terms)
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), result)
	}
	return cmd
}

// libraryCheckCommand reports counts and licensing caveats without changing library files.
func libraryCheckCommand(options Options, started *bool) *cobra.Command {
	cmd, f := newLibraryCommand("check", "Validate every library group, rule, asset, and declared term", 0, options.Directory)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		*started = true
		result, err := authoring.CheckLibrary(cmd.Context(), f.libraryOptions())
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), result)
	}
	return cmd
}

// libraryGroupCommand requires all group metadata before attempting exclusive publication.
func libraryGroupCommand(options Options, started *bool) *cobra.Command {
	cmd, f := newLibraryCommand("group ID", "Create library group metadata", 1, options.Directory)
	f.addGroupFlags(cmd, "")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := f.require("name", "description", "when-to-read"); err != nil {
			return err
		}
		*started = true
		result, err := authoring.AddLibraryGroup(cmd.Context(), args[0], f.group(""), f.libraryOptions())
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), result)
	}
	return cmd
}

// libraryRuleCommand creates supplied guidance or a marked canonical draft, optionally with new group metadata.
func libraryRuleCommand(options Options, started *bool) *cobra.Command {
	cmd, f := newLibraryCommand("rule ID", "Create a complete library rule or marked draft", 1, options.Directory)
	for name, description := range map[string]string{"title": "Action-oriented rule title", "when-to-read": "When to read this rule", "impact": "Consequence level", "impact-description": "Why this rule matters", "body-file": "Existing UTF-8 Markdown body"} {
		f.add(cmd, name, description)
	}
	f.addGroupFlags(cmd, "group-")
	var createGroup bool
	cmd.Flags().BoolVar(&createGroup, "create-group", false, "Create group metadata with this rule")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if strings.HasSuffix(args[0], ".md") {
			return fmt.Errorf("use a rule ID without the .md extension")
		}
		if err := f.require("title", "when-to-read", "impact", "impact-description"); err != nil {
			return err
		}
		if err := f.offerGroup(args[0], &createGroup, true); err != nil {
			return err
		}
		ro := authoring.LibraryRuleOptions{LibraryOptions: f.libraryOptions()}
		if createGroup {
			if err := f.require("group-name", "group-description", "group-when-to-read"); err != nil {
				return err
			}
			group := f.group("group-")
			ro.Group = &group
		} else {
			for _, name := range []string{"group-name", "group-description", "group-when-to-read"} {
				if f.value(name) != "" {
					return fmt.Errorf("group metadata requires --create-group")
				}
			}
		}
		*started = true
		if f.value("body-file") != "" {
			body, err := readBody(cmd.Context(), f.file("body-file"))
			if err != nil {
				return err
			}
			ro.Body = &body
		}
		metadata := authoring.RuleMetadata{Title: f.value("title"), WhenToRead: f.value("when-to-read"), Impact: f.value("impact"), ImpactDescription: f.value("impact-description")}
		result, err := authoring.AddLibraryRule(cmd.Context(), args[0], metadata, ro)
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), result)
	}
	return cmd
}
