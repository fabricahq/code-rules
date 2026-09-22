// Create project operation fixtures and verify categorized errors.

package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fabricahq/code-rules/internal/filetxn"
)

// openProject creates a disposable project root owned by the test.
func openTestProject(t *testing.T) *os.Root {
	t.Helper()
	directory := filepath.Join(t.TempDir(), ".code-rules")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root
}

// writeFixture writes an authored test file and creates its parent directories.
func writeFixture(t *testing.T, root *os.Root, name, text string) {
	t.Helper()
	if err := root.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile(name, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

// projectCode checks a stable caller-visible category without matching incidental prose.
func projectCode(t *testing.T, err error, want string) {
	t.Helper()
	var failure *filetxn.Error
	if !errors.As(err, &failure) || failure.Code != want {
		t.Fatalf("wanted %s, got %v", want, err)
	}
}
