// Parse a group's display text and reading guidance without accessing files.

package rules

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GroupMetadata describes a group for selection. Text is preserved, not trimmed.
type GroupMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// WhenToRead retains input order and owns its storage. Successful parsing
	// always returns a non-nil slice, including an empty list.
	WhenToRead []string `json:"whenToRead"`
}

// ParseGroupMetadata validates a group's JSON document. Unknown fields are
// ignored except license and licenses, which belong to the library manifest.
// Field names are case-sensitive; repeated JSON keys use their last value.
// On error, the returned metadata is the zero value.
func ParseGroupMetadata(input json.RawMessage, location string) (GroupMetadata, error) {
	if !json.Valid(input) {
		return GroupMetadata{}, invalid(location, "invalid JSON")
	}
	// Raw fields preserve exact key matching and defer decoding ignored values.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil || fields == nil {
		return GroupMetadata{}, invalid(location, "expected an object")
	}
	for _, key := range []string{"license", "licenses"} {
		if _, present := fields[key]; present {
			return GroupMetadata{}, invalid(location+"."+key, "declare one license for the whole library in rule-library.json; group-level licenses are unsupported")
		}
	}
	name, err := metadataText(fields["name"], location+".name")
	if err != nil {
		return GroupMetadata{}, err
	}
	description, err := metadataText(fields["description"], location+".description")
	if err != nil {
		return GroupMetadata{}, err
	}
	guidance, err := readingGuidance(fields["whenToRead"], location+".whenToRead")
	if err != nil {
		return GroupMetadata{}, err
	}
	return GroupMetadata{Name: name, Description: description, WhenToRead: guidance}, nil
}

func metadataText(input json.RawMessage, location string) (string, error) {
	var text string
	if err := json.Unmarshal(input, &text); err != nil || strings.TrimFunc(text, jsWhitespace) == "" {
		return "", invalid(location, "expected nonempty text")
	}
	return text, nil
}

func readingGuidance(input json.RawMessage, location string) ([]string, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(input, &items); err != nil || items == nil {
		return nil, invalid(location, "expected an array of strings")
	}
	guidance := make([]string, len(items))
	for i, item := range items {
		text, err := metadataText(item, fmt.Sprintf("%s[%d]", location, i))
		if err != nil {
			return nil, err
		}
		guidance[i] = text
	}
	// All item text is checked before duplicates, matching the reference order.
	seen := make(map[string]bool, len(guidance))
	for _, text := range guidance {
		if seen[text] {
			return nil, invalid(location, "duplicate entries")
		}
		seen[text] = true
	}
	return guidance, nil
}
