// Interpret library-selection inputs independently of the project configuration's storage format.

package project

import (
	"encoding/json"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// SourceInput selects a repository, one exact ref, and library groups.
// A single wildcard selector may replace explicit group paths; exclusions and replacements are configured separately.
type SourceInput struct {
	Repository string
	Ref        string
	Groups     []string
}

// ParseSourceRef returns the tag or full commit SHA without surrounding whitespace.
// Branch names, abbreviated commits, and version ranges are rejected.
func ParseSourceRef(input string) (string, error) {
	value := strings.TrimSpace(input)
	if _, err := rules.ParseGitRef(value, "--ref"); err != nil {
		return "", err
	}
	return value, nil
}

// ParseSourceGroups validates explicit paths or one wildcard without interpreting comma-separated prompt text.
func ParseSourceGroups(groups []string) (rules.GroupSelection, error) {
	var selection any = groups
	if len(groups) == 1 && (groups[0] == "*" || groups[0] == "techs/*" || groups[0] == "practices/*") {
		selection = groups[0]
	}
	data, err := json.Marshal(selection)
	if err != nil {
		return rules.GroupSelection{}, err
	}
	return rules.ParseGroupSelection(data, "--groups")
}

func parseSourceInput(input SourceInput) (rules.Source, error) {
	ref, err := ParseSourceRef(input.Ref)
	if err != nil {
		return rules.Source{}, err
	}
	groups, err := ParseSourceGroups(input.Groups)
	if err != nil {
		return rules.Source{}, err
	}
	return rules.Source{Repository: input.Repository, Ref: ref, Groups: groups}, nil
}
