// Collect and validate the repository, selection, and optional revision for a library imported into a project.

package cli

import (
	"errors"
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
)

// collectSource prompts only for a missing repository and, when neither groups nor rules were supplied, for groups.
// The optional ref is never prompted for. Explicitly supplied flags are preserved.
func (f *authoringFlags) collectSource(groups *[]string, ruleIDs []string) error {
	if f.value("ref") != "" {
		if _, err := project.ParseSourceRef(f.value("ref")); err != nil {
			return usage(err)
		}
	}
	if err := f.require("repository"); err != nil {
		return err
	}
	if len(*groups) == 0 && len(ruleIDs) == 0 {
		if !f.interactive() {
			return usage(errors.New("supply at least one --groups or --rules"))
		}
		text, err := f.askValidated("Groups (comma-separated paths, *, practices/*, or techs/*):", func(value string) error {
			if err := validateAnswer("groups", value); err != nil {
				return err
			}
			_, err := project.ParseSourceGroups(splitPromptGroups(value))
			return err
		})
		if err != nil {
			return err
		}
		*groups = splitPromptGroups(text)
	}
	return nil
}

// splitPromptGroups trims each comma-separated path before domain validation.
func splitPromptGroups(answer string) []string {
	groups := strings.Split(answer, ",")
	for i := range groups {
		groups[i] = strings.TrimSpace(groups[i])
	}
	return groups
}
