// Format command outcomes for people or as a single structured response for automation.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// commandOutput retains one invocation's result until its exit status and output mode are known.
type commandOutput struct {
	json   bool
	report commandReport
	text   bytes.Buffer // Cobra help, version, and license output, rendered only after command completion.
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
		switch {
		case problem.Kind == "usage":
			code = 2
		case errors.Is(err, context.Canceled):
			// Only an interrupt, such as Ctrl-C, cancels a command; exit as a shell reports SIGINT.
			code = 130
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
		// A report that is itself a failure, such as an out-of-date check, says so in its own text and exit status.
		var text strings.Builder
		// A post-commit presentation failure must not hide the completed operation's receipt. Reports show library
		// text, such as change summaries, so control characters are escaped.
		text.WriteString(terminalText(o.report.human))
		if err == nil {
			text.WriteString(o.text.String())
		}
		if text.Len() > 0 {
			_, writeErr = io.WriteString(streams.Out, text.String())
		}
		if err != nil {
			// An interrupt leaves the cursor after the ^C the terminal echoed, so the error starts a line of its own.
			if interruptedMidLine(err) && isTerminal(streams.Err) {
				_, _ = io.WriteString(streams.Err, "\n")
			}
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
	var git *gitexec.Error
	switch {
	case errors.As(err, &domain):
		result.Code = domain.Code
	case errors.As(err, &git):
		result.Code = git.Code
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		result.Kind = "cancelled"
	case errors.As(err, &invalid):
		result.Kind = "usage"
	case result.Code == "invalid-rule-path", result.Code == "invalid-arguments":
		result.Kind = "usage"
	case errors.As(err, &validation):
		result.Kind = "validation"
		result.Location = validation.Location
	}
	// Every usage refusal, such as an unknown flag or flags that can't be combined, has one code.
	if result.Kind == "usage" && result.Code == "" {
		result.Code = "invalid-arguments"
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

// formatAuthored presents completed file changes consistently for project and library operations: created files
// under Added and modified ones under Changed, each relative to workdir when it's inside workdir, then warnings.
func formatAuthored(out *strings.Builder, added, changed, warnings []string, workdir string) {
	if len(added)+len(changed) == 0 {
		out.WriteString("No files changed.\n")
	}
	for _, group := range []struct {
		label string
		files []string
	}{{"Added", added}, {"Changed", changed}} {
		if len(group.files) > 0 {
			out.WriteString(group.label + ":\n")
		}
		for _, file := range group.files {
			fmt.Fprintf(out, "  %s\n", displayPath(workdir, file))
		}
	}
	for _, warning := range warnings {
		fmt.Fprintf(out, "Warning: %s\n", warning)
	}
}

// displayPath returns file relative to workdir, the process's working directory when empty, when file is inside
// it, and file unchanged otherwise. It resolves symbolic links in workdir, because operations report resolved paths, such as /private/var for /var on macOS.
func displayPath(workdir, file string) string {
	if absolute, err := filepath.Abs(workdir); err == nil {
		workdir = absolute
	}
	if resolved, err := filepath.EvalSymlinks(workdir); err == nil {
		workdir = resolved
	}
	if relative, err := filepath.Rel(workdir, file); err == nil && filepath.IsAbs(file) && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return relative
	}
	return file
}

// interruptedMidLine reports whether err is an interrupt that arrived while the command was working, so the
// terminal's cursor follows the ^C it echoed; a prompt's interrupt has already ended its line.
func interruptedMidLine(err error) bool {
	var prompt *cancelled
	return errors.Is(err, context.Canceled) && !errors.As(err, &prompt)
}

// isTerminal reports whether destination is a terminal.
func isTerminal(destination io.Writer) bool {
	file, ok := destination.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}
