// Verify project authoring through the installed command shape with no JavaScript runtime or Git.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAuthoringCLIProcess creates, extends, builds, and checks a local-only project through the real binary.
func TestAuthoringCLIProcess(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	for _, args := range [][]string{
		{"init", "--non-interactive"},
		{"init"},
		{"local", "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."},
		{"local", "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md"},
		{"build"}, {"check"},
		{"add", "source", "team", "--repository", "https://github.com/acme/rules", "--version", ">= 1.2.3, < 2.0.0", "--groups", "techs/go"},
	} {
		if err := os.WriteFile(filepath.Join(directory, "body.md"), []byte("# Return errors\n\nReturn failures to callers.\n"), 0600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runCLI(t, binary, directory, args...)
		if code != 0 || stderr != "" || stdout == "" {
			t.Fatalf("%v: exit%d stdout=%s stderr=%s", args, code, stdout, stderr)
		}
	}
	for _, args := range [][]string{{"local", "add", "group", "techs/rust"}, {"add", "source", "missing", "--repository", "https://github.com/acme/other"}} {
		stdout, stderr, code := runCLI(t, binary, directory, args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "usage") {
			t.Fatal(args, code, stdout, stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(directory, ".code-rules/vendor")); !os.IsNotExist(err) {
		t.Fatal("authoring unexpectedly fetched source", err)
	}
}
