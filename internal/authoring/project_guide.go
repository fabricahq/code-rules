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

// CheckProjectGuide compares the saved guide with the installed template without creating files or acquiring a writer lock.
func CheckProjectGuide(ctx context.Context, options Options) (GuideStatus, error) {
	root, name, err := openProject(ctx, options, false)
	if err != nil {
		return GuideStatus{}, err
	}
	defer root.Close()
	guideName, wanted := ProjectGuide(filepath.Join(root.Name(), name))
	status := GuideStatus{Path: filepath.Join(root.Name(), guideName)}
	if err = project.RequireIdle(root); err != nil {
		return status, err
	}
	config, _, err := configuration(ctx, root, name)
	if err != nil {
		return status, err
	}
	current, err := optionalFile(ctx, root, guideName)
	if err != nil {
		return status, err
	}
	if err := requireGuideUnchanged(ctx, root, name, guideName, config, current); err != nil {
		return status, err
	}
	status.Current = bytes.Equal(current, wanted)
	if !status.Current {
		return status, failure("guide-outdated", guideName+": missing or out of date for this CLI; run code-rules init with the same --config to refresh the project guide", nil)
	}
	return status, nil
}

// requireGuideUnchanged rejects overlapping edits and active writers before reporting guide freshness.
func requireGuideUnchanged(ctx context.Context, root *os.Root, configName, guideName string, config, guide []byte) error {
	currentConfig, err := optionalFile(ctx, root, configName)
	if err != nil {
		return err
	}
	currentGuide, err := optionalFile(ctx, root, guideName)
	if err != nil {
		return err
	}
	if !bytes.Equal(config, currentConfig) || !bytes.Equal(guide, currentGuide) {
		return failure("concurrent-change", "configuration or project guide changed during the check; retry after edits finish", nil)
	}
	return project.RequireIdle(root)
}
