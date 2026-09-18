// Verify guide upgrades preserve authored files and refuse ambiguous ownership.

package authoring

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fabricahq/code-rules/internal/project"
)

// TestProjectGuideRefresh updates an older generated guide while preserving configuration and local guidance byte for byte.
func TestProjectGuideRefresh(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	options := Options{ConfigPath: filepath.Join(directory, "config.json")}
	if _, err := InitializeProject(ctx, options); err != nil {
		t.Fatal(err)
	}
	old := stampProjectGuide([]byte("# Earlier release guide\n"))
	config := []byte("{\n  \"schemaVersion\": 1, \"sources\": {}\n}\n")
	local := []byte("My local authoring notes.\n")
	for name, data := range map[string][]byte{"CODE_RULES.md": old, "config.json": config, "local/README.md": local} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	result, err := InitializeProject(ctx, options)
	if err != nil || len(result.Files) != 1 || result.Files[0] != filepath.Join(directory, "CODE_RULES.md") {
		t.Fatal(result, err)
	}
	for name, want := range map[string][]byte{"CODE_RULES.md": renderProjectGuide("config.json"), "config.json": config, "local/README.md": local} {
		got, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal(name, err)
		}
	}
	if result, err := InitializeProject(ctx, options); err != nil || len(result.Files) != 0 {
		t.Fatal(result, err)
	}
}

// TestProjectGuideRefusesUnmanagedFilesAndLinks verifies collisions fail before scaffolding is published.
func TestProjectGuideRefusesUnmanagedFilesAndLinks(t *testing.T) {
	for _, kind := range []string{"authored", "edited", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			directory := t.TempDir()
			guide := filepath.Join(directory, "CODE_RULES.md")
			content := []byte("# My project notes\n")
			if kind == "edited" {
				content = append(stampProjectGuide([]byte("Old guide\n")), []byte("Edit\n")...)
			}
			if kind == "symlink" {
				target := filepath.Join(t.TempDir(), "notes.md")
				if err := os.WriteFile(target, content, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, guide); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(guide, content, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := InitializeProject(context.Background(), Options{ConfigPath: filepath.Join(directory, "config.json")}); err == nil {
				t.Fatal("expected refusal")
			}
			got, err := os.ReadFile(guide)
			if err != nil || !bytes.Equal(got, content) {
				t.Fatal("overwrote guide", err)
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 {
				t.Fatal("published partial scaffolding", entries, err)
			}
		})
	}
}

// TestProjectGuideConfigCollision rejects a configuration that would share its managed guide's path.
func TestProjectGuideConfigCollision(t *testing.T) {
	for _, config := range []string{"CODE_RULES.md", ".code-rules/README.md"} {
		t.Run(config, func(t *testing.T) {
			directory := t.TempDir()
			_, err := InitializeProject(context.Background(), Options{ConfigPath: filepath.Join(directory, config)})
			var problem *project.Error
			if !errors.As(err, &problem) || problem.Code != "path-collision" {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(directory, config)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("published colliding file", err)
			}
		})
	}
}

// TestProjectGuideDefault agrees with the project operations' omitted configuration path.
func TestProjectGuideDefault(t *testing.T) {
	name, content := ProjectGuide("")
	if name != "README.md" || !bytes.Equal(content, renderProjectGuide("config.json")) {
		t.Fatal("default guide must match .code-rules/config.json", name)
	}
}
