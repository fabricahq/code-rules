// Package cli implements the native command boundary without owning process exit or signal handlers.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/fabricahq/code-rules/internal/imports"
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
	output := &commandOutput{}
	root := &cobra.Command{Use: "code-rules", Short: "The package manager for your engineering rules", SilenceErrors: true, SilenceUsage: true, Args: cobra.NoArgs}
	root.CompletionOptions.DisableDefaultCmd = true
	// An unnamed, hidden command prevents Cobra from installing its help subcommand. Help flags remain local.
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.SetIn(streams.In)
	root.SetOut(&output.text)
	root.PersistentFlags().Bool("json", false, "Return one JSON response, including errors; never prompt")
	root.SetErr(streams.Err)
	root.SetArgs(args)
	root.Version = options.Version
	root.SetVersionTemplate("{{.Version}}\n")
	root.SetUsageFunc(commandUsage)
	root.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
	root.AddCommand(newProjectCommand(options, output))
	addLibraryCommands(root, options, output)
	// Discover the output mode even when Cobra stops at an earlier invalid argument.
	classifyArguments(root)
	output.json = requestsJSON(root, args)
	if err := rejectMissingValues(root, args); err != nil {
		return output.finish(streams, root, usage(err))
	}
	if command, err := validateCommandPath(root, args); err != nil {
		return output.finish(streams, command, usage(err))
	}
	command, err := root.ExecuteContextC(ctx)
	if command == nil {
		command = root
	}
	return output.finish(streams, command, err)
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
