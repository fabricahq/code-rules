// Exercise group documentation through the native authoring and validation commands.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGroupGuides verifies explicit group creation produces documentation that does not become a rule.
func TestGroupGuides(t *testing.T) {
	binary := buildCLI(t)
	for _, library := range []bool{false, true} {
		name := "local"
		if library {
			name = "library"
		}
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			prefix := "local"
			groupDir := filepath.Join(directory, ".code-rules/local/techs/go")
			init := []string{"init"}
			if library {
				prefix = "library"
				groupDir = filepath.Join(directory, "techs/go")
				init = []string{"library", "init"}
			}
			// run requires the actual executable to complete each authoring or verification step.
			run := func(args ...string) string {
				t.Helper()
				out, diagnostic, code := runCLI(t, binary, directory, args...)
				if code != 0 {
					t.Fatalf("%v: exit %d\n%s\n%s", args, code, out, diagnostic)
				}
				return out
			}
			run(init...)
			run(prefix, "add", "group", "techs/go", "--name", "Go", "--description", "Go conventions.", "--when-to-read", "When editing Go.")
			if err := os.WriteFile(filepath.Join(directory, "body.md"), []byte("Return errors to the caller.\n"), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{prefix, "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md"}

			run(args...)
			guide, err := os.ReadFile(filepath.Join(groupDir, "README.md"))
			if err != nil {
				t.Fatal(err)
			}
			for _, text := range []string{"techs/go", "_group.json", "Instructions for agents", "code-rules " + prefix + " add rule"} {
				if !strings.Contains(string(guide), text) {
					t.Fatalf("guide missing %q", text)
				}
			}
			if library {
				out := run("library", "check", "--json")
				if !strings.Contains(out, `"rules": 1`) {
					t.Fatal(out)
				}
			} else {
				run("build")
				run("check")
			}
			out, diagnostic, code := runCLI(t, binary, directory, prefix, "add", "group", "techs/go", "--name", "Changed", "--description", "Changed", "--when-to-read", "Changed")
			if code != 1 {
				t.Fatal(code, out, diagnostic)
			}
			after, err := os.ReadFile(filepath.Join(groupDir, "README.md"))
			if err != nil || string(after) != string(guide) {
				t.Fatal("overwrote group guide", err)
			}
		})
	}
}
