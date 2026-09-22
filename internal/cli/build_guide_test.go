// Reproduce upgrades where generated rules are current but the bundled project guide has changed.

package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestBuildThenCheckAfterGuideUpgrade requires a successful rebuild to leave the managed guide current too.
func TestBuildThenCheckAfterGuideUpgrade(t *testing.T) {
	binary := buildCLI(t)
	for _, command := range []string{"build", "sync"} {
		for _, config := range []string{".code-rules/config.json", "custom.json"} {
			for _, state := range []string{"old", "missing"} {
				t.Run(command+"/"+config+"/"+state, func(t *testing.T) {
					directory := t.TempDir()
					if _, diagnostic, code := runCLI(t, binary, directory, "project", "init", "--config", config); code != 0 {
						t.Fatal(code, diagnostic)
					}
					guideName := "README.md"
					if config == "custom.json" {
						guideName = "CODE_RULES.md"
					}
					guidePath := filepath.Join(directory, filepath.Dir(config), guideName)
					// Simulate an untouched guide from an earlier release using its ownership marker.
					body := []byte("# Code Rules\n\nProject guide from an earlier release.\n")
					guide := []byte(fmt.Sprintf("<!-- code-rules:project-guide sha256:%x -->\n%s", sha256.Sum256(body), body))
					if err := os.WriteFile(guidePath, guide, 0600); err != nil {
						t.Fatal(err)
					}
					if state == "missing" {
						if err := os.Remove(guidePath); err != nil {
							t.Fatal(err)
						}
					}
					for i, action := range []string{command, command, "check"} {
						out, diagnostic, code := runCLI(t, binary, directory, "project", action, "--config", config)
						if code != 0 {
							t.Fatalf("project %s after guide upgrade: exit %d\n%s%s", action, code, out, diagnostic)
						}
						if i == 0 && !strings.Contains(out, "Code Rules guide ") {
							t.Fatal("guide update not reported", out)
						}
						if i == 1 && strings.Contains(out, "Code Rules guide ") {
							t.Fatal("guide updated twice", out)
						}
						if action == "check" && !strings.Contains(out, "Status: up to date.") {
							t.Fatal(out)
						}
					}
				})
			}
		}
	}
}

// TestBuildPreservesEditedGuide refuses unsafe guide upgrades before publishing generated output.
func TestBuildPreservesEditedGuide(t *testing.T) {
	binary := buildCLI(t)
	for _, command := range []string{"build", "sync"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			for _, action := range []string{"init", "build"} {
				if out, diagnostic, code := runCLI(t, binary, dir, "project", action); code != 0 {
					t.Fatal(code, out, diagnostic)
				}
			}
			guide := filepath.Join(dir, ".code-rules/README.md")
			file, err := os.OpenFile(guide, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = file.WriteString("\nMy project notes.\n"); err != nil {
				t.Fatal(err)
			}
			if err = file.Close(); err != nil {
				t.Fatal(err)
			}
			before := projectFileContents(t, dir)
			out, diagnostic, code := runCLI(t, binary, dir, "project", command)
			if code != 1 || out != "" || !strings.Contains(diagnostic, "manually edited project guide") {
				t.Fatal(code, out, diagnostic)
			}
			if !reflect.DeepEqual(before, projectFileContents(t, dir)) {
				t.Fatal("refusal changed project files")
			}
		})
	}
}

// TestBuildGuideJSON reports guide changes without mixing them into generated-relative paths.
func TestBuildGuideJSON(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	if _, diagnostic, code := runCLI(t, binary, dir, "project", "init"); code != 0 {
		t.Fatal(diagnostic)
	}
	if err := os.Remove(filepath.Join(dir, ".code-rules/README.md")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		out, diagnostic, code := runCLI(t, binary, dir, "project", "build", "--json")
		var result struct {
			OK    bool
			Value struct {
				Guide *struct {
					Path    string
					Created bool
				}
				Added []string
			}
		}
		if code != 0 || diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil || !result.OK {
			t.Fatal(code, out, diagnostic)
		}
		if i == 0 {
			if result.Value.Guide == nil || result.Value.Guide.Path != "README.md" || !result.Value.Guide.Created {
				t.Fatal(out)
			}
		} else if result.Value.Guide != nil {
			t.Fatal("unchanged guide reported", out)
		}
		for _, name := range result.Value.Added {
			if name == "README.md" {
				t.Fatal("mixed path bases", out)
			}
		}
	}
}
