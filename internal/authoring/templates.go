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

A **local group** belongs to this project. Its metadata and rules live here, and you edit them directly. It needs no library or source declaration.

A **library group** comes from a versioned Git repository. The project selects it in its configuration and imports it with sync. Change its source selection or upstream library, then sync again; keep project-specific changes local.

## When groups share an ID

Local and library definitions can contribute to the same group, such as techs/go:

- Their rules are combined. Adding a local rule does not replace an imported rule, even when their filenames match.
- Local _group.json metadata takes precedence over library metadata for the group's name, description, and reading cue.
- Use explicit exclusions or replacements in the project configuration to remove or override imported rules.

## Manage local guidance

Follow [the project guide](../{{PROJECT_GUIDE}}) to add groups and rules. Keep rules within their group's scope and follow the [rule authoring rubric](https://github.com/fabricahq/code-rules/blob/main/docs/src/content/docs/reference/rule-authoring.md).

Complete drafts before building. Run build after local edits, or sync after changing a library source. Read the resulting [resolved rules](../generated/RULES.md) when working on the project.
`

// renderLocalReadme links newly initialized local guidance to the configuration's managed guide.
func renderLocalReadme(configPath string) []byte {
	name, _ := ProjectGuide(configPath)
	return []byte(strings.ReplaceAll(localReadme, "{{PROJECT_GUIDE}}", name))
}

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
