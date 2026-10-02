// Parse group selections while preserving configuration diagnostics and ordering.

package rules

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/decode"
	"github.com/fabricahq/code-rules/internal/librarytree"
)

const selectionFormat = `expected an array of group IDs or one of "*", "techs/*", "practices/*"`

// GroupSelection is either a wildcard Pattern or explicit sorted Groups.
// A nonempty Pattern means Groups is nil. Otherwise Groups is non-nil, including
// an empty selection. Parsing never discovers groups or expands wildcards.
type GroupSelection struct {
	Pattern string
	Groups  []string
}

// ParseGroupSelection reads a configuration groups JSON value. It accepts the
// three wildcard strings or distinct group IDs in an array. Missing and null
// inputs are invalid. Element text errors precede duplicates, then ID errors;
// locations retain original indices even though the returned IDs are sorted.
func ParseGroupSelection(input json.RawMessage, location string) (GroupSelection, error) {
	if len(input) == 0 {
		return GroupSelection{}, invalid(location, selectionFormat)
	}
	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return GroupSelection{}, invalid(location, "expected a groups JSON value")
	}
	if pattern, ok := value.(string); ok && (pattern == "*" || pattern == "techs/*" || pattern == "practices/*") {
		return GroupSelection{Pattern: pattern}, nil
	}
	items, ok := value.([]any)
	if !ok {
		return GroupSelection{}, invalid(location, selectionFormat)
	}
	groups := make([]string, len(items))
	for i, item := range items {
		text, ok := item.(string)
		if !ok || strings.TrimFunc(text, decode.IsSpace) == "" {
			return GroupSelection{}, invalid(fmt.Sprintf("%s[%d]", location, i), "expected nonempty text")
		}
		groups[i] = text
	}
	seen := make(map[string]bool, len(groups))
	for _, group := range groups {
		if seen[group] {
			return GroupSelection{}, invalid(location, "duplicate entries")
		}
		seen[group] = true
	}
	for i, group := range groups {
		if err := librarytree.ValidateGroupID(group, fmt.Sprintf("%s[%d]", location, i)); err != nil {
			return GroupSelection{}, err
		}
	}
	// Valid IDs are ASCII, so byte order also matches the reference's UTF-16 order.
	slices.Sort(groups)
	return GroupSelection{Groups: groups}, nil
}

// Includes reports whether the selection imports group in full: a group it lists, or any group within its
// wildcard's scope. It checks only the ID, not whether the group exists.
func (s GroupSelection) Includes(group string) bool {
	switch s.Pattern {
	case "*":
		return strings.HasPrefix(group, "techs/") || strings.HasPrefix(group, "practices/")
	case "techs/*", "practices/*":
		return strings.HasPrefix(group, strings.TrimSuffix(s.Pattern, "*"))
	}
	return slices.Contains(s.Groups, group)
}

// MarshalJSON preserves the configuration's string-or-array representation.
func (s GroupSelection) MarshalJSON() ([]byte, error) {
	if s.Pattern != "" {
		return json.Marshal(s.Pattern)
	}
	return json.Marshal(s.Groups)
}
