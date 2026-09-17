// Package cli implements the native command boundary without owning process exit or signal handlers.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/spf13/cobra"
)

// Options supplies trusted embedding settings; command-line input cannot replace Git execution settings.
type Options struct {
	Version string
	// Directory anchors relative paths without changing the process working directory.
	Directory string
	Git       imports.Options
}

// Streams separates user input, machine-readable results, and diagnostics for each invocation.
type Streams struct {
	In       io.Reader
	Out, Err io.Writer
}

// Run executes a fresh command tree and returns 0 for success, 1 for operation failure/stale output, or 2 for usage errors.
func Run(ctx context.Context, args []string, streams Streams, options Options) int {
	if streams.In == nil {
		streams.In = emptyInput{}
	}
	if streams.Out == nil {
		streams.Out = io.Discard
	}
	if streams.Err == nil {
		streams.Err = io.Discard
	}
	if options.Version == "" {
		options.Version = "0.0.0-development"
	}
	started := false
	stale := false
	root := &cobra.Command{Use: "code-rules", Short: "Manage versioned engineering rules for your project", SilenceErrors: true, SilenceUsage: true, Args: cobra.NoArgs}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetIn(streams.In)
	root.SetOut(streams.Out)
	root.SetErr(streams.Err)
	root.SetArgs(args)
	root.Version = options.Version
	root.SetVersionTemplate("{{.Version}}\n")
	root.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
	for _, name := range []string{"sync", "build", "check"} {
		config := &singleString{}
		descriptions := map[string]string{"sync": "Fetch libraries and rebuild vendor and generated files", "build": "Rebuild generated guidance from verified local snapshots", "check": "Check generated guidance without changing files or using Git"}
		cmd := &cobra.Command{Use: name, Short: descriptions[name], Args: cobra.NoArgs}
		cmd.Flags().Var(config, "config", "Configuration file (default .code-rules/config.json)")
		cmd.RunE = func(cmd *cobra.Command, _ []string) error {
			started = true
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
				changes, err = project.Check(cmd.Context(), projectOptions)
			}
			if err != nil {
				return err
			}
			stale = name == "check" && len(changes.Added)+len(changes.Changed)+len(changes.Removed) > 0
			return writeJSON(streams.Out, changes)
		}
		root.AddCommand(cmd)
	}
	addProjectAuthoringCommands(root, options, &started)
	addLibraryCommands(root, options, &started)
	if _, err := root.ExecuteContextC(ctx); err != nil {
		fmt.Fprintln(streams.Err, err)
		if !started {
			fmt.Fprintln(streams.Err, "Run code-rules --help for usage.")
			return 2
		}
		return 1
	}
	if stale {
		return 1
	}
	return 0
}

// writeJSON emits readable machine output without HTML-specific escapes.
func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// singleString rejects duplicate scalar flags so an accidental repeated option cannot silently replace its value.
type singleString struct {
	value string
	set   bool
}

// Set accepts a scalar value once, rejecting blank text and option tokens used as missing values.
func (v *singleString) Set(value string) error {
	if v.set {
		return fmt.Errorf("option may only be specified once")
	}
	if strings.TrimSpace(value) == "" || strings.HasPrefix(value, "-") {
		return fmt.Errorf("expected a nonempty value, not another option")
	}
	v.value = value
	v.set = true
	return nil
}

// String returns the current flag value for help rendering.
func (v *singleString) String() string { return v.value }

// Type describes the flag's value in Cobra help.
func (v *singleString) Type() string { return "string" }

// emptyInput supplies EOF when an embedding caller has no input stream.
type emptyInput struct{}

// Read reports EOF without waiting for terminal input.
func (emptyInput) Read([]byte) (int, error) { return 0, io.EOF }
