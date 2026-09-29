// Decode and edit project YAML while retaining comments and existing scalar values.

package rules

import (
	"bytes"

	"go.yaml.in/yaml/v4"
)

// ParseConfigurationYAML validates one UTF-8 YAML document using the configuration schema.
// Only string-keyed mappings, sequences, and JSON scalar types are supported; aliases and tags are rejected.
func ParseConfigurationYAML(input []byte) (Configuration, error) {
	_, data, err := authoredYAML(input, "configuration")
	if err != nil {
		return Configuration{}, err
	}
	return ParseConfiguration(data)
}

// AppendConfigurationSource adds a source without dropping existing comments or reordering entries.
// Empty optional fields are omitted, and groups is omitted when only individual rules are selected.
// Folded scalars use literal style in the result because the YAML encoder can change their values.
// The complete resulting configuration is validated before any bytes are returned.
func AppendConfigurationSource(input []byte, alias string, source Source) ([]byte, error) {
	document, data, err := authoredYAML(input, "configuration")
	if err != nil {
		return nil, err
	}
	if _, err := ParseConfiguration(data); err != nil {
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
	var groups any
	if source.Groups.Pattern != "" {
		groups = source.Groups.Pattern
	} else if len(source.Groups.Groups) > 0 {
		groups = source.Groups.Groups
	}
	declaration := struct {
		Repository string               `yaml:"repository"`
		Groups     any                  `yaml:"groups,omitempty"`
		Rules      []string             `yaml:"rules,omitempty"`
		Ref        string               `yaml:"ref,omitempty"`
		Pins       map[string]Pin       `yaml:"pins,omitempty"`
		Exclude    map[string]Exclusion `yaml:"exclude,omitempty"`
	}{source.Repository, groups, source.Rules, source.Ref, source.Pins, source.Exclude}
	var entry yaml.Node
	if err := entry.Encode(declaration); err != nil {
		return nil, err
	}
	sources.Content = append(sources.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: alias}, &entry)
	sources.Style = 0
	normalizeFoldedScalars(document)
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

// The YAML encoder can change folded scalar values; literal style preserves them.
func normalizeFoldedScalars(node *yaml.Node) {
	if node.Style&yaml.FoldedStyle != 0 {
		node.Style = node.Style&^yaml.FoldedStyle | yaml.LiteralStyle
	}
	for _, child := range node.Content {
		normalizeFoldedScalars(child)
	}
}
