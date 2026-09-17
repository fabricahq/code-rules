// Verify guide upgrades preserve authored files and refuse ambiguous ownership.

package authoring

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
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
	for name, data := range map[string][]byte{"README.md": old, "config.json": config, "local/README.md": local} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	status, err := CheckProjectGuide(ctx, options)
	if err == nil || status.Current {
		t.Fatal(status, err)
	}
	unchanged, _ := os.ReadFile(filepath.Join(directory, "README.md"))
	if !bytes.Equal(unchanged, old) {
		t.Fatal("check wrote the guide")
	}
	result, err := InitializeProject(ctx, options)
	if err != nil || len(result.Files) != 1 || result.Files[0] != filepath.Join(directory, "README.md") {
		t.Fatal(result, err)
	}
	for name, want := range map[string][]byte{"README.md": renderProjectGuide("config.json"), "config.json": config, "local/README.md": local} {
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
			guide := filepath.Join(directory, "README.md")
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
				t.Fatal("overwrote README", err)
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
