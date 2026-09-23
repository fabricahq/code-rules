// Collect and validate the revision and group selection for a library imported into a project.

package cli

import (
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
)

// collectSource prompts only for missing source inputs and preserves explicitly supplied flags.
func (f *authoringFlags) collectSource(groups *[]string) error {
	if f.value("ref") != "" {
		if _, _, err := project.ParseSourceRef(f.value("ref")); err != nil {
			return usage(err)
		}
	}
	if err := f.require("repository", "ref"); err != nil {
		return err
	}
	if len(*groups) == 0 {
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
