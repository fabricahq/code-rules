// Format command outcomes for people or as a single structured response for automation.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/spf13/cobra"
)

// commandOutput retains one invocation's result until its exit status and output mode are known.
type commandOutput struct {
	json  bool
	value any
	text  bytes.Buffer // Cobra help and version output, rendered only after command completion.
}

// response is the common JSON envelope; a stale check includes both current problems and an error.
type response struct {
	OK    bool           `json:"ok"`
	Value any            `json:"value,omitempty"`
	Error *responseError `json:"error,omitempty"`
}

// responseError classifies failures without requiring callers to parse the human message.
type responseError struct {
	Kind     string `json:"kind"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

// finish emits one response and preserves command exit codes; write failures go to stderr and exit 1.
func (o *commandOutput) finish(streams Streams, cmd *cobra.Command, err error, code int) int {
	var writeErr error
	if o.json {
		value := o.value
		if value == nil && o.text.Len() > 0 && err == nil {
			value = map[string]string{"text": o.text.String()}
		}
		result := response{OK: code == 0, Value: value}
		if err != nil {
			result.Error = classifyError(err, code)
		}
		encoder := json.NewEncoder(streams.Out)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		writeErr = encoder.Encode(result)
	} else {
		var text strings.Builder
		if o.value != nil {
			formatHuman(&text, cmd, o.value)
		} else if err == nil {
			text.WriteString(o.text.String())
		}
		if text.Len() > 0 {
			_, writeErr = io.WriteString(streams.Out, text.String())
		}
		if err != nil && err != errCheckOutOfDate {
			fmt.Fprintln(streams.Err, err)
			if code == 2 {
				fmt.Fprintf(streams.Err, "\nRun %s --help for usage.\n", cmd.CommandPath())
			}
		}
	}
	if writeErr != nil {
		fmt.Fprintf(streams.Err, "write command output: %v\n", writeErr)
		return 1
	}
	return code
}

// classifyError preserves validation locations and stable failure categories in JSON responses.
func classifyError(err error, code int) *responseError {
	result := &responseError{Kind: "operation", Message: err.Error()}
	var validation *rules.ValidationError
	switch {
	case errors.Is(err, errCheckOutOfDate):
		result.Kind = "out_of_date"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		result.Kind = "cancelled"
	case code == 2:
		result.Kind = "usage"
	case errors.As(err, &validation):
		result.Kind = "validation"
		result.Location = validation.Location
	}
	return result
}

// formatHuman describes changed paths, validation counts, warnings, and next actions without JSON syntax.
func formatHuman(out *strings.Builder, cmd *cobra.Command, value any) {
	switch result := value.(type) {
	case projectCheckResult:
		if result.Status == "up_to_date" {
			out.WriteString("Status: up to date.\nGenerated guidance and the project guide are current.\nNo files were changed.\n")
		} else {
			out.WriteString("Status: out of date.\nNo files were changed.\nPaths are relative to the configuration directory.\n\nProblems:\n")
			for _, problem := range result.Problems {
				fmt.Fprintf(out, "  %s: %s\n    Next: %s\n", problem.Message, problem.Path, problem.NextStep)
			}
		}
	case project.AuthoringResult:
		if cmd.Name() == "init" {
			formatProjectInitialized(out, cmd, result)
		} else {
			formatAuthored(out, result.Files, result.Warnings, result.Next)
		}
	case library.AuthoringResult:
		formatAuthored(out, result.Files, result.Warnings, result.Next)
	case library.CheckResult:
		fmt.Fprintf(out, "Library is valid: %d group(s), %d rule(s).\n", result.Groups, result.Rules)
		for _, warning := range result.Warnings {
			fmt.Fprintf(out, "Warning: %s\n", warning)
		}
	case project.FileChanges:
		stale := len(result.Added)+len(result.Changed)+len(result.Removed) > 0
		fmt.Fprintf(out, "%s complete: %d added, %d changed, %d removed.\n", strings.ToUpper(cmd.Name()[:1])+cmd.Name()[1:], len(result.Added), len(result.Changed), len(result.Removed))
		if stale && cmd.Name() == "build" {
			out.WriteString("Paths relative to generated/:\n")
		}
		if stale && cmd.Name() == "sync" {
			out.WriteString("Paths relative to the configuration directory:\n")
		}
		for _, group := range []struct {
			label string
			paths []string
		}{{"Add", result.Added}, {"Change", result.Changed}, {"Remove", result.Removed}} {
			for _, path := range group.paths {
				fmt.Fprintf(out, "  %s: %s\n", group.label, path)
			}
		}
	}
}

// formatProjectInitialized orients people after setup while leaving the structured result unchanged.
func formatProjectInitialized(out *strings.Builder, cmd *cobra.Command, result project.AuthoringResult) {
	if len(result.Files) == 0 {
		out.WriteString("Code Rules is already initialized.\nNo files changed.\n")
	} else {
		out.WriteString("Code Rules initialized!\n")
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(out, "Warning: %s\n", warning)
	}
	config := cmd.Flags().Lookup("config").Value.String()
	if len(result.Files) == 0 {
		out.WriteString("\nRun code-rules project --help to manage this project's rules.\n")
		return
	}
	location := config
	if location == "" {
		location = ".code-rules/config.json"
	}
	fmt.Fprintf(out, "\nConfiguration: %s\n", location)
	fmt.Fprintf(out, "\nStart with a project-only rule (example):\n  %s\n  %s\n",
		checkRepairCommand("add group practices/testing", config),
		checkRepairCommand("add rule practices/testing/my-rule", config))
	fmt.Fprintf(out, "\nOr use a shared library (example):\n  %s\n  %s\n",
		checkRepairCommand("add library team", config),
		checkRepairCommand("sync", config))
}

// requestsJSON recognizes the output flag before usage validation, ignoring equals-form values and positional literals after --.
func requestsJSON(root *cobra.Command, args []string) bool {
	command, _, _ := root.Find(args)
	if command == nil {
		command = root
	}
	enabled := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		name, value, equals := strings.Cut(arg, "=")
		if name == "--json" {
			enabled = true
			if equals {
				if parsed, err := strconv.ParseBool(value); err == nil {
					enabled = parsed
				}
			}
			continue
		}
		if equals || !strings.HasPrefix(name, "--") {
			continue
		}
		flag := command.Flags().Lookup(strings.TrimPrefix(name, "--"))
		if flag == nil {
			flag = root.PersistentFlags().Lookup(strings.TrimPrefix(name, "--"))
		}
		if flag != nil && flag.NoOptDefVal == "" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
		}
	}
	return enabled
}

// formatAuthored presents completed file changes consistently for project and library operations.
func formatAuthored(out *strings.Builder, files, warnings []string, next string) {
	if len(files) == 0 {
		out.WriteString("No files changed.\n")
	} else {
		out.WriteString("Updated files:\n")
		for _, path := range files {
			fmt.Fprintf(out, "  %s\n", path)
		}
	}
	for _, warning := range warnings {
		fmt.Fprintf(out, "Warning: %s\n", warning)
	}
	if next != "" {
		fmt.Fprintf(out, "\nNext: %s\n", next)
	}
}
