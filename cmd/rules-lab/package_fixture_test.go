// Check that lab copying cannot make an interrupted source artifact collection appear complete.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackageLabRejectsIncompleteSource checks the actual Install scenario against a source collection marker.
func TestPackageLabRejectsIncompleteSource(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "INCOMPLETE"), []byte("building"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := packageFromArtifacts(json.RawMessage(`{"scenario":"install"}`), directory)
	if err != nil || result.OK || result.Error == nil || !strings.Contains(result.Error.Message, "incomplete") {
		t.Fatal(result, err)
	}
}
