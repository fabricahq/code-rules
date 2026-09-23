// Decode and edit project YAML while retaining comments and scalar styles on existing entries.

package rules

import (
	"bytes"

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
	return authoredYAML(input, "configuration")
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
