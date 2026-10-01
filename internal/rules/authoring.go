// Render validated authoring documents with a canonical unfinished template and no invented policy.

package rules

import (
	"bytes"
	_ "embed"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/fabricahq/code-rules/libraryformat"
)

// RuleMetadata is the complete author-supplied discovery and consequence metadata.
type RuleMetadata struct {
	Title             string `json:"title" yaml:"title"`
	WhenToRead        string `json:"whenToRead" yaml:"whenToRead"`
	Impact            string `json:"impact" yaml:"impact"`
	ImpactDescription string `json:"impactDescription" yaml:"impactDescription"`
}

// ruleTemplate is compiled into the native executable, so drafting needs no source checkout.
//
//go:embed rule-template.md
var ruleTemplate string

// RenderGroup returns the _group.yaml text of metadata with its text trimmed, failing unless libraryformat reads it.
func RenderGroup(metadata libraryformat.GroupMetadata) ([]byte, error) {
	draft, err := encodeGroup(metadata)
	if err != nil {
		return nil, err
	}
	normalized, err := libraryformat.ParseGroupMetadata(draft, "_group.yaml")
	if err != nil {
		return nil, err
	}
	return encodeGroup(normalized)
}

// encodeGroup writes metadata as YAML with two-space indentation.
func encodeGroup(metadata libraryformat.GroupMetadata) ([]byte, error) {
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(metadata); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// RenderRule validates YAML-safe metadata and either the supplied body or an explicitly unfinished canonical draft.
func RenderRule(id string, metadata RuleMetadata, body *string) ([]byte, error) {
	header, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	content := ""
	if body != nil {
		content = *body
	} else {
		_, content, _ = strings.Cut(ruleTemplate[4:], "\n---\n")
		content = strings.TrimLeft(content, "\n")
		content = strings.Replace(content, "## <Short action-oriented title>", "## "+strings.NewReplacer("\r", " ", "\n", " ").Replace(metadata.Title), 1)
	}
	text := "---\n" + string(header) + "---\n\n" + content
	if _, err := libraryformat.ParseRule(text, id+".md", "local"); err != nil {
		return nil, err
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if body == nil {
		text += "\n" + DraftMarker + "\n"
	}
	return []byte(text), nil
}
