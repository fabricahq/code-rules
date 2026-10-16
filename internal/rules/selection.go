// Parse rule selectors and group selections while preserving configuration diagnostics and ordering.

package rules

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/fabricahq/syllabus/internal/decode"
	"github.com/fabricahq/syllabus/internal/librarytree"
)

const selectionFormat = `expected one of "*", "techs/*", "practices/*", or an array of group IDs and those wildcards`

// GroupSelection names the groups a source imports in full: every group within Pattern's scope, and each of
// Groups. Pattern is "", "*", "techs/*", or "practices/*". Groups is sorted and may name groups within Pattern's
// scope; parsers return it empty, never nil, when no group is listed. Parsing never discovers groups or expands
// wildcards.
type GroupSelection struct {
	Pattern string
	Groups  []string
}

// ParseGroupSelection reads a groups JSON value, such as a source record's groupSelection or the --groups flag: one
// wildcard string, or an array of distinct group IDs and wildcards. Missing and null inputs are invalid. Wildcards
// combine, so techs/* with practices/* is "*". Element text errors precede duplicates, then ID errors; locations
// retain original indices even though the returned IDs are sorted.
func ParseGroupSelection(input json.RawMessage, location string) (GroupSelection, error) {
	if len(input) == 0 {
		return GroupSelection{}, invalid(location, selectionFormat)
	}
	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return GroupSelection{}, invalid(location, "expected a groups JSON value")
	}
	if pattern, ok := value.(string); ok && isGroupWildcard(pattern) {
		return GroupSelection{Pattern: pattern, Groups: []string{}}, nil
	}
	items, ok := value.([]any)
	if !ok {
		return GroupSelection{}, invalid(location, selectionFormat)
	}
	entries := make([]string, len(items))
	for i, item := range items {
		text, ok := item.(string)
		if !ok || strings.TrimFunc(text, decode.IsSpace) == "" {
			return GroupSelection{}, invalid(fmt.Sprintf("%s[%d]", location, i), "expected nonempty text")
		}
		entries[i] = text
	}
	if err := requireDistinct(entries, location); err != nil {
		return GroupSelection{}, err
	}
	selection := GroupSelection{Groups: []string{}}
	for i, entry := range entries {
		if isGroupWildcard(entry) {
			selection.Pattern = joinPatterns(selection.Pattern, entry)
			continue
		}
		if err := librarytree.ValidateGroupID(entry, fmt.Sprintf("%s[%d]", location, i)); err != nil {
			return GroupSelection{}, err
		}
		selection.Groups = append(selection.Groups, entry)
	}
	// Valid IDs are ASCII, so byte order also matches the reference's UTF-16 order.
	slices.Sort(selection.Groups)
	return selection, nil
}

// ParseRuleSelection reads a rules.include JSON array of distinct selectors, each "*", "techs/*", "practices/*", a
// group followed by /*, such as techs/go/*, or a rule ID. It returns the groups the selectors import in full and
// the individually selected rule IDs, sorted and empty, never nil, when there are none. An empty array is valid
// here; whether a source must select something is the caller's decision. Element text errors precede duplicates,
// then selector syntax errors, each at its original index.
func ParseRuleSelection(input json.RawMessage, location string) (GroupSelection, []string, error) {
	var items []json.RawMessage
	if json.Unmarshal(input, &items) != nil || items == nil {
		return GroupSelection{}, nil, invalid(location, `expected an array of selectors, such as ["techs/go/*", "practices/testing/verify-retry-limits"]`)
	}
	entries := make([]string, len(items))
	for i, item := range items {
		text, err := decode.Text(item, fmt.Sprintf("%s[%d]", location, i))
		if err != nil {
			return GroupSelection{}, nil, err
		}
		entries[i] = text
	}
	if err := requireDistinct(entries, location); err != nil {
		return GroupSelection{}, nil, err
	}
	groups := GroupSelection{Groups: []string{}}
	ids := []string{}
	for i, entry := range entries {
		parsed, err := parseSelector(entry, fmt.Sprintf("%s[%d]", location, i))
		if err != nil {
			return GroupSelection{}, nil, err
		}
		switch {
		case parsed.pattern != "":
			groups.Pattern = joinPatterns(groups.Pattern, parsed.pattern)
		case parsed.group != "":
			groups.Groups = append(groups.Groups, parsed.group)
		default:
			ids = append(ids, parsed.rule)
		}
	}
	// Valid IDs are ASCII, so byte order is also code-point order.
	slices.Sort(groups.Groups)
	slices.Sort(ids)
	return groups, ids, nil
}

// ParseRuleList validates a JSON array of distinct library rule IDs, reporting each original index, and returns them sorted.
// Element text errors precede duplicates, then ID syntax errors, matching group selection.
func ParseRuleList(input json.RawMessage, location string) ([]string, error) {
	var items []json.RawMessage
	if json.Unmarshal(input, &items) != nil || items == nil {
		return nil, invalid(location, "expected an array of rule IDs, such as practices/testing/verify-retry-limits")
	}
	ids := make([]string, len(items))
	for i, item := range items {
		text, err := decode.Text(item, fmt.Sprintf("%s[%d]", location, i))
		if err != nil {
			return nil, err
		}
		ids[i] = text
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return nil, invalid(location, "duplicate entries")
		}
		seen[id] = true
	}
	for i, id := range ids {
		if err := librarytree.ValidateRuleID(id, fmt.Sprintf("%s[%d]", location, i)); err != nil {
			return nil, err
		}
	}
	// Valid IDs are ASCII, so byte order is also code-point order.
	slices.Sort(ids)
	return ids, nil
}

