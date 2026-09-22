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

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/spf13/cobra"
)

// authoringFlags owns one command's scalar flags and resolves paths against the caller's directory.
type authoringFlags struct {
	command   *cobra.Command
	values    map[string]*singleString
	prompts   map[string]string
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
			label := f.command.Flags().Lookup(name).Usage
			if prompt, ok := f.prompts[name]; ok {
				label = prompt
			}
			value, err := f.ask(label + ":")
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
func (f *authoringFlags) options() project.Options {
	if f.value("config") == "" {
		return project.Options{ConfigPath: filepath.Join(f.directory, ".code-rules", "config.json")}
	}
	return project.Options{ConfigPath: f.file("config")}
}

// group returns the explicit metadata for an existing or newly created group.
func (f *authoringFlags) group(prefix string) rules.GroupMetadata {
	return rules.GroupMetadata{Name: f.value(prefix + "name"), Description: f.value(prefix + "description"), WhenToRead: f.value(prefix + "when-to-read")}
}

// addGroupFlags defines the three required group fields with an optional creation prefix.
func (f *authoringFlags) addGroupFlags(cmd *cobra.Command, prefix string) {
	f.add(cmd, prefix+"name", `Title shown in rule indexes and group pages (e.g. "Testing")`)
	f.add(cmd, prefix+"description", `What this group covers (e.g. "Unit and integration testing")`)
	f.add(cmd, prefix+"when-to-read", "When an agent should read this group's rules")
	f.prompts = map[string]string{
		prefix + "name":        "Group name (e.g. Testing)",
		prefix + "description": "What this group covers (e.g. Unit and integration testing)",
	}
	cmd.Long = cmd.Short + "\n\nID identifies the group in paths and configuration (e.g. practices/testing).\nThe name is its readable title (e.g. Testing or Testing and quality)."
}

// addProjectAuthoringCommands installs project initialization, source configuration, and local authoring.
func addProjectAuthoringCommands(root *cobra.Command, options Options, started *bool, output *commandOutput) {
	initialize, f := newAuthoringCommand("init", "Set up Code Rules in this project", 0, options.Directory)
	initialize.RunE = func(cmd *cobra.Command, _ []string) error {
		*started = true

		result, err := project.Initialize(cmd.Context(), f.options())
		if err != nil {
			return err
		}
		output.value = result
		return nil
	}
	root.AddCommand(initialize)
	add := &cobra.Command{Use: "add", Short: "Add a project-only rule, project-only group, or library"}
	root.AddCommand(add)
	source, sf := newAuthoringCommand("library ALIAS", "Configure a shared library to use (without fetching)", 1, options.Directory)
	for name, description := range map[string]string{"repository": "Git repository URL", "ref": "Exact tag or full commit", "version": "HashiCorp version constraint"} {
		sf.add(source, name, description)
	}
	source.Long = "Record a shared library in this project's configuration without fetching it. Run code-rules project sync to fetch configured libraries and build guidance. ALIAS names the library in the sources configuration."
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
		result, err := project.AddSource(cmd.Context(), args[0], data, sf.options())
		if err != nil {
			return err
		}
		output.value = result
		return nil
	}
	add.AddCommand(source)
	group, gf := newAuthoringCommand("group ID", "Create a project-only rule group", 1, options.Directory)
	gf.addGroupFlags(group, "")
	group.RunE = func(cmd *cobra.Command, args []string) error {
		if err := gf.require("name", "description", "when-to-read"); err != nil {
			return err
		}
		*started = true
		result, err := project.AddLocalGroup(cmd.Context(), args[0], gf.group(""), gf.options())
		if err != nil {
			return err
		}
		output.value = result
		return nil
	}
	add.AddCommand(group)
	rule, rf := newAuthoringCommand("rule ID", "Create a project-only rule or unfinished draft", 1, options.Directory)
	for name, description := range map[string]string{"title": "Action-oriented rule title", "when-to-read": "When an agent should read this rule", "impact": "Consequence level", "impact-description": "Why the rule matters", "body-file": "Existing UTF-8 Markdown body file"} {
		rf.add(rule, name, description)
	}
	rule.RunE = func(cmd *cobra.Command, args []string) error {
		metadata, body, err := rf.collectRule(cmd.Context(), args[0], false, started)
		if err != nil {
			return err
		}
		ro := project.RuleOptions{Options: rf.options(), Body: body}
		result, err := project.AddLocalRule(cmd.Context(), args[0], metadata, ro)
		if err != nil {
			return err
		}
		output.value = result
		return nil
	}
	add.AddCommand(rule)
}

// collectRule validates the existing group before prompting and collects all input before writer ownership.
// started preserves the CLI distinction between usage failures and failed authoring operations.
func (f *authoringFlags) collectRule(ctx context.Context, id string, library bool, started *bool) (rules.RuleMetadata, *string, error) {
	if strings.HasSuffix(id, ".md") {
		return rules.RuleMetadata{}, nil, fmt.Errorf("use a rule ID without the .md extension")
	}
	if err := f.requireRuleGroup(id, library); err != nil {
		*started = true
		return rules.RuleMetadata{}, nil, err
	}
	if err := f.require("title", "when-to-read", "impact", "impact-description"); err != nil {
		return rules.RuleMetadata{}, nil, err
	}
	*started = true
	var body *string
	if f.value("body-file") != "" {
		text, err := readBody(ctx, f.file("body-file"))
		if err != nil {
			return rules.RuleMetadata{}, nil, err
		}
		body = &text
	}
	return rules.RuleMetadata{Title: f.value("title"), WhenToRead: f.value("when-to-read"), Impact: f.value("impact"), ImpactDescription: f.value("impact-description")}, body, nil
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
