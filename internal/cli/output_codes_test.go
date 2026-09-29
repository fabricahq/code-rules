// Check that failures from Git and imports keep their stable codes in JSON output.

package cli

import (
	"encoding/json"
	"testing"
)

// TestSyncJSON_ReportsGitFailureCode lets scripts branch on why an import failed without parsing its message.
func TestSyncJSON_ReportsGitFailureCode(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	for _, args := range [][]string{
		{"project", "init"},
		{"project", "add", "library", "team", "--repository", "https://example.invalid/team.git", "--ref", "v1.2.3", "--groups", "*", "--non-interactive"},
	} {
		if out, diagnostic, code := runCLI(t, binary, directory, args...); code != 0 {
			t.Fatal(args, code, out, diagnostic)
		}
	}
	// runCLI puts no executables on PATH, so the import fails before contacting the repository.
	out, _, code := runCLI(t, binary, directory, "project", "sync", "--json")
	var result response
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(out, err)
	}
	if code != 1 || result.Error == nil || result.Error.Code != "git-unavailable" || result.Error.Kind != "operation" {
		t.Fatalf("exit %d, response %s", code, out)
	}
}
