// Explain required authoring arguments before prompts or filesystem operations begin.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// requiredArgument validates one named argument, using the invoked path so compatibility commands get valid examples.
func requiredArgument(name, example, explanation string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch len(args) {
		case 0:
			return fmt.Errorf("missing %s.\n%s\n\nUsage:\n  %s\n\nExample:\n  %s %s", name, explanation, cmd.UseLine(), cmd.CommandPath(), example)
		case 1:
			return nil
		default:
			return fmt.Errorf("expected one %s; received %d arguments.\n\nUsage:\n  %s", name, len(args), cmd.UseLine())
		}
	}
}
