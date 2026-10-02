// Render human failures consistently without changing structured error contracts.

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/project"
	"golang.org/x/term"
)

// humanError separates failures from preceding output and colors only an interactive label.
// Unwrapped validation errors expose their location separately; wrapped errors retain all outer context.
// The message can carry library text, so its control characters are escaped; only the label's color stays.
func humanError(destination io.Writer, err error) string {
	label := "Error:"
	if file, ok := destination.(*os.File); ok && term.IsTerminal(int(file.Fd())) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "" && os.Getenv("TERM") != "dumb" {
		label = "\x1b[1;31mError:\x1b[0m"
	}
	var validation errs.ValidationError
	if errors.As(err, &validation) && err == error(validation) {
		return fmt.Sprintf("%s %s\n\nLocation: %s\n", label, terminalText(sentence(validation.ValidationProblem())), terminalText(validation.ValidationLocation()))
	}
	// A command that wrote nothing says so after the problem, keeping a validation error's location on its own line.
	var unchanged *project.UnchangedError
	if errors.As(err, &unchanged) && err == error(unchanged) && errors.As(unchanged.Err, &validation) && unchanged.Err == error(validation) {
		return fmt.Sprintf("%s %s %s\n\nLocation: %s\n", label, terminalText(sentence(validation.ValidationProblem())), unchanged.Outcome(), terminalText(validation.ValidationLocation()))
	}
	return fmt.Sprintf("%s %s\n", label, terminalText(sentence(err.Error())))
}

// sentence ends a message with a period, as the CLI shows every message, unless it ends with punctuation already or
// spans several lines, whose last line may be a command to copy.
func sentence(message string) string {
	message = strings.TrimRight(message, " ")
	if message == "" || strings.Contains(message, "\n") || strings.ContainsAny(message[len(message)-1:], ".!?:") {
		return message
	}
	return message + "."
}
