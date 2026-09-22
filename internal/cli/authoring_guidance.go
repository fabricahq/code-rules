// Orient authors before metadata prompts and explain how to finish the Markdown rule afterward.

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func groupIntroduction(groupPath string) string {
	return fmt.Sprintf(`Adding a group at: %s

- The path %q uniquely identifies the group.
- The name is a human-friendly label, such as "Testing".
- A group collects related rules. Use individual rules to give agents guidance.

Example group:
  Path: practices/testing
  Name: Testing
  Description: Unit and integration testing.
  When to read: When writing or changing tests.

Enter your group's details below. The example is not saved.

`, groupPath, groupPath)
}

func ruleIntroduction(rulePath, bodyFile string) string {
	next := "First, enter the details below. Then edit the created Markdown file to write the full rule."
	if bodyFile != "" {
		next = fmt.Sprintf("The rule text will be read from %s. Enter the remaining details below.", bodyFile)
	}
	return fmt.Sprintf(`Adding a rule at: %s
The rule path identifies it; the title is its readable name.

Example rule:
  Path: practices/testing/test-boundaries
  Title: Test boundary conditions
  When to read: When writing or changing tests.
  Impact: HIGH
  Why it matters: Boundary bugs can silently produce incorrect results.
  Rule text: Test empty inputs and values at each supported limit.

The example is not saved. Your title, reading cue, and impact details go at the
top of the Markdown file. The instructions and examples go below that metadata.
%s

`, rulePath, next)
}

// addRuleFlags keeps the two authoring scopes' field labels and body instructions consistent.
func (f *authoringFlags) addRuleFlags(cmd *cobra.Command) {
	cmd.Long = cmd.Short + "\n\nRULE_PATH includes the group path and rule slug, without .md (e.g. practices/testing/my-rule).\nThe title is the rule's readable name. Enter metadata here, then edit the created\nMarkdown file to write the instructions and examples. Use --body-file to supply\nexisting rule text instead of creating a draft."
	for name, description := range map[string]string{
		"title":              "Readable, action-oriented rule title",
		"when-to-read":       "When an agent should read this rule",
		"impact":             "Consequence level: CRITICAL, HIGH, MEDIUM-HIGH, MEDIUM, LOW-MEDIUM, or LOW",
		"impact-description": "Why this rule matters",
		"body-file":          "Read rule text from this UTF-8 Markdown file instead of creating a draft",
	} {
		f.add(cmd, name, description)
	}
	f.prompts = map[string]string{
		"title":              "Rule title",
		"when-to-read":       "When to read",
		"impact":             "Impact (CRITICAL, HIGH, MEDIUM-HIGH, MEDIUM, LOW-MEDIUM, LOW)",
		"impact-description": "Why it matters",
	}
}

// formatRuleCreated uses published paths, preserves warnings, and points to the correct scope and location.
func formatRuleCreated(out *strings.Builder, cmd *cobra.Command, files, warnings []string, isLibrary bool) {
	draft := cmd.Flags().Lookup("body-file").Value.String() == ""
	if draft {
		out.WriteString("Rule draft created:\n")
	} else {
		out.WriteString("Rule created from --body-file:\n")
	}
	for _, file := range files {
		fmt.Fprintf(out, "  %s\n", file)
	}
	for _, warning := range warnings {
		fmt.Fprintf(out, "Warning: %s\n", warning)
	}
	if draft {
		out.WriteString("\nNext: Open the Markdown file above in your editor.\nKeep the metadata between the --- lines at the top. Below it:\n  - State the instructions and explain why they matter.\n  - Add correct and incorrect examples, then describe how to check compliance.\n  - Replace <...> placeholders and remove unused template sections.\n")
		if isLibrary {
			out.WriteString("  - Remove the <!-- code-rules:draft --> marker when the rule is complete.\n")
		}
	} else {
		out.WriteString("\nReview the Markdown file above. Make future edits directly in that file.\n")
	}
	out.WriteString("\nWhen the rule is ready, run:\n")
	if isLibrary {
		command := "code-rules library check"
		if directory := cmd.Flags().Lookup("directory").Value.String(); directory != "" {
			command += " --directory='" + strings.ReplaceAll(directory, "'", "'\"'\"'") + "'"
		}
		fmt.Fprintf(out, "  %s\n", command)
	} else {
		config := cmd.Flags().Lookup("config").Value.String()
		fmt.Fprintf(out, "  %s\n  %s\n", checkRepairCommand("build", config), checkRepairCommand("check", config))
	}
}
