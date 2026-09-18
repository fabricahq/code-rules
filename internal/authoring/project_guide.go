// Maintain the agent-facing project guide separately from user-owned rules and configuration.

package authoring

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// ProjectGuide returns the managed guide's relative path and bytes for the selected configuration.
// A conventional rules directory owns its README; other directories retain their general project README.
func ProjectGuide(configPath string) (name string, content []byte) {
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
func prepareProjectGuide(ctx context.Context, root *os.Root, configName string) (*authoredFile, error) {
	name, wanted := ProjectGuide(filepath.Join(root.Name(), configName))
	if strings.EqualFold(configName, name) {
		return nil, failure("path-collision", "configuration and project guide must use different filenames", nil)
	}
	current, err := optionalFile(ctx, root, name)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(current, wanted) {
		return nil, nil
	}
	if current != nil && !unmodifiedProjectGuide(current) {
		return nil, failure("guide-edited", name+": unrecognized or manually edited project guide; preserve your notes in a separate file, move this guide aside, then run init again", nil)
	}
	return &authoredFile{name: name, data: wanted, before: current}, nil
}
