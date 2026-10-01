// Classify invalid command input explicitly, independently of operation or prompt ordering.

package cli

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

// usageError retains an input failure's original identity while assigning the CLI usage contract.
type usageError struct{ cause error }

func (e *usageError) Error() string { return e.cause.Error() }
func (e *usageError) Unwrap() error { return e.cause }

func usage(err error) error {
	if err == nil {
		return nil
	}
	return &usageError{cause: err}
}

// invalidFlagValue matches the error the flag parser gives a value it can't convert, which ends with Go's
// conversion error, such as invalid argument "maybe" for "--dry-run" flag: strconv.ParseBool: ....
var invalidFlagValue = regexp.MustCompile(`^invalid argument "(.*)" for "(.+)" flag: strconv\.Parse(Bool|Int|Uint|Float)\b`)

// flagError restates a flag value the parser couldn't convert without Go's conversion error, and keeps every other
// flag error as it is.
func flagError(err error) error {
	parts := invalidFlagValue.FindStringSubmatch(err.Error())
	if parts == nil {
		return err
	}
	expected := "a number"
	if parts[3] == "Bool" {
		expected = "true or false"
	}
	return fmt.Errorf("invalid value %q for %s: expected %s", parts[1], parts[2], expected)
}

// classifyArguments wraps Cobra's positional and flag validation without changing its parsing behavior.
func classifyArguments(command *cobra.Command) {
	command.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usage(flagError(err)) })
	if validate := command.Args; validate != nil {
		command.Args = func(cmd *cobra.Command, args []string) error { return usage(validate(cmd, args)) }
	}
	for _, child := range command.Commands() {
		classifyArguments(child)
	}
}
