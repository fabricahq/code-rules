// Maintain the agent-facing project guide separately from user-owned rules and configuration.

package project

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
)

// projectGuideTemplate ships with the binary; its shell examples are exercised by CLI integration tests.
//
//go:embed project-guide.md
var projectGuideTemplate string

// renderProjectGuide anchors commands to the guide directory and quotes custom config names for POSIX shells.
func renderProjectGuide(configName string) []byte {
	argument := "'./" + strings.ReplaceAll(configName, "'", "'\"'\"'") + "'"
	content := strings.NewReplacer(
		"{{CONFIG_ARG}}", argument,
		"{{CONFIG_NAME}}", strings.ReplaceAll(configName, "`", "&#96;"),
	).Replace(projectGuideTemplate)
	return stampProjectGuide([]byte(content))
}

// projectGuide returns the managed guide's relative path and bytes for the selected configuration.
// A conventional rules directory owns its README; other directories retain their general project README.
func projectGuide(configPath string) (name string, content []byte) {
	if configPath == "" {
		configPath = filepath.Join(".code-rules", "config.json")
	}
	name = "CODE_RULES.md"
	if filepath.Base(filepath.Dir(configPath)) == ".code-rules" {
		name = "README.md"
	}
	return name, renderProjectGuide(filepath.Base(configPath))
}

// stampProjectGuide records the generated body's digest so later init runs can distinguish edits from older templates.
func stampProjectGuide(body []byte) []byte {
	return append([]byte(fmt.Sprintf("<!-- code-rules:project-guide sha256:%x -->\n", sha256.Sum256(body))), body...)
}

// unmodifiedProjectGuide accepts only our exact ownership header and an unchanged body; it is not an authenticity check.
func unmodifiedProjectGuide(data []byte) bool {
	_, body, found := bytes.Cut(data, []byte("\n"))
	return found && bytes.Equal(data, stampProjectGuide(body))
}

// prepareProjectGuide returns a safe creation or refresh, refusing to overwrite manually authored or edited bytes.
func prepareProjectGuide(ctx context.Context, root *os.Root, configName string) (*filetxn.File, error) {
	name, wanted := projectGuide(filepath.Join(root.Name(), configName))
	if strings.EqualFold(configName, name) {
		return nil, failure("path-collision", "configuration and project guide must use different filenames", nil)
	}
	current, err := filetxn.ReadOptional(ctx, root, name)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(current, wanted) {
		return nil, nil
	}
	if current != nil && !unmodifiedProjectGuide(current) {
		return nil, failure("guide-edited", name+": unrecognized or manually edited project guide; preserve your notes in a separate file, move this guide aside, then run code-rules project init again", nil)
	}
	return &filetxn.File{Path: name, Content: wanted, Previous: current}, nil
}

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

Complete drafts before building. Run code-rules project build after local edits, or code-rules project sync after changing a library source. Read the resulting [resolved rules](../generated/RULES.md) when working on the project.
`

// renderLocalReadme links newly initialized local guidance to the configuration's managed guide.
func renderLocalReadme(configPath string) []byte {
	name, _ := projectGuide(configPath)
	return []byte(strings.ReplaceAll(localReadme, "{{PROJECT_GUIDE}}", name))
}
