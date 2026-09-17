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
	status, err := CheckProjectGuide(ctx, options)
	if err == nil || status.Current {
		t.Fatal(status, err)
	}
	unchanged, _ := os.ReadFile(filepath.Join(directory, "CODE_RULES.md"))
	if !bytes.Equal(unchanged, old) {
		t.Fatal("check wrote the guide")
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
	if status, err := CheckProjectGuide(ctx, options); err != nil || !status.Current {
		t.Fatal(status, err)
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

// TestGuideCheckDoesNotInitialize confirms the read-only check leaves a missing project absent.
func TestGuideCheckDoesNotInitialize(t *testing.T) {
	parent := t.TempDir()
	if _, err := CheckProjectGuide(context.Background(), Options{ConfigPath: filepath.Join(parent, ".code-rules/config.json")}); err == nil {
		t.Fatal("expected missing project error")
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
}

// TestGuideCheckRejectsChangedSnapshot verifies a final comparison cannot approve edits or an active writer.
func TestGuideCheckRejectsChangedSnapshot(t *testing.T) {
	for _, change := range []string{"config", "guide", "writer"} {
		t.Run(change, func(t *testing.T) {
			directory := t.TempDir()
			options := Options{ConfigPath: filepath.Join(directory, "config.json")}
			if _, err := InitializeProject(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			config, err := os.ReadFile(options.ConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			guideName, guide := ProjectGuide(options.ConfigPath)
			if change == "writer" {
				err = root.Mkdir(".code-rules-lock", 0700)
			} else if change == "config" {
				err = os.WriteFile(options.ConfigPath, append(config, '\n'), 0600)
			} else {
				err = os.WriteFile(filepath.Join(directory, guideName), []byte("Edited guide."), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			err = requireGuideUnchanged(context.Background(), root, "config.json", guideName, config, guide)
			var problem *project.Error
			want := "concurrent-change"
			if change == "writer" {
				want = "busy"
			}
			if !errors.As(err, &problem) || problem.Code != want {
				t.Fatalf("expected %s, got %v", want, err)
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
