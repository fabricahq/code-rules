// Collect and validate the repository, rule and skill selectors, and optional revision of a source a project adds.

package cli

import (
	"errors"
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
)

// collectSource prompts only for a missing repository and, when neither rule nor skill selectors were supplied, for
// both, of which at least one must be answered. The optional ref is never prompted for. Explicitly supplied flags
// are preserved.
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
		return usage(errors.New("supply at least one --rules or --skills selector"))
	}
	text, err := f.askValidated("Rules (comma-separated selectors, or empty for none):", func(value string) error {
		_, _, err := project.ParseSourceRules(splitPromptList(value))
		return err
	})
	if err != nil {
		return err
	}
	*ruleSelectors = splitPromptList(text)
	text, err = f.askValidated("Skills (comma-separated skill IDs or *, or empty for none):", func(value string) error {
		if len(splitPromptList(value)) == 0 && len(*ruleSelectors) == 0 {
			return errors.New("include rules or skills: enter at least one skill, or rule selectors at the previous prompt")
		}
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
