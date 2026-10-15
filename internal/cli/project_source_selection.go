// Collect and validate the repository, rule and skill selectors, and optional revision of a source a project adds.

package cli

import (
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
)

// collectSource prompts only for a missing repository and, when neither rule nor skill selectors were supplied, for
// both, either of which can be left empty, since a source that includes nothing still installs its library's
// required skills. Without prompts, no selectors include nothing. The optional ref is never prompted for. Explicitly
// supplied flags are preserved.
func (f *authoringFlags) collectSource(ruleSelectors, skillSelectors *[]string) error {
	if f.value("ref") != "" {
		if _, err := project.ParseSourceRef(f.value("ref")); err != nil {
			return usage(err)
		}
	}
	if err := f.require("repository"); err != nil {
		return err
	}
	if len(*ruleSelectors) > 0 || len(*skillSelectors) > 0 {
		return nil
	}
	if !f.interactive() {
		return nil
	}
	text, err := f.askValidated("Rules (comma-separated selectors, or empty for none):", func(value string) error {
		_, _, err := project.ParseSourceRules(splitPromptList(value))
		return err
	})
	if err != nil {
		return err
	}
	*ruleSelectors = splitPromptList(text)
	text, err = f.askValidated("Skills (skill IDs, recommended, or *; empty for required only):", func(value string) error {
		_, err := project.ParseSourceSkills(splitPromptList(value))
		return err
	})
	if err != nil {
		return err
	}
	*skillSelectors = splitPromptList(text)
	return nil
}

// splitPromptList trims each comma-separated entry before domain validation, dropping empty ones.
func splitPromptList(answer string) []string {
	entries := []string{}
	for _, entry := range strings.Split(answer, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			entries = append(entries, entry)
		}
	}
	return entries
}
