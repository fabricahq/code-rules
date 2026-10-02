// Parse a group's display text and reading guidance without accessing files.

package coderules

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/decode"
)

// GroupMetadata describes a group for selection, as its _group.yaml file declares it. Text has no surrounding
// whitespace.
type GroupMetadata struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	WhenToRead  string `json:"whenToRead" yaml:"whenToRead"`
}

// ParseGroupMetadata parses a group's _group.yaml file. location names the file in errors, such as
// practices/testing/_group.yaml. The file must be one YAML mapping with exactly the nonblank text fields name,
// description, and whenToRead, without anchors, aliases, explicit tags, or duplicate keys; license fields get a
// specific error, since a library declares one license for all its rules. On error, the returned metadata is the
// zero value.
func ParseGroupMetadata(input []byte, location string) (GroupMetadata, error) {
	_, data, err := decode.YAML(input, location)
	if err != nil {
		return GroupMetadata{}, err
	}
	return groupMetadataFields(data, location)
}

// groupMetadataFields validates group metadata decoded to JSON and rejects unknown fields.
// Field names are case-sensitive; repeated JSON keys use their last value.
// Text fields reject malformed Unicode instead of silently replacing it.
// On error, the returned metadata is the zero value.
func groupMetadataFields(input json.RawMessage, location string) (GroupMetadata, error) {
	if !json.Valid(input) {
		return GroupMetadata{}, invalid(location, "invalid JSON")
	}
	// Raw fields preserve exact key matching and defer decoding field values.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil || fields == nil {
		return GroupMetadata{}, invalid(location, "expected an object")
	}
	for _, key := range []string{"license", "licenses"} {
		if _, present := fields[key]; present {
			return GroupMetadata{}, invalid(location+"."+key, "declare one license for the whole library in rule-library.yaml; group-level licenses are unsupported")
		}
	}
	// Sort keys so multiple unknown fields produce a deterministic first error.
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "name", "description", "whenToRead":
		default:
			return GroupMetadata{}, invalid(location+"."+key, "unknown field; allowed fields: name, description, whenToRead")
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

// metadataText returns a JSON string's text without surrounding whitespace, rejecting malformed Unicode, values
// that aren't strings, and blank text.
func metadataText(input json.RawMessage, location string) (string, error) {
	if len(input) > 0 && input[0] == '"' && !decode.ValidUnicode(input) {
		return "", invalid(location, "expected valid Unicode text: invalid UTF-8 or unpaired surrogate escape")
	}
	var text string
	if err := json.Unmarshal(input, &text); err != nil {
		return "", invalid(location, "expected nonempty text")
	}
	text = strings.TrimFunc(text, decode.IsSpace)
	if text == "" {
		return "", invalid(location, "expected nonempty text")
	}
	return text, nil
}
