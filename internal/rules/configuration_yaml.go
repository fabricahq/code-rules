// Decode and edit project YAML while retaining comments and scalar styles on existing entries.

package rules

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"

	"go.yaml.in/yaml/v4"
)

// ParseConfigurationYAML validates one UTF-8 YAML document using the configuration schema.
// Only string-keyed mappings, sequences, and JSON scalar types are supported; aliases and tags are rejected.
func ParseConfigurationYAML(input []byte) (Configuration, error) {
	_, data, err := configurationYAML(input)
	if err != nil {
		return Configuration{}, err
	}
	return ParseConfiguration(data)
}

func configurationYAML(input []byte) (*yaml.Node, []byte, error) {
	if !utf8.Valid(input) {
		return nil, nil, invalid("configuration", "expected UTF-8 YAML")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, invalid("configuration", "expected one YAML document")
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, nil, invalid("configuration", "expected one YAML document")
	}
	if len(document.Content) != 1 {
		return nil, nil, invalid("configuration", "expected a mapping")
	}
	value, err := configurationValue(document.Content[0], "configuration", 0)
	if err != nil {
		return nil, nil, err
	}
	data, err := json.Marshal(value)
	return &document, data, err
}

func configurationValue(node *yaml.Node, location string, depth int) (any, error) {
	if depth > 32 {
		return nil, invalid(location, "configuration nesting is too deep")
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode || node.Style&yaml.TaggedStyle != 0 {
		return nil, invalid(location, "anchors, aliases, and explicit tags are not supported")
	}
	switch node.Kind {
	case yaml.MappingNode:
		result := map[string]any{}
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Anchor != "" || key.Style&yaml.TaggedStyle != 0 {
				return nil, invalid(location, "mapping keys must be strings; quote wildcard selectors")
			}
			if _, exists := result[key.Value]; exists {
				return nil, invalid(location+"."+key.Value, "duplicate field")
			}
			value, err := configurationValue(node.Content[i+1], location+"."+key.Value, depth+1)
			if err != nil {
				return nil, err
			}
			result[key.Value] = value
		}
		return result, nil
	case yaml.SequenceNode:
		result := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			value, err := configurationValue(child, location, depth+1)
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
				return nil, invalid(location, "invalid scalar")
			}
			if _, err := json.Marshal(value); err != nil {
				return nil, invalid(location, "expected a finite JSON scalar")
			}
			return value, nil
		}
	}
	return nil, invalid(location, "unsupported YAML value; quote strings such as dates and wildcard selectors")
}

// AppendConfigurationSource adds a source without dropping existing comments or reordering entries.
// The complete resulting configuration is validated before any bytes are returned.
func AppendConfigurationSource(input []byte, alias string, source Source) ([]byte, error) {
	document, _, err := configurationYAML(input)
	if err != nil {
		return nil, err
	}
	if _, err := ParseConfigurationYAML(input); err != nil {
		return nil, err
	}
	root := document.Content[0]
	var sources *yaml.Node
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "sources" {
			sources = root.Content[i+1]
		}
	}
	for i := 0; i < len(sources.Content); i += 2 {
		if sources.Content[i].Value == alias {
			return nil, invalid("sources."+alias, "source already exists")
		}
	}
	var groups any = source.Groups.Groups
	if source.Groups.Pattern != "" {
		groups = source.Groups.Pattern
	}
	declaration := struct {
		Repository string                 `yaml:"repository"`
		Ref        string                 `yaml:"ref,omitempty"`
		Version    string                 `yaml:"version,omitempty"`
		Groups     any                    `yaml:"groups"`
		Exclude    map[string]string      `yaml:"exclude"`
		Replace    map[string]Replacement `yaml:"replace"`
	}{source.Repository, source.Ref, source.Version, groups, source.Exclude, source.Replace}
	var entry yaml.Node
	if err := entry.Encode(declaration); err != nil {
		return nil, err
	}
	sources.Content = append(sources.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: alias}, &entry)
	sources.Style = 0
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	if _, err := ParseConfigurationYAML(out.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
