// Classify invalid command input explicitly, independently of operation or prompt ordering.

package cli

import (
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

// classifyArguments wraps Cobra's positional and flag validation without changing its parsing behavior.
func classifyArguments(command *cobra.Command) {
	command.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usage(err) })
	if validate := command.Args; validate != nil {
		command.Args = func(cmd *cobra.Command, args []string) error { return usage(validate(cmd, args)) }
	}
	for _, child := range command.Commands() {
		classifyArguments(child)
	}
}
