// Present command-specific and common options consistently without changing flag scope or parsing.

package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const documentationHelp = "\n\nLearn more: https://code-rules.fabricahq.com"

// commandUsage preserves command navigation while grouping only the options available at this scope.
func commandUsage(cmd *cobra.Command) error {
	var out strings.Builder
	out.WriteString("Usage:\n")
	if cmd.Runnable() {
		fmt.Fprintf(&out, "  %s\n", cmd.UseLine())
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(&out, "  %s [command]\n", cmd.CommandPath())
	}
	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(&out, "\nAliases:\n  %s\n", cmd.NameAndAliases())
	}
	if cmd.HasAvailableSubCommands() {
		if len(cmd.Groups()) == 0 {
			writeCommandGroup(&out, cmd, "Available Commands:", "")
		} else {
			for _, group := range cmd.Groups() {
				writeCommandGroup(&out, cmd, group.Title, group.ID)
			}
			if !cmd.AllChildCommandsHaveGroup() {
				writeCommandGroup(&out, cmd, "Additional Commands:", "")
			}
		}
	}
	specific, common := commandOptions(cmd)
	if specific.HasAvailableFlags() {
		fmt.Fprintf(&out, "\n%s:\n%s", commandOptionsTitle(cmd), specific.FlagUsages())
	}
	if common.HasAvailableFlags() {
		fmt.Fprintf(&out, "\nCommon options:\n%s", common.FlagUsages())
	}
	example := cmd.Example
	if cmd.Name() == "group" {
		example = fmt.Sprintf("  %s practices/testing --name \"Testing\"", cmd.CommandPath())
	}
	if example != "" {
		fmt.Fprintf(&out, "\nExample:\n%s\n", strings.TrimRight(example, "\n"))
	}
	if cmd.HasHelpSubCommands() {
		out.WriteString("\nAdditional help topics:\n")
		for _, child := range cmd.Commands() {
			if child.IsAdditionalHelpTopicCommand() {
				fmt.Fprintf(&out, "  %-*s %s\n", child.CommandPathPadding(), child.CommandPath(), child.Short)
			}
		}
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(&out, "\nUse \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
	}
	_, err := io.WriteString(cmd.OutOrStderr(), out.String())
	return err
}

func writeCommandGroup(out *strings.Builder, cmd *cobra.Command, title, id string) {
	fmt.Fprintf(out, "\n%s\n", title)
	for _, child := range commandsInHelpOrder(cmd) {
		if child.GroupID == id && child.IsAvailableCommand() {
			fmt.Fprintf(out, "  %-*s %s\n", child.NamePadding(), child.Name(), child.Short)
		}
	}
}

// commandsInHelpOrder lists known workflows first, then any new commands in Cobra's default order.
// Keep ordering local to presentation: Cobra's sorting setting is shared across command trees and callers.
func commandsInHelpOrder(cmd *cobra.Command) []*cobra.Command {
	order := map[string][]string{
		"code-rules": {"project", "library"},
		"project":    {"init", "add", "sync", "build", "check"},
		"library":    {"init", "add", "check"},
		"add":        {"rule", "group", "library"},
	}[cmd.Name()]
	commands := cmd.Commands()
	ordered := make([]*cobra.Command, 0, len(commands))
	for _, name := range order {
		for _, child := range commands {
			if child.Name() == name {
				ordered = append(ordered, child)
			}
		}
	}
	for _, child := range commands {
		if !slices.Contains(order, child.Name()) {
			ordered = append(ordered, child)
		}
	}
	return ordered
}

// commandOptions builds presentation-only sets; the original flags retain their local or inherited scope.
func commandOptions(cmd *cobra.Command) (*pflag.FlagSet, *pflag.FlagSet) {
	specific := pflag.NewFlagSet("specific", pflag.ContinueOnError)
	common := pflag.NewFlagSet("common", pflag.ContinueOnError)
	available := pflag.NewFlagSet("available", pflag.ContinueOnError)
	available.AddFlagSet(cmd.LocalFlags())
	available.AddFlagSet(cmd.InheritedFlags())
	available.VisitAll(func(flag *pflag.Flag) {
		switch flag.Name {
		case "config", "directory", "help", "json", "non-interactive":
			common.AddFlag(flag)
		case "version":
			// The root prints the tool version; library adoption uses a version constraint.
			if cmd.HasParent() {
				specific.AddFlag(flag)
			} else {
				common.AddFlag(flag)
			}
		default:
			specific.AddFlag(flag)
		}
	})
	return specific, common
}

func commandOptionsTitle(cmd *cobra.Command) string {
	switch cmd.Name() {
	case "group":
		return "Group options"
	case "rule":
		return "Rule options"
	case "library":
		return "Library options"
	case "init":
		if cmd.HasParent() && cmd.Parent().Name() == "library" {
			return "Library options"
		}
	}
	return "Command options"
}
