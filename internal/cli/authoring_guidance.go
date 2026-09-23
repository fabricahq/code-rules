// Orient authors before metadata prompts and explain how to finish the Markdown rule afterward.

package cli

import "fmt"

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

// ruleReadyHeading explains the next action consistently after group and rule creation.
func ruleReadyHeading(isLibrary bool) string {
	if isLibrary {
		return "After writing the rule text, run:"
	}
	return "Project-only groups and rules are discovered automatically when you build; you don't need to list them in config.json.\n\nAfter writing the rule text, to make the rule accessible to this project, run:"
}
