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

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/spf13/cobra"
)

// commandOutput retains one invocation's result until its exit status and output mode are known.
type commandOutput struct {
	json   bool
	report commandReport
	text   bytes.Buffer // Cobra help and version output, rendered only after command completion.
}

// response is the common JSON envelope; a stale check includes both current problems and an error.
type response struct {
	OK    bool           `json:"ok"`
	Value any            `json:"value,omitempty"`
	Error *responseError `json:"error,omitempty"`
}

// responseError classifies failures without requiring callers to parse the human message.
type responseError struct {
	Code     string `json:"code,omitempty"`
	Kind     string `json:"kind"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

// finish emits one response from a completed report or a typed execution error.
func (o *commandOutput) finish(streams Streams, cmd *cobra.Command, err error) int {
	problem := o.report.failure
	if err != nil {
		problem = classifyError(err)
	}
	code := 0
	if problem != nil {
		code = 1
		if problem.Kind == "usage" {
			code = 2
		}
	}
	var writeErr error
	if o.json {
		value := o.report.value
		if value == nil && o.text.Len() > 0 && err == nil {
			value = map[string]string{"text": o.text.String()}
		}
		result := response{OK: code == 0, Value: value, Error: problem}
		encoder := json.NewEncoder(streams.Out)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		writeErr = encoder.Encode(result)
	} else {
		var text strings.Builder
		if err == nil && problem != nil {
			text.WriteString(humanError(streams.Out, errors.New(problem.Message)))
			text.WriteByte('\n')
		}
		// A post-commit presentation failure must not hide the completed operation's receipt.
		text.WriteString(o.report.human)
		if err == nil {
			text.WriteString(o.text.String())
		}
		if text.Len() > 0 {
			_, writeErr = io.WriteString(streams.Out, text.String())
		}
		if err != nil {
			_, diagnosticErr := io.WriteString(streams.Err, humanError(streams.Err, err))
			writeErr = errors.Join(writeErr, diagnosticErr)
			if code == 2 {
				_, hintErr := fmt.Fprintf(streams.Err, "\nRun %s --help for usage.\n", cmd.CommandPath())
				writeErr = errors.Join(writeErr, hintErr)
			}
		}
	}
	if writeErr != nil {
		if o.json {
			fmt.Fprintf(streams.Err, "write command output: %v\n", writeErr)
		} else {
			fmt.Fprint(streams.Err, humanError(streams.Err, fmt.Errorf("write command output: %w", writeErr)))
		}
		return 1
	}
	return code
}

// classifyError keeps generic kinds stable and exposes domain codes without parsing diagnostic text.
func classifyError(err error) *responseError {
	result := &responseError{Kind: "operation", Message: err.Error()}
	var invalid *usageError
	var validation *rules.ValidationError
	var domain *filetxn.Error
	if errors.As(err, &domain) {
		result.Code = domain.Code
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		result.Kind = "cancelled"
	case errors.As(err, &invalid):
		result.Kind = "usage"
	case result.Code == "invalid-rule-path":
		result.Kind = "usage"
	case errors.As(err, &validation):
		result.Kind = "validation"
		result.Location = validation.Location
	}
	return result
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
func formatAuthored(out *strings.Builder, files, warnings []string) {
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
}
