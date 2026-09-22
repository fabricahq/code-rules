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
		{"project", "init", "--non-interactive"},
		{"project", "init"},
		{"project", "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."},
		{"project", "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md"},
		{"project", "build"}, {"project", "check"},
		{"project", "add", "library", "team", "--repository", "https://github.com/acme/rules", "--ref", ">= 1.2.3, < 2.0.0", "--groups", "techs/go"},
	} {
		if err := os.WriteFile(filepath.Join(directory, "body.md"), []byte("# Return errors\n\nReturn failures to callers.\n"), 0600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runCLI(t, binary, directory, args...)
		if code != 0 || stderr != "" || stdout == "" {
			t.Fatalf("%v: exit%d stdout=%s stderr=%s", args, code, stdout, stderr)
		}
	}
	for _, args := range [][]string{{"project", "add", "group", "techs/rust"}, {"project", "add", "library", "missing", "--repository", "https://github.com/acme/other"}} {
		stdout, stderr, code := runCLI(t, binary, directory, args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "usage") {
			t.Fatal(args, code, stdout, stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(directory, ".code-rules/vendor")); !os.IsNotExist(err) {
		t.Fatal("authoring unexpectedly fetched source", err)
	}
}
