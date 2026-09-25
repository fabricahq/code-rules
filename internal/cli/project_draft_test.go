// Exercise draft rejection and completed local rules through the native CLI.

package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestProjectDraftBlocksConsumption keeps unfinished local rules out of every generated path.
func TestProjectDraftBlocksConsumption(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	commands := [][]string{
		{"project", "init"},
		{"project", "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When writing Go."},
		{"project", "build"},
		{"project", "add", "rule", "techs/go/draft", "--title", "Return errors", "--when-to-read", "When writing Go.", "--impact", "HIGH", "--impact-description", "Preserves failures."},
	}
	for _, args := range commands {
		out, diagnostic, code := runCLI(t, binary, directory, args...)
		if code != 0 || diagnostic != "" {
			t.Fatal(args, code, out, diagnostic)
		}
	}
	draftPath := filepath.Join(directory, ".code-rules/local/techs/go/draft.md")
	draft, err := os.ReadFile(draftPath)
	if err != nil || !strings.Contains(string(draft), "<!-- code-rules:draft -->") {
		t.Fatal("unmarked draft", err, string(draft))
	}
	before := projectFileContents(t, directory)
	for _, action := range []string{"build", "sync", "check"} {
		out, diagnostic, code := runCLI(t, binary, directory, "project", action)
		if code != 1 || out != "" || !strings.Contains(diagnostic, "local/techs/go/draft.md") || !strings.Contains(diagnostic, "remove its code-rules:draft marker") {
			t.Fatal(action, code, out, diagnostic)
		}
		if after := projectFileContents(t, directory); !reflect.DeepEqual(before, after) {
			t.Fatal(action, "changed project files on draft failure")
		}
	}
	complete := "---\ntitle: Return errors\nwhenToRead: When writing Go.\nimpact: HIGH\nimpactDescription: Preserves failures.\n---\n\n## Return errors\n\nReturn errors to the caller when an operation fails.\n"
	if err := os.WriteFile(draftPath, []byte(complete), 0600); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"build", "check", "sync"} {
		out, diagnostic, code := runCLI(t, binary, directory, "project", action)
		if code != 0 || diagnostic != "" {
			t.Fatal(action, code, out, diagnostic)
		}
	}
}
