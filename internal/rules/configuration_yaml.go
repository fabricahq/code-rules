// Decode and edit project YAML while retaining comments and existing scalar values.

package rules

import (
	"bytes"
	"maps"
	"slices"

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
// Pinned versions are written quoted, such as version: "1.3.0".
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
	sources := mappingValue(document.Content[0], "sources")
	if mappingValue(sources, alias) != nil {
		return nil, invalid("sources."+alias, "source already exists")
	}
	var groups any
	if source.Groups.Pattern != "" {
		groups = source.Groups.Pattern
	} else if len(source.Groups.Groups) > 0 {
		groups = source.Groups.Groups
	}
	declaration := struct {
		Repository string   `yaml:"repository"`
		Groups     any      `yaml:"groups,omitempty"`
		Rules      []string `yaml:"rules,omitempty"`
		Ref        string   `yaml:"ref,omitempty"`
	}{source.Repository, groups, source.Rules, source.Ref.String()}
	var entry yaml.Node
	if err := entry.Encode(declaration); err != nil {
		return nil, err
	}
	if err := addEntries(&entry, "pins", pinNodes(source.Pins), "sources."+alias); err != nil {
		return nil, err
	}
	if err := addEntries(&entry, "exclude", exclusionNodes(source.Exclude), "sources."+alias); err != nil {
		return nil, err
	}
	sources.Content = append(sources.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: alias}, &entry)
	sources.Style = 0
	return encodeConfiguration(document)
}

// SourceEdit lists pins and exclusions to add to one source; either map may be empty or nil.
type SourceEdit struct {
	Pins    map[string]Pin
	Exclude map[string]Exclusion
}

// EditConfigurationSource adds pins and exclusions to the existing source alias without dropping comments or
// reordering entries. New entries follow the source's existing ones in rule ID order, in a pins or exclude map
// that is created when absent. Versions are written quoted, such as version: "1.3.0", so YAML reads them as text.
// A rule the source already pins or excludes fails rather than being replaced. Folded scalars use literal style,
// as in AppendConfigurationSource. The complete resulting configuration is validated before any bytes are returned.
func EditConfigurationSource(input []byte, alias string, edit SourceEdit) ([]byte, error) {
	document, data, err := authoredYAML(input, "configuration")
	if err != nil {
		return nil, err
	}
	if _, err := ParseConfiguration(data); err != nil {
		return nil, err
	}
	source := mappingValue(mappingValue(document.Content[0], "sources"), alias)
	if source == nil {
		return nil, invalid("sources."+alias, "source doesn't exist")
	}
	source.Style = 0
	if err := addEntries(source, "pins", pinNodes(edit.Pins), "sources."+alias); err != nil {
		return nil, err
	}
	if err := addEntries(source, "exclude", exclusionNodes(edit.Exclude), "sources."+alias); err != nil {
		return nil, err
	}
	return encodeConfiguration(document)
}

// mappingValue returns the value of key in mapping, or nil when mapping is nil or lacks the key.
func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// addEntries appends entries, in key order, to the map field of source, creating the field when it's absent and
// entries isn't empty. A key the map already holds fails with its location under where.
func addEntries(source *yaml.Node, field string, entries map[string]*yaml.Node, where string) error {
	if len(entries) == 0 {
		return nil
	}
	target := mappingValue(source, field)
	if target == nil {
		target = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		source.Content = append(source.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: field}, target)
	}
	target.Style = 0
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		if mappingValue(target, id) != nil {
			return invalid(where+"."+field+"."+id, "the source already has this entry; edit it in the configuration instead")
		}
		target.Content = append(target.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: id}, entries[id])
	}
	return nil
}

// pinNodes encodes each pin as a mapping whose version is double-quoted.
func pinNodes(pins map[string]Pin) map[string]*yaml.Node {
	nodes := map[string]*yaml.Node{}
	for id, pin := range pins {
		nodes[id] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "version"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: pin.Version.String(), Style: yaml.DoubleQuotedStyle},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "reason"},
			textNode(pin.Reason),
		}}
	}
	return nodes
}

// exclusionNodes encodes each exclusion as a mapping with its reason and, when set, replacedBy.
func exclusionNodes(exclude map[string]Exclusion) map[string]*yaml.Node {
	nodes := map[string]*yaml.Node{}
	for id, exclusion := range exclude {
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "reason"}, textNode(exclusion.Reason)}}
		if exclusion.ReplacedBy != "" {
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "replacedBy"}, textNode(exclusion.ReplacedBy))
		}
		nodes[id] = node
	}
	return nodes
}

// textNode returns a string scalar in the style the encoder chooses for text, quoting it when YAML would
// otherwise read another type.
func textNode(text string) *yaml.Node {
	node := &yaml.Node{}
	// Encoding a string never fails.
	_ = node.Encode(text)
	return node
}

// encodeConfiguration writes an edited document with two-space indentation, keeping folded scalars' values, and
// validates the result.
func encodeConfiguration(document *yaml.Node) ([]byte, error) {
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
