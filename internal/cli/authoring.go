// Register authoring commands and collect missing terminal inputs before filesystem operations.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/authoring"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/spf13/cobra"
)

// authoringFlags owns one command's scalar flags and resolves paths against the caller's directory.
type authoringFlags struct {
	command   *cobra.Command
	values    map[string]*singleString
	directory string
}

// newAuthoringCommand registers shared configuration and noninteractive flags with strict positional arity.
func newAuthoringCommand(use, description string, arity int, directory string) (*cobra.Command, *authoringFlags) {
	cmd := &cobra.Command{Use: use, Short: description, Args: cobra.ExactArgs(arity)}
	flags := &authoringFlags{command: cmd, values: map[string]*singleString{}, directory: directory}
	flags.add(cmd, "config", "Configuration file (default .code-rules/config.json)")
	cmd.Flags().Bool("non-interactive", false, "Require explicit flags; never prompt")
	return cmd, flags
}

// add registers one scalar option that refuses accidental repetition.
func (f *authoringFlags) add(cmd *cobra.Command, name, description string) {
	value := &singleString{}
	f.values[name] = value
	cmd.Flags().Var(value, name, description)
}

// value returns an option or empty text when omitted.
func (f *authoringFlags) value(name string) string {
	if value := f.values[name]; value != nil {
		return value.value
	}
	return ""
}

// require diagnoses absent command inputs before an operation can write files.
func (f *authoringFlags) require(names ...string) error {
	for _, name := range names {
		if f.value(name) == "" {
			value, err := f.ask(f.command.Flags().Lookup(name).Usage + " (--" + name + "):")
			if err != nil {
				return err
			}
			if value == "" {
				return fmt.Errorf("provide --%s", name)
			}
			f.values[name].value = value
		}
	}
	return nil
}

// file resolves user paths without changing process working directory.
func (f *authoringFlags) file(name string) string {
	value := f.value(name)
	if !filepath.IsAbs(value) {
		value = filepath.Join(f.directory, value)
	}
	return value
}

// options supplies the configured path or its documented default.
func (f *authoringFlags) options() authoring.Options {
	if f.value("config") == "" {
		return authoring.Options{ConfigPath: filepath.Join(f.directory, ".code-rules", "config.json")}
	}
	return authoring.Options{ConfigPath: f.file("config")}
}

// group returns the explicit metadata for an existing or newly created group.
func (f *authoringFlags) group(prefix string) rules.GroupMetadata {
	return rules.GroupMetadata{Name: f.value(prefix + "name"), Description: f.value(prefix + "description"), WhenToRead: f.value(prefix + "when-to-read")}
}

// addGroupFlags defines the three required group fields with an optional creation prefix.
func (f *authoringFlags) addGroupFlags(cmd *cobra.Command, prefix string) {
	f.add(cmd, prefix+"name", "Group display name")
	f.add(cmd, prefix+"description", "Group scope")
	f.add(cmd, prefix+"when-to-read", "When an agent should read this group")
}

