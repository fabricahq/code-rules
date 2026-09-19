// Render validated authoring documents with a canonical unfinished template and no invented policy.

package rules

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"

	"go.yaml.in/yaml/v4"
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

// encodeAuthoredJSON serializes authored JSON as readable UTF-8 with one final newline.
func encodeAuthoredJSON(value any) ([]byte, error) {
	var out bytes.Buffer
	e := json.NewEncoder(&out)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	err := e.Encode(value)
	return out.Bytes(), err
}

// RenderGroup validates and trims group metadata before serialization.
func RenderGroup(metadata GroupMetadata) ([]byte, error) {
	data, err := encodeAuthoredJSON(metadata)
	if err != nil {
		return nil, err
	}
	normalized, err := ParseGroupMetadata(data, "_group.json")
	if err != nil {
		return nil, err
	}
	return encodeAuthoredJSON(normalized)
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
		content = strings.Replace(content, "## <Short action-oriented title>", "## "+strings.NewReplacer("\r", " ", "\n", " ").Replace(metadata.Title), 1)
	}
	text := "---\n" + string(header) + "---\n\n" + content
	if _, err := Parse(text, id+".md", "local"); err != nil {
		return nil, err
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return []byte(text), nil
}
