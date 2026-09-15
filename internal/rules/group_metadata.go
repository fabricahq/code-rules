// Parse a group's display text and reading guidance without accessing files.

package rules

import (
	"encoding/json"
	"strings"
)

// GroupMetadata describes a group for selection. Text has no surrounding whitespace.
type GroupMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WhenToRead  string `json:"whenToRead"`
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
	guidance, err := metadataText(fields["whenToRead"], location+".whenToRead")
	if err != nil {
		return GroupMetadata{}, err
	}
	return GroupMetadata{Name: name, Description: description, WhenToRead: guidance}, nil
}

func metadataText(input json.RawMessage, location string) (string, error) {
	var text string
	if err := json.Unmarshal(input, &text); err != nil {
		return "", invalid(location, "expected nonempty text")
	}
	text = strings.TrimFunc(text, jsWhitespace)
	if text == "" {
		return "", invalid(location, "expected nonempty text")
	}
	return text, nil
}
