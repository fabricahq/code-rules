// Register project list skills and render its listing.

package cli

import (
	"fmt"
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/spf13/cobra"
)

// projectListCommand groups the project's listings.
func projectListCommand(options Options, output *commandOutput) *cobra.Command {
	list := &cobra.Command{Use: "list", Short: "List what the project installs and could install"}
	skills := &cobra.Command{Use: "skills", Short: "List every installed and available skill, with its level", Args: cobra.NoArgs}
	skills.Long = "List each skill the project's sources offer, whether the project installs it,\nexcludes it, replaces it with its own skill, or could add it, and the level the\nsource's library gives it: required, recommended, or offered. Then list the\nproject's own skills. The listing reads the source records sync wrote, so run\nsyllabus project sync first after changing the configuration." + documentationHelp
	skills.RunE = func(cmd *cobra.Command, _ []string) error {
		directory, err := commandDirectory(cmd.Context(), options.Directory, "project", false)
		if err != nil {
			return err
		}
		listing, err := project.ListSkills(cmd.Context(), project.Options{Directory: directory, ToolVersion: options.Version})
		if err != nil {
			return err
		}
		output.report = commandReport{value: listing, human: formatSkillListing(listing)}
		return nil
	}
	list.AddCommand(skills)
	return list
}

// formatSkillListing renders the listing as aligned columns: the skill, its level, the library release that set
// it, and its status.
func formatSkillListing(listing project.SkillListing) string {
	if len(listing.Skills) == 0 {
		return "The project has no skills, and its sources offer none.\n"
	}
	rows := [][]string{{"SKILL", "LEVEL", "SET BY", "STATUS"}}
	for _, skill := range listing.Skills {
		level, setBy := "-", "-"
		switch {
		case skill.Level != "":
			level, setBy = skill.Level, skill.SetBy.Source
			if skill.SetBy.Release != 0 {
				setBy += fmt.Sprintf(", release %d", skill.SetBy.Release)
			} else {
				setBy += ", commit " + shortCommit(skill.SetBy.Commit)
			}
		case skill.Source == "":
			setBy = "this project"
		default:
			setBy = skill.Source + ", not a library"
		}
		status := skill.Status
		if skill.ReplacedBy != "" {
			status += " by " + skill.ReplacedBy
		}
		rows = append(rows, []string{skill.ID, level, setBy, status})
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], len(cell))
		}
	}
	var out strings.Builder
	for _, row := range rows {
		line := ""
		for i, cell := range row {
			if i < len(row)-1 {
				line += fmt.Sprintf("%-*s  ", widths[i], cell)
			} else {
				line += cell
			}
		}
		out.WriteString(line + "\n")
	}
	return out.String()
}
