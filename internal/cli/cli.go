// Package cli implements the native command boundary without owning process exit or signal handlers.
package cli

import (
	"context"
	"errors"
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

// Streams separates user input, command results, and human diagnostics for each invocation.
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
	output := &commandOutput{}
	root := &cobra.Command{Use: "code-rules", Short: "The package manager for your engineering rules", SilenceErrors: true, SilenceUsage: true, Args: cobra.NoArgs}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetIn(streams.In)
	root.SetOut(&output.text)
	root.PersistentFlags().Bool("json", false, "Return one JSON response, including errors; never prompt")
	root.SetErr(streams.Err)
	root.SetArgs(args)
	root.Version = options.Version
	root.SetVersionTemplate("{{.Version}}\n")
	root.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
	for _, name := range []string{"sync", "build", "check"} {
		config := &singleString{}
		descriptions := map[string]string{"sync": "Fetch libraries and rebuild vendor and generated files", "build": "Rebuild generated guidance from verified local snapshots", "check": "Check generated guidance and the project README without changing files or using Git"}
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
		root.AddCommand(cmd)
	}
	addProjectAuthoringCommands(root, options, &started, output)
	addLibraryCommands(root, options, &started, output)
	// Discover the output mode even when Cobra stops at an earlier invalid argument.
	output.json = requestsJSON(root, args)
	if err := rejectMissingValues(root, args); err != nil {
		return output.finish(streams, root, err, 2)
	}
	command, err := root.ExecuteContextC(ctx)
	if command == nil {
		command = root
	}
	code := 0
	if err != nil {
		code = 1
		if !started && ctx.Err() == nil && !errors.Is(err, context.Canceled) {
			code = 2
		}
	}
	return output.finish(streams, command, err, code)
}

// singleString rejects duplicate scalar flags so an accidental repeated option cannot silently replace its value.
type singleString struct {
	value string
	set   bool
}

// Set accepts a nonblank scalar once; rejectMissingValues checks consumed option tokens before parsing.
func (v *singleString) Set(value string) error {
	if v.set {
		return fmt.Errorf("option may only be specified once")
	}
	if strings.TrimSpace(value) == "" {
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

// rejectMissingValues distinguishes a consumed option token from an explicit equals-form value.
func rejectMissingValues(root *cobra.Command, args []string) error {
	command, _, err := root.Find(args)
	if err != nil {
		return nil
	} // Cobra reports unknown command paths itself.
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "--") || strings.Contains(arg, "=") {
			continue
		}
		flag := command.Flags().Lookup(strings.TrimPrefix(arg, "--"))
		if flag == nil || flag.NoOptDefVal != "" {
			continue
		}
		if i+1 < len(args) && strings.HasPrefix(args[i+1], "-") {
			return fmt.Errorf("%s requires a value; use %s=value for a value beginning with '-'", arg, arg)
		}
		i++
	}
	return nil
}