// selector is one parsed rule selector; exactly one of its fields is set: pattern for "*", "techs/*", or
// "practices/*", group for a group's selector, such as techs/go for techs/go/*, or rule for a rule ID.
type selector struct {
	pattern, group, rule string
}

// parseSelector classifies one rule selector. A * is accepted only alone or as the final segment after a slash,
// and a selector ending in /* must name a group or one type of group, never a directory inside a group. A bare
// group ID is rejected, because it reads as a rule ID; the error names its group selector instead.
func parseSelector(text, location string) (selector, error) {
	prefix, wildcard := strings.CutSuffix(text, "/*")
	if text == "*" {
		prefix, wildcard = "", true
	}
	if strings.Contains(prefix, "*") {
		return selector{}, invalid(location, "invalid selector "+decode.Quote(text)+`: * is allowed only at the end, after a slash, such as techs/go/*, or alone as "*"`)
	}
	if !wildcard {
		if librarytree.ValidateGroupID(text, location) == nil {
			return selector{}, invalid(location, decode.Quote(text)+" is a group ID; select the whole group with "+text+"/*")
		}
		if err := librarytree.ValidateRuleID(text, location); err != nil {
			return selector{}, err
		}
		return selector{rule: text}, nil
	}
	if isGroupWildcard(text) {
		return selector{pattern: text}, nil
	}
	if strings.Count(prefix, "/") > 1 {
		return selector{}, invalid(location, "invalid selector "+decode.Quote(text)+": a selector ending in /* selects a group, such as techs/go/*, or every group of one type, techs/* or practices/*")
	}
	if err := librarytree.ValidateGroupID(prefix, location); err != nil {
		return selector{}, err
	}
	return selector{group: prefix}, nil
}

// SelectorMatches reports whether the valid rule selector text selects rule id: the rule itself, a rule in the
// group it names, or any rule in its wildcard's scope. It checks only the IDs, not whether the rule exists.
func SelectorMatches(text, id string) bool {
	if text == "*" {
		return true
	}
	if prefix, ok := strings.CutSuffix(text, "*"); ok {
		return strings.HasPrefix(id, prefix)
	}
	return text == id
}

// isSelectorPattern reports whether a valid rule selector selects more than one rule ID.
func isSelectorPattern(text string) bool {
	return strings.HasSuffix(text, "*")
}

// requireDistinct rejects the first entry that repeats an earlier one, at the list's location.
func requireDistinct(entries []string, location string) error {
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if seen[entry] {
			return invalid(location, "duplicate entries")
		}
		seen[entry] = true
	}
	return nil
}

// isGroupWildcard reports whether pattern selects every group, or every group of one type.
func isGroupWildcard(pattern string) bool {
	return pattern == "*" || pattern == "techs/*" || pattern == "practices/*"
}

// joinPatterns returns the narrowest wildcard whose scope covers both a and b, either of which may be empty.
func joinPatterns(a, b string) string {
	switch {
	case a == "" || a == b:
		return b
	case b == "":
		return a
	}
	return "*"
}

// Includes reports whether the selection imports group in full: a group it lists, or any group within its
// wildcard's scope. It checks only the ID, not whether the group exists.
func (s GroupSelection) Includes(group string) bool {
	switch s.Pattern {
	case "*":
		return strings.HasPrefix(group, "techs/") || strings.HasPrefix(group, "practices/")
	case "techs/*", "practices/*":
		if strings.HasPrefix(group, strings.TrimSuffix(s.Pattern, "*")) {
			return true
		}
	}
	return slices.Contains(s.Groups, group)
}

// Covers reports whether the selection imports in full every group that other does, judged by their wildcards and
// listed groups alone: a listed group it lacks counts as uncovered even when a library has no such group.
func (s GroupSelection) Covers(other GroupSelection) bool {
	if other.Pattern != "" && joinPatterns(s.Pattern, other.Pattern) != s.Pattern {
		return false
	}
	for _, group := range other.Groups {
		if !s.Includes(group) {
			return false
		}
	}
	return true
}

// Equal reports whether two selections name the same wildcard and the same listed groups; a nil and an empty
// Groups are equal.
func (s GroupSelection) Equal(other GroupSelection) bool {
	return s.Pattern == other.Pattern && slices.Equal(s.Groups, other.Groups)
}

// IsEmpty reports whether the selection imports no group in full.
func (s GroupSelection) IsEmpty() bool {
	return s.Pattern == "" && len(s.Groups) == 0
}

// Roots returns, sorted, the top-level directories whose groups the wildcard covers: none without one, techs or
// practices for one type, or both for "*".
func (s GroupSelection) Roots() []string {
	switch s.Pattern {
	case "*":
		return []string{"practices", "techs"}
	case "":
		return []string{}
	}
	return []string{strings.TrimSuffix(s.Pattern, "/*")}
}

// MarshalJSON writes a selection with only a wildcard as that string, and any other as a sorted array of its
// listed groups and wildcard, the representation ParseGroupSelection reads.
func (s GroupSelection) MarshalJSON() ([]byte, error) {
	if s.Pattern != "" && len(s.Groups) == 0 {
		return json.Marshal(s.Pattern)
	}
	entries := append([]string{}, s.Groups...)
	if s.Pattern != "" {
		entries = append(entries, s.Pattern)
	}
	slices.Sort(entries)
	return json.Marshal(entries)
}

// UnmarshalJSON parses data with ParseGroupSelection, so MarshalJSON's output round-trips. Invalid data fails with
// a *ValidationError and leaves s unchanged.
func (s *GroupSelection) UnmarshalJSON(data []byte) error {
	parsed, err := ParseGroupSelection(data, "groups")
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