// addProjectAuthoringCommands installs project initialization, source configuration, and local authoring.
func addProjectAuthoringCommands(root *cobra.Command, options Options, started *bool) {
	initialize, f := newAuthoringCommand("init", "Initialize project configuration and local orientation", 0, options.Directory)
	initialize.RunE = func(cmd *cobra.Command, _ []string) error {
		*started = true
		result, err := authoring.InitializeProject(cmd.Context(), f.options())
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), result)
	}
	root.AddCommand(initialize)
	add := &cobra.Command{Use: "add", Short: "Add project configuration"}
	root.AddCommand(add)
	source, sf := newAuthoringCommand("source ALIAS", "Record one Git source without fetching", 1, options.Directory)
	for name, description := range map[string]string{"repository": "Git repository URL", "ref": "Exact tag or full commit", "version": "HashiCorp version constraint"} {
		sf.add(source, name, description)
	}
	var groups []string
	source.Flags().StringArrayVar(&groups, "groups", nil, "Group ID (repeat) or one wildcard")
	source.RunE = func(cmd *cobra.Command, args []string) error {
		if err := sf.collectSource(&groups); err != nil {
			return err
		}
		var selection any = groups
		if len(groups) == 1 && (groups[0] == "*" || groups[0] == "techs/*" || groups[0] == "practices/*") {
			selection = groups[0]
		}
		declaration := map[string]any{"repository": sf.value("repository"), "groups": selection, "exclude": map[string]string{}, "replace": map[string]any{}}
		if sf.value("ref") != "" {
			declaration["ref"] = sf.value("ref")
		} else {
			declaration["version"] = sf.value("version")
		}
		data, err := json.Marshal(declaration)
		if err != nil {
			return err
		}
		*started = true
		result, err := authoring.AddSource(cmd.Context(), args[0], data, sf.options())
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), result)
	}
	add.AddCommand(source)
	local := &cobra.Command{Use: "local", Short: "Author local engineering rules"}
	localAdd := &cobra.Command{Use: "add", Short: "Add a local group or rule"}
	local.AddCommand(localAdd)
	root.AddCommand(local)
	group, gf := newAuthoringCommand("group ID", "Create local group metadata", 1, options.Directory)
	gf.addGroupFlags(group, "")
	group.RunE = func(cmd *cobra.Command, args []string) error {
		if err := gf.require("name", "description", "when-to-read"); err != nil {
			return err
		}
		*started = true
		result, err := authoring.AddLocalGroup(cmd.Context(), args[0], gf.group(""), gf.options())
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), result)
	}
	localAdd.AddCommand(group)
	rule, rf := newAuthoringCommand("rule ID", "Create a local rule body or canonical unfinished draft", 1, options.Directory)
	for name, description := range map[string]string{"title": "Action-oriented rule title", "when-to-read": "When an agent should read this rule", "impact": "Consequence level", "impact-description": "Why the rule matters", "body-file": "Existing UTF-8 Markdown body file"} {
		rf.add(rule, name, description)
	}
	rf.addGroupFlags(rule, "group-")
	var createGroup bool
	rule.Flags().BoolVar(&createGroup, "create-group", false, "Create missing local metadata in the same operation")
	rule.RunE = func(cmd *cobra.Command, args []string) error {
		if strings.HasSuffix(args[0], ".md") {
			return fmt.Errorf("use a rule ID without the .md extension")
		}
		if err := rf.require("title", "when-to-read", "impact", "impact-description"); err != nil {
			return err
		}
		if err := rf.offerGroup(args[0], &createGroup, false); err != nil {
			return err
		}
		ro := authoring.RuleOptions{Options: rf.options()}
		if createGroup {
			if err := rf.require("group-name", "group-description", "group-when-to-read"); err != nil {
				return err
			}
			metadata := rf.group("group-")
			ro.Group = &metadata
		} else {
			for _, name := range []string{"group-name", "group-description", "group-when-to-read"} {
				if rf.value(name) != "" {
					return fmt.Errorf("group metadata requires --create-group")
				}
			}
		}
		*started = true
		if rf.value("body-file") != "" {
			body, err := readBody(cmd.Context(), rf.file("body-file"))
			if err != nil {
				return err
			}
			ro.Body = &body
		}
		metadata := authoring.RuleMetadata{Title: rf.value("title"), WhenToRead: rf.value("when-to-read"), Impact: rf.value("impact"), ImpactDescription: rf.value("impact-description")}
		result, err := authoring.AddLocalRule(cmd.Context(), args[0], metadata, ro)
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), result)
	}
	localAdd.AddCommand(rule)
}

// readBody loads a bounded regular UTF-8 input without interpreting it as a rule document.
func readBody(ctx context.Context, name string) (string, error) {
	return readAuthoringText(ctx, name, "rule body")
}

// readAuthoringText reads bounded UTF-8 inputs without changing their original line endings or whitespace.
func readAuthoringText(ctx context.Context, name, label string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	info, err := os.Stat(name)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", label, err)
	}
	if !info.Mode().IsRegular() || info.Size() > 8*1024*1024 {
		return "", fmt.Errorf("%s must be a regular file no larger than 8 MiB", label)
	}
	file, err := os.Open(name)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", label, err)
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !actual.Mode().IsRegular() {
		return "", fmt.Errorf("%s must be a regular file", label)
	}
	data, err := io.ReadAll(io.LimitReader(file, 8*1024*1024+1))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", label, err)
	}
	if len(data) > 8*1024*1024 {
		return "", fmt.Errorf("%s exceeds 8 MiB", label)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s must contain valid UTF-8", label)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return string(data), nil
}
