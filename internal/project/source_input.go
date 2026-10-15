// Interpret library-selection inputs independently of the project configuration's storage format.

package project

import (
	"encoding/json"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// SourceInput selects a repository, library groups, individual rules, and an optional exact ref.
// Groups may combine group paths with the wildcards *, techs/*, and practices/*. At least one group or rule is
// required. An empty Ref means the source follows rule versions. Pins and exclusions are configured separately.
type SourceInput struct {
	Repository string
	Ref        string
	Groups     []string
	Rules      []string
}

// ParseSourceRef returns the tag or full commit SHA, without surrounding whitespace, as a validated ref.
// Branch names, abbreviated commits, and version ranges are rejected.
func ParseSourceRef(input string) (rules.GitRef, error) {
	return rules.ParseGitRef(strings.TrimSpace(input), "--ref")
}

// ParseSourceGroups validates group paths and wildcards without interpreting comma-separated prompt text.
// No groups is an empty explicit selection.
func ParseSourceGroups(groups []string) (rules.GroupSelection, error) {
	data, err := json.Marshal(append([]string{}, groups...))
	if err != nil {
		return rules.GroupSelection{}, err
	}
	return rules.ParseGroupSelection(data, "--groups")
}

// ParseSourceRules validates distinct library rule IDs and returns them sorted; no rules is an empty list.
func ParseSourceRules(ids []string) ([]string, error) {
	data, err := json.Marshal(append([]string{}, ids...))
	if err != nil {
		return nil, err
	}
	return rules.ParseRuleList(data, "--rules")
}

func parseSourceInput(input SourceInput) (rules.Source, error) {
	source := rules.Source{Repository: input.Repository}
	var err error
	if input.Ref != "" {
		if source.Ref, err = ParseSourceRef(input.Ref); err != nil {
			return rules.Source{}, err
		}
	}
	if source.Groups, err = ParseSourceGroups(input.Groups); err != nil {
		return rules.Source{}, err
	}
	if source.Rules, err = ParseSourceRules(input.Rules); err != nil {
		return rules.Source{}, err
	}
	if source.Groups.IsEmpty() && len(source.Rules) == 0 {
		return rules.Source{}, &ValidationError{Location: "--groups", Problem: "supply at least one --groups or --rules"}
	}
	return source, nil
}
