// Orient authors before metadata prompts and explain how to finish the Markdown rule afterward.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func groupIntroduction(groupPath string, isLibrary bool) string {
	heading := fmt.Sprintf("Adding a group at: %s", groupPath)
	if !isLibrary {
		heading = fmt.Sprintf("Adding a project-only group at: %s\n\nThis group belongs to this project.\nIts files are stored in .code-rules/local/%s/.", groupPath, groupPath)
	}
	return fmt.Sprintf(`%s

- A group collects related rules. Use individual rules to give agents guidance.
- The path %q uniquely identifies this group.

Example group:
  Path: practices/testing
  Name: Testing
  Description: Unit and integration testing.
  When to read: When writing or changing tests.

Enter your group's details below.

`, heading, groupPath)
}

func ruleIntroduction(rulePath, bodyFile string, isLibrary bool) string {
	heading := fmt.Sprintf("Adding a rule at: %s", rulePath)
	if !isLibrary {
		heading = fmt.Sprintf("Adding a project-only rule at: %s\n\nThis rule belongs to this project.\nIts file is stored in:\n  .code-rules/local/%s.md", rulePath, rulePath)
	}
	next := "Then edit the created Markdown file to write the full rule."
	if bodyFile != "" {
		next = fmt.Sprintf("The rule text will be read from %s.", bodyFile)
	}
	return fmt.Sprintf(`%s

- A rule gives agents guidance for a specific task or situation.
- The path %q uniquely identifies this rule.

Example rule:
  Path: practices/testing/test-boundaries
  Title: Test boundary conditions
  When to read: When writing or changing tests.
  Impact: HIGH
  Why it matters: Boundary bugs can silently produce incorrect results.
  Rule text: Test empty inputs and values at each supported limit.

Enter your rule's details below.
%s

`, heading, rulePath, next)
}

// addRuleFlags keeps the two authoring scopes' field labels and body instructions consistent.
func (f *authoringFlags) addRuleFlags(cmd *cobra.Command) {
	cmd.Long = cmd.Short + "\n\nA rule is a Markdown file that gives agents guidance for a specific task or situation.\n\nRULE_PATH includes the group path and rule slug, without .md (e.g. practices/testing/my-rule).\nThe title is the rule's readable name. Enter metadata here, then edit the created\nMarkdown file to write the instructions and examples. Use --body-file to supply\nexisting rule text instead of creating a draft." + documentationHelp
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

// ruleReadyHeading explains the next action consistently after group and rule creation.
func ruleReadyHeading(isLibrary bool) string {
	if isLibrary {
		return "After writing the rule text, run:"
	}
	return "Project-only groups and rules are discovered automatically when you build; you don't need to list them in config.json.\n\nAfter writing the rule text, to make the rule accessible to this project, run:"
}
