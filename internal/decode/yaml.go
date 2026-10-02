// Decode authored YAML with one shared strict policy before applying each document schema.

package decode

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"

	"go.yaml.in/yaml/v4"
)

// YAML decodes input like Document and also converts its content to JSON like JSON, for schemas that read every
// field.
func YAML(input []byte, location string) (*yaml.Node, []byte, error) {
	document, err := Document(input, location)
	if err != nil {
		return nil, nil, err
	}
	data, err := JSON(document.Content[0], location)
	return document, data, err
}

// Document decodes exactly one YAML document with content and checks the rules that hold for every node
// of an authored document: no anchors, aliases, explicit tags, or duplicate keys, scalar mapping keys, and at most
// 32 levels of nesting. It interprets no scalar values, so callers can ignore fields whose values they can't read.
func Document(input []byte, location string) (*yaml.Node, error) {
	if !utf8.Valid(input) {
		return nil, Invalid(location, "expected UTF-8 YAML")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, Invalid(location, "expected one YAML document")
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, Invalid(location, "expected one YAML document")
	}
	if len(document.Content) != 1 {
		return nil, Invalid(location, "expected a mapping")
	}
	if err := hygiene(document.Content[0], location, 0); err != nil {
		return nil, err
	}
	return &document, nil
}

// hygiene checks node and everything below it against Document's rules.
func hygiene(node *yaml.Node, location string, depth int) error {
	if depth > 32 {
		return Invalid(location, "YAML nesting is too deep")
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode || node.Style&yaml.TaggedStyle != 0 {
		return Invalid(location, "anchors, aliases, and explicit tags are not supported")
	}
	switch node.Kind {
	case yaml.MappingNode:
		seen := map[string]bool{}
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Anchor != "" || key.Style&yaml.TaggedStyle != 0 {
				return Invalid(location, "mapping keys must be strings; quote wildcard selectors")
			}
			if seen[key.Value] {
				return Invalid(location+"."+key.Value, "duplicate field")
			}
			seen[key.Value] = true
			if err := hygiene(node.Content[i+1], location+"."+key.Value, depth+1); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := hygiene(child, location, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// JSON converts node to JSON, accepting only strings, finite numbers, booleans, and null among scalars, as
// every authored field a reader interprets requires.
func JSON(node *yaml.Node, location string) ([]byte, error) {
	value, err := jsonValue(node, location, 0)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func jsonValue(node *yaml.Node, location string, depth int) (any, error) {
	if depth > 32 {
		return nil, Invalid(location, "YAML nesting is too deep")
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode || node.Style&yaml.TaggedStyle != 0 {
		return nil, Invalid(location, "anchors, aliases, and explicit tags are not supported")
	}
	switch node.Kind {
	case yaml.MappingNode:
		result := map[string]any{}
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Anchor != "" || key.Style&yaml.TaggedStyle != 0 {
				return nil, Invalid(location, "mapping keys must be strings; quote wildcard selectors")
			}
			if _, exists := result[key.Value]; exists {
				return nil, Invalid(location+"."+key.Value, "duplicate field")
			}
			value, err := jsonValue(node.Content[i+1], location+"."+key.Value, depth+1)
			if err != nil {
				return nil, err
			}
			result[key.Value] = value
		}
		return result, nil
	case yaml.SequenceNode:
		result := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			value, err := jsonValue(child, location, depth+1)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str":
			return node.Value, nil
		case "!!int", "!!float", "!!bool", "!!null":
			var value any
			if err := node.Decode(&value); err != nil {
				return nil, Invalid(location, "invalid scalar")
			}
			if _, err := json.Marshal(value); err != nil {
				return nil, Invalid(location, "expected a finite JSON scalar")
			}
			return value, nil
		}
	}
	return nil, Invalid(location, "unsupported YAML value; quote strings such as dates and wildcard selectors")
}
