// Render human failures consistently without changing structured error contracts.

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/fabricahq/code-rules/internal/rules"
	"golang.org/x/term"
)

// humanError separates failures from preceding output and colors only an interactive label.
// Unwrapped validation errors expose their location separately; wrapped errors retain all outer context.
func humanError(destination io.Writer, err error) string {
	label := "Error:"
	if file, ok := destination.(*os.File); ok && term.IsTerminal(int(file.Fd())) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "" && os.Getenv("TERM") != "dumb" {
		label = "\x1b[1;31mError:\x1b[0m"
	}
	var validation *rules.ValidationError
	if errors.As(err, &validation) && err == validation {
		return fmt.Sprintf("\n%s %s\n\nLocation: %s\n", label, validation.Problem, validation.Location)
	}
	return fmt.Sprintf("\n%s %s\n", label, err)
}
