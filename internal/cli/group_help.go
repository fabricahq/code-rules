// Present group metadata separately from common options without changing flag scope or parsing.

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// groupUsage renders the options actually available on this command, including inherited JSON output.
func groupUsage(cmd *cobra.Command) error {
	group := pflag.NewFlagSet("group", pflag.ContinueOnError)
	common := pflag.NewFlagSet("common", pflag.ContinueOnError)
	flags := pflag.NewFlagSet("available", pflag.ContinueOnError)
	flags.AddFlagSet(cmd.LocalFlags())
	flags.AddFlagSet(cmd.InheritedFlags())
	flags.VisitAll(func(flag *pflag.Flag) {
		switch flag.Name {
		case "name", "description", "when-to-read":
			group.AddFlag(flag)
		default:
			common.AddFlag(flag)
		}
	})
	_, err := fmt.Fprintf(cmd.OutOrStderr(), "Usage:\n  %s\n\nGroup options:\n%s\n\nCommon options:\n%s\n\nExample:\n  %s practices/testing --name \"Testing\"\n",
		cmd.UseLine(), strings.TrimRight(group.FlagUsages(), "\n"), strings.TrimRight(common.FlagUsages(), "\n"), cmd.CommandPath())
	return err
}
