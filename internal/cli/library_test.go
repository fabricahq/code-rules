// Exercise library commands through the compiled native CLI with no runtime executables on PATH.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLibraryCLIProcess checks terms preservation, complete authoring, and actionable unfinished-draft failure.
func TestLibraryCLIProcess(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "terms.txt"), []byte("Terms\r\n"), 0600)
	os.WriteFile(filepath.Join(dir, "body.md"), []byte("Return failures."), 0600)
	commands := [][]string{
		{"library", "init", "--directory", "library", "--spdx", "MIT", "--license-file", "terms.txt"},
		{"library", "add", "group", "techs/go", "--directory", "library", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."},
		{"library", "add", "rule", "techs/go/errors", "--directory", "library", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md"},
		{"library", "check", "--directory", "library"},
	}
	for _, args := range commands {
		out, diagnostic, code := runCLI(t, binary, dir, args...)
		if code != 0 || out == "" || diagnostic != "" {
			t.Fatal(args, code, out, diagnostic)
		}
	}
	data, _ := os.ReadFile(filepath.Join(dir, "library/LICENSE.md"))
	if string(data) != "Terms\r\n" {
		t.Fatal("terms changed")
	}
	draft := append([]string{}, commands[2][:len(commands[2])-2]...)
	draft[3] = "techs/go/draft"
	out, diagnostic, code := runCLI(t, binary, dir, draft...)
	if code != 0 {
		t.Fatal(draft, code, out, diagnostic)
	}
	out, diagnostic, code = runCLI(t, binary, dir, "library", "check", "--directory", "library")
	if code != 1 || out != "" || !strings.Contains(diagnostic, "code-rules:draft") {
		t.Fatal(code, out, diagnostic)
	}
}
