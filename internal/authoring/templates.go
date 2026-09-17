// Render validated authoring documents with a canonical unfinished template and no invented policy.

package authoring

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
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

const localReadme = `# Local rules

Start with [the project guide](../README.md) for Code Rules commands and agent workflows.

Put project rules under techs/<group>/ or practices/<group>/.
Describe each local group in _group.json with its name, description, and whenToRead guidance.
Local rules join your project's rule set automatically. Imported rules in the same group join them; local group metadata supplies the project's description.
Use explicit exclusions or replacements in config.json to override imported rules.

Follow the [canonical rule rubric and template](https://github.com/fabricahq/code-rules/blob/main/docs/src/content/docs/reference/rule-authoring.md).
Complete drafts before building. Run build after local edits, or sync after adding or updating a library source.
`

// jsonText serializes authored JSON as readable UTF-8 with one final newline.
func jsonText(value any) ([]byte, error) {
	var out bytes.Buffer
	e := json.NewEncoder(&out)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	err := e.Encode(value)
	return out.Bytes(), err
}

// renderGroup validates and trims group metadata before serialization.
func renderGroup(metadata rules.GroupMetadata) ([]byte, error) {
	data, err := jsonText(metadata)
	if err != nil {
		return nil, err
	}
	normalized, err := rules.ParseGroupMetadata(data, "_group.json")
	if err != nil {
		return nil, err
	}
	return jsonText(normalized)
}

// renderRule validates YAML-safe metadata and either the supplied body or an explicitly unfinished canonical draft.
func renderRule(id string, metadata RuleMetadata, body *string) ([]byte, error) {
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
	if _, err := rules.Parse(text, id+".md", "local"); err != nil {
		return nil, err
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return []byte(text), nil
}
