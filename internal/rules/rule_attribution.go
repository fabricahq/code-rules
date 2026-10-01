// Add an attribution entry to a rule document's frontmatter, keeping the rest of the document.

package rules

import (
	"reflect"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/authored"
)

// AddRuleAttribution returns document, a rule at path, with entry after its existing attribution entries. When the
// frontmatter is a block mapping without an attribution field, the field is appended and every existing byte is
// kept, unless appending would change another value, such as a final block scalar that keeps its line breaks.
// Otherwise the frontmatter is written again, keeping comments and values but not their formatting, with folded
// scalars in literal style. The body never changes. It fails when document isn't a valid rule or the result's
// attribution isn't valid, such as for a URL that isn't an absolute HTTP(S) URL.
func AddRuleAttribution(document, path string, entry coderules.Attribution) (string, error) {
	original, err := coderules.ParseRule(document, path, "local")
	if err != nil {
		return "", err
	}
	sections, err := coderules.SplitDocument(document, path)
	if err != nil {
		return "", err
	}
	// The frontmatter starts right after the opening --- line.
	start := strings.IndexByte(document, '\n') + 1
	end := start + len(sections.Frontmatter)
	newline := "\n"
	if strings.HasPrefix(document[end:], "\r\n") {
		newline = "\r\n"
	}
	var metadata yaml.Node
	if err := yaml.NewDecoder(strings.NewReader(authored.DecodeSurrogatePairs(document[start:end]))).Decode(&metadata); err != nil {
		return "", authored.Invalid(path, "invalid YAML: "+err.Error())
	}
	item := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "url"}, textNode(entry.URL),
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "description"}, textNode(entry.Description),
	}}
	mapping := metadata.Content[0]
	if mappingValue(mapping, "attribution") == nil && mapping.Style&yaml.FlowStyle == 0 {
		field, err := encodeFrontmatter(&yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "attribution"},
			{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{item}},
		}})
		if err != nil {
			return "", err
		}
		result := document[:end] + newline + strings.ReplaceAll(field, "\n", newline) + document[end:]
		if err := requireAttributionAdded(original, result, path, entry); err == nil {
			return result, nil
		}
	}
	list := mappingValue(mapping, "attribution")
	if list == nil {
		list = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "attribution"}, list)
	}
	list.Content = append(list.Content, item)
	normalizeFoldedScalars(&metadata)
	frontmatter, err := encodeFrontmatter(&metadata)
	if err != nil {
		return "", err
	}
	result := document[:start] + strings.ReplaceAll(frontmatter, "\n", newline) + document[end:]
	if err := requireAttributionAdded(original, result, path, entry); err != nil {
		return "", err
	}
	return result, nil
}

// requireAttributionAdded fails unless result is original's rule with only entry added after its attribution.
func requireAttributionAdded(original coderules.Rule, result, path string, entry coderules.Attribution) error {
	parsed, err := coderules.ParseRule(result, path, "local")
	if err != nil {
		return err
	}
	added := len(parsed.Attribution) - 1
	if added != len(original.Attribution) || parsed.Attribution[added].Description != entry.Description {
		return authored.Invalid(path, "could not add the attribution entry; add it by hand")
	}
	want := original
	want.Attribution = append(slices.Clone(original.Attribution), parsed.Attribution[added])
	want.Document = result
	if !reflect.DeepEqual(want, parsed) {
		return authored.Invalid(path, "adding an attribution entry would change the rule's other metadata; add it by hand")
	}
	return nil
}

// encodeFrontmatter writes node as YAML with two-space indentation and no line wrapping, without a final line break.
func encodeFrontmatter(node *yaml.Node) (string, error) {
	data, err := yaml.Dump(node, yaml.WithV3Defaults(), yaml.WithIndent(2), yaml.WithLineWidth(-1))
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}
