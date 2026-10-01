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

// SourceEdit lists pins and exclusions to add to one source, pins to remove from it, and basedOn versions to set;
// each may be empty or nil.
type SourceEdit struct {
	Pins    map[string]Pin
	Exclude map[string]Exclusion
	// Unpin names rules whose existing pins the edit removes.
	Unpin []string
	// BasedOn sets the basedOn version of each existing replacement it names, adding or replacing the field.
	BasedOn map[string]RuleVersion
}

// EditConfigurationSource adds pins and exclusions to the existing source alias, removes the pins edit.Unpin
// names, and sets the basedOn version of the existing replacements edit.BasedOn names, keeping a replaced value's
// comment, without dropping other comments or reordering entries. Setting basedOn of a rule the source doesn't
// replace fails. A pins map left empty is removed; a pin to remove
// that the source lacks fails. New entries follow the source's existing ones in rule ID order, in a pins or exclude map
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
	if err := removeEntries(source, "pins", edit.Unpin, "sources."+alias); err != nil {
		return nil, err
	}
	if err := addEntries(source, "pins", pinNodes(edit.Pins), "sources."+alias); err != nil {
		return nil, err
	}
	if err := addEntries(source, "exclude", exclusionNodes(edit.Exclude), "sources."+alias); err != nil {
		return nil, err
	}
	for _, id := range slices.Sorted(maps.Keys(edit.BasedOn)) {
		exclusion := mappingValue(mappingValue(source, "exclude"), id)
		if exclusion == nil || mappingValue(exclusion, "replacedBy") == nil {
			return nil, invalid("sources."+alias+".exclude."+id, "the source doesn't replace this rule, so it has no basedOn version to set")
		}
		version := edit.BasedOn[id]
		if existing := mappingValue(exclusion, "basedOn"); existing != nil {
			existing.Kind, existing.Tag, existing.Value, existing.Style = yaml.ScalarNode, "!!str", version.String(), yaml.DoubleQuotedStyle
			continue
		}
		exclusion.Content = append(exclusion.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "basedOn"}, versionNode(version))
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

// removeEntries deletes each of ids from the map field of source, and the field itself once it's empty. An ID the
// map lacks fails with its location under where.
func removeEntries(source *yaml.Node, field string, ids []string, where string) error {
	if len(ids) == 0 {
		return nil
	}
	target := mappingValue(source, field)
	for _, id := range ids {
		index := -1
		if target != nil {
			for i := 0; i+1 < len(target.Content); i += 2 {
				if target.Content[i].Value == id {
					index = i
				}
			}
		}
		if index < 0 {
			return invalid(where+"."+field+"."+id, "the source has no such entry to remove")
		}
		target.Content = slices.Delete(target.Content, index, index+2)
	}
	if len(target.Content) == 0 {
		for i := 0; i+1 < len(source.Content); i += 2 {
			if source.Content[i].Value == field {
				source.Content = slices.Delete(source.Content, i, i+2)
				break
			}
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
			versionNode(pin.Version),
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "reason"},
			textNode(pin.Reason),
		}}
	}
	return nodes
}

// exclusionNodes encodes each exclusion as a mapping with its reason and, when set, replacedBy and basedOn.
func exclusionNodes(exclude map[string]Exclusion) map[string]*yaml.Node {
	nodes := map[string]*yaml.Node{}
	for id, exclusion := range exclude {
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "reason"}, textNode(exclusion.Reason)}}
		if exclusion.ReplacedBy != "" {
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "replacedBy"}, textNode(exclusion.ReplacedBy))
		}
		if exclusion.BasedOn != nil {
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "basedOn"}, versionNode(*exclusion.BasedOn))
		}
		nodes[id] = node
	}
	return nodes
}

// versionNode encodes a rule version double-quoted, such as "1.3.0", so YAML reads it as text.
func versionNode(version RuleVersion) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: version.String(), Style: yaml.DoubleQuotedStyle}
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
