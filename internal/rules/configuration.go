// Parse project source declarations without resolving Git refs or reading files.

package rules

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// Configuration contains sources in alias order; an empty list is a local-only project.
type Configuration struct {
	Sources []Source `json:"sources"`
}

// Source declares one remote library, the rules it selects, and the project's pins and exceptions.
// Rule IDs in Rules, Pins, and Exclude are library-relative and checked for syntax only;
// whether each names a rule the library supplies is checked against the imported catalog.
type Source struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	// Ref is the authored tag or full commit SHA, or empty when the source follows rule versions. GitRef parses it.
	Ref string `json:"ref,omitempty"`
	// Groups is an empty explicit list when the source selects only individual rules.
	Groups GroupSelection `json:"groups"`
	// Rules lists individually selected rule IDs in sorted order; it is empty, never nil, when there are none.
	Rules []string `json:"rules"`
	// Pins is empty, never nil, when no rule is pinned, and always empty when Ref is set.
	Pins map[string]Pin `json:"pins"`
	// Exclude is empty, never nil, when the source has no exceptions.
	Exclude map[string]Exclusion `json:"exclude"`
}

// GitRef returns the source's ref, parsed and normalized. It returns false and no error when the source has no
// ref. Ref is exported, so a Source built without ParseConfiguration can hold any text; text that isn't a tag or
// full commit SHA fails with a *ValidationError at sources.<name>.ref.
func (s Source) GitRef() (GitRef, bool, error) {
	if s.Ref == "" {
		return GitRef{}, false, nil
	}
	ref, err := ParseGitRef(s.Ref, "sources."+s.Name+".ref")
	if err != nil {
		return GitRef{}, false, err
	}
	return ref, true, nil
}

// SameRef reports whether ref, such as the ref a snapshot recorded for the source, and the source's ref are the
// same normalized reference once parsed, so release/5 and refs/tags/release/5 match. An empty ref matches only a
// source without one. It fails with a *ValidationError when either ref isn't a tag or full commit SHA.
func (s Source) SameRef(ref string) (bool, error) {
	own, hasRef, err := s.GitRef()
	if err != nil {
		return false, err
	}
	if ref == "" || !hasRef {
		return ref == "" && !hasRef, nil
	}
	other, err := ParseGitRef(ref, "ref")
	if err != nil {
		return false, err
	}
	return other == own, nil
}

// Pin keeps one rule at an exact published version, with the project's reason.
type Pin struct {
	Version RuleVersion `json:"version" yaml:"version"`
	Reason  string      `json:"reason" yaml:"reason"`
}

// Exclusion leaves one rule out of generated guidance, with the project's reason.
type Exclusion struct {
	Reason string `json:"reason" yaml:"reason"`
	// ReplacedBy is empty, or a contained path under local/ naming the local rule agents read instead.
	ReplacedBy string `json:"replacedBy,omitempty" yaml:"replacedBy,omitempty"`
}

var sourceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ParseConfiguration validates schema version 1, rejecting unknown fields and
// contradictory source declarations. Errors return no partial configuration.
// Strings retain authored spacing. Source aliases, group IDs, and rule IDs are sorted.
func ParseConfiguration(input json.RawMessage) (Configuration, error) {
	fields, err := jsonObject(input, "configuration")
	if err != nil {
		return Configuration{}, err
	}
	if _, ok := fields["localGroups"]; ok {
		return Configuration{}, invalid("localGroups", "remove localGroups; local groups are discovered from local/<group>/_group.yaml")
	}
	if err := knownJSONFields(fields, []string{"schemaVersion", "sources"}, "configuration"); err != nil {
		return Configuration{}, err
	}
	var schema float64
	if json.Unmarshal(fields["schemaVersion"], &schema) != nil || schema != 1 {
		return Configuration{}, invalid("schemaVersion", "only version 1 is supported")
	}
	sources, err := jsonObject(fields["sources"], "sources")
	if err != nil {
		return Configuration{}, err
	}
	result := Configuration{Sources: make([]Source, 0, len(sources))}
	repositories := make(map[string]bool)
	for _, name := range slices.Sorted(maps.Keys(sources)) {
		source, err := parseSource(name, sources[name], repositories)
		if err != nil {
			return Configuration{}, err
		}
		result.Sources = append(result.Sources, source)
	}
	return result, nil
}

