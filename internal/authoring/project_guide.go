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

	"github.com/fabricahq/code-rules/internal/project"
)

// projectGuideTemplate ships with the binary; its shell examples are exercised by CLI integration tests.
//
//go:embed project-guide.md
var projectGuideTemplate string

// GuideStatus identifies the checked guide and whether its exact bytes match this executable's template.
type GuideStatus struct {
	Path    string `json:"path"`
	Current bool   `json:"current"`
}

// renderProjectGuide anchors commands to the README directory and quotes custom config names for POSIX shells.
func renderProjectGuide(configName string) []byte {
	argument := "'./" + strings.ReplaceAll(configName, "'", "'\"'\"'") + "'"
	content := strings.ReplaceAll(projectGuideTemplate, "{{CONFIG_ARG}}", argument)
	content = strings.ReplaceAll(content, "{{CONFIG_NAME}}", strings.ReplaceAll(configName, "`", "&#96;"))
	return stampProjectGuide([]byte(content))
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
	if strings.EqualFold(configName, "README.md") {
		return nil, failure("path-collision", "configuration and project guide must use different filenames", nil)
	}
	current, err := optionalFile(ctx, root, "README.md")
	if err != nil {
		return nil, err
	}
	wanted := renderProjectGuide(configName)
	if bytes.Equal(current, wanted) {
		return nil, nil
	}
	if current != nil && !unmodifiedProjectGuide(current) {
		return nil, failure("guide-edited", "README.md: unrecognized or manually edited project guide; preserve your notes in a separate file, move this README aside, then run init again", nil)
	}
	return &authoredFile{name: "README.md", data: wanted, before: current}, nil
}

// CheckProjectGuide compares the saved README with the installed template without creating files or acquiring a writer lock.
func CheckProjectGuide(ctx context.Context, options Options) (GuideStatus, error) {
	root, name, err := openProject(ctx, options, false)
	if err != nil {
		return GuideStatus{}, err
	}
	defer root.Close()
	status := GuideStatus{Path: filepath.Join(root.Name(), "README.md")}
	if err = project.RequireIdle(root); err != nil {
		return status, err
	}
	if _, _, err = configuration(ctx, root, name); err != nil {
		return status, err
	}
	current, err := optionalFile(ctx, root, "README.md")
	if err != nil {
		return status, err
	}
	status.Current = bytes.Equal(current, renderProjectGuide(name))
	if !status.Current {
		return status, failure("guide-outdated", "README.md: missing or out of date for this CLI; run code-rules init with the same --config to refresh the project guide", nil)
	}
	return status, nil
}
