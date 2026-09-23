// Collect and validate the revision and group selection for a library imported into a project.

package cli

import (
	"encoding/json"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// libraryRef returns the config field and validated value without resolving Git refs.
// Bare versions remain literal tags. refs/tags/ disambiguates tags that resemble constraints.
func libraryRef(input string) (field, value string, err error) {
	value = strings.TrimSpace(input)
	if !strings.HasPrefix(value, "refs/tags/") && value != "" &&
		(strings.ContainsAny(value[:1], "=!<>~^") || strings.Contains(value, ",")) {
		constraint, err := rules.ParseVersionConstraint(value, "--ref")
		if err != nil {
			return "", "", err
		}
		return "version", constraint.String(), nil
	}
	if _, err := rules.ParseGitRef(value, "--ref"); err != nil {
		return "", "", err
	}
	return "ref", value, nil
}

// collectSource prompts only for missing source inputs and preserves explicitly supplied flags.
func (f *authoringFlags) collectSource(groups *[]string) error {
	if f.value("ref") != "" {
		if _, _, err := libraryRef(f.value("ref")); err != nil {
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
			data, err := json.Marshal(sourceGroupSelection(splitPromptGroups(value)))
			if err != nil {
				return err
			}
			_, err = rules.ParseGroupSelection(data, "--groups")
			return err
		})
		if err != nil {
			return err
		}
		*groups = splitPromptGroups(text)
	}
	return nil
}

// sourceGroupSelection keeps prompt validation and the saved declaration on the same wildcard representation.
func sourceGroupSelection(groups []string) any {
	if len(groups) == 1 && (groups[0] == "*" || groups[0] == "techs/*" || groups[0] == "practices/*") {
		return groups[0]
	}
	return groups
}

// splitPromptGroups trims each comma-separated path before domain validation.
func splitPromptGroups(answer string) []string {
	groups := strings.Split(answer, ",")
	for i := range groups {
		groups[i] = strings.TrimSpace(groups[i])
	}
	return groups
}