// parseSource validates identity, then revision, then selection, then pins and exclusions.
func parseSource(name string, input json.RawMessage, repositories map[string]bool) (Source, error) {
	where := "sources." + name
	if !sourceNamePattern.MatchString(name) || name == "local" {
		return Source{}, invalid(where, "invalid or reserved source name")
	}
	fields, err := jsonObject(input, where)
	if err != nil {
		return Source{}, err
	}
	if err := knownJSONFields(fields, []string{"repository", "groups", "rules", "pins", "ref", "exclude"}, where); err != nil {
		return Source{}, err
	}
	repository, err := jsonText(fields["repository"], where+".repository")
	if err != nil {
		return Source{}, err
	}
	address, err := ParseRepository(fields["repository"], where+".repository")
	if err != nil {
		return Source{}, err
	}
	if repositories[address.Identity] {
		return Source{}, invalid(where, "repository "+repository+" is declared more than once")
	}
	repositories[address.Identity] = true
	result := Source{Name: name, Repository: repository}
	if raw, ok := fields["ref"]; ok {
		if _, pinned := fields["pins"]; pinned {
			return Source{}, invalid(where, "pins and ref can't be combined; remove ref to pin individual rules, or remove pins to import one revision")
		}
		if first := strings.TrimSpace(string(raw)); first != "" && strings.ContainsRune("-0123456789", rune(first[0])) {
			return Source{}, invalid(where+".ref", "expected a tag or full commit SHA in quotes; YAML reads an unquoted value of digits as a number")
		}
		text, err := jsonText(raw, where+".ref")
		if err != nil {
			return Source{}, err
		}
		if _, err := ParseGitRef(text, where+".ref"); err != nil {
			return Source{}, err
		}
		result.Ref = text
	}
	if result.Groups, result.Rules, err = parseSelection(fields, where); err != nil {
		return Source{}, err
	}
	if result.Pins, err = parsePins(fields["pins"], where+".pins"); err != nil {
		return Source{}, err
	}
	if result.Exclude, err = parseExclusions(fields["exclude"], where+".exclude"); err != nil {
		return Source{}, err
	}
	return result, nil
}

// parseSelection returns the group selection and sorted rule IDs, requiring at least one group,
// wildcard, or rule. A missing groups field is an empty explicit list.
func parseSelection(fields map[string]json.RawMessage, where string) (GroupSelection, []string, error) {
	groups := GroupSelection{Groups: []string{}}
	if raw, ok := fields["groups"]; ok {
		var err error
		if groups, err = ParseGroupSelection(raw, where+".groups"); err != nil {
			return GroupSelection{}, nil, err
		}
	}
	ids := []string{}
	if raw, ok := fields["rules"]; ok {
		var err error
		if ids, err = ParseRuleList(raw, where+".rules"); err != nil {
			return GroupSelection{}, nil, err
		}
	}
	if groups.Pattern == "" && len(groups.Groups) == 0 && len(ids) == 0 {
		return GroupSelection{}, nil, invalid(where, "select at least one group in groups or one rule in rules")
	}
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
		text, err := jsonText(item, fmt.Sprintf("%s[%d]", location, i))
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
		if err := ValidateRuleID(id, fmt.Sprintf("%s[%d]", location, i)); err != nil {
			return nil, err
		}
	}
	// Valid IDs are ASCII, so byte order is also code-point order.
	slices.Sort(ids)
	return ids, nil
}

// parsePins validates each pin's rule ID, exact version, and reason; a missing field is no pins.
func parsePins(input json.RawMessage, location string) (map[string]Pin, error) {
	result := map[string]Pin{}
	if input == nil {
		return result, nil
	}
	entries, err := jsonObject(input, location)
	if err != nil {
		return nil, err
	}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		where := location + "." + id
		if err := ValidateRuleID(id, where); err != nil {
			return nil, err
		}
		fields, err := jsonObject(entries[id], where)
		if err != nil {
			return nil, err
		}
		if err := knownJSONFields(fields, []string{"version", "reason"}, where); err != nil {
			return nil, err
		}
		var text string
		if json.Unmarshal(fields["version"], &text) != nil {
			return nil, invalid(where+".version", `expected an exact rule version in quotes, such as "1.3.0"`)
		}
		version, err := ParseRuleVersion(text, where+".version")
		if err != nil {
			return nil, err
		}
		reason, err := jsonText(fields["reason"], where+".reason")
		if err != nil {
			return nil, err
		}
		result[id] = Pin{Version: version, Reason: reason}
	}
	return result, nil
}

// parseExclusions validates each exclusion's rule ID, reason, and optional contained local replacement.
// A missing field is no exclusions. Whether the replacement file exists is checked when rules resolve.
func parseExclusions(input json.RawMessage, location string) (map[string]Exclusion, error) {
	result := map[string]Exclusion{}
	if input == nil {
		return result, nil
	}
	entries, err := jsonObject(input, location)
	if err != nil {
		return nil, err
	}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		where := location + "." + id
		if err := ValidateRuleID(id, where); err != nil {
			return nil, err
		}
		fields, err := jsonObject(entries[id], where)
		if err != nil {
			return nil, err
		}
		if err := knownJSONFields(fields, []string{"reason", "replacedBy"}, where); err != nil {
			return nil, err
		}
		reason, err := jsonText(fields["reason"], where+".reason")
		if err != nil {
			return nil, err
		}
		exclusion := Exclusion{Reason: reason}
		if raw, ok := fields["replacedBy"]; ok {
			file, err := jsonPath(raw, where+".replacedBy")
			if err != nil {
				return nil, err
			}
			if !strings.HasPrefix(file, "local/") {
				return nil, invalid(where+".replacedBy", "replacement files must be under local/")
			}
			exclusion.ReplacedBy = file
		}
		result[id] = exclusion
	}
	return result, nil
}
