// Verify the public output contract through the native executable.

package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestCLIOutputModes covers human initialization and structured success, help, and failures.
func TestCLIOutputModes(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	out, diagnostic, code := runCLI(t, binary, dir, "init")
	if code != 0 || diagnostic != "" || !strings.Contains(out, "Updated files:") || !strings.Contains(out, "config.json") || json.Valid([]byte(out)) {
		t.Fatalf("human init: %d %q %q", code, out, diagnostic)
	}
	for _, tc := range []struct {
		args  []string
		code  int
		field string
	}{
		{[]string{"--json", "build"}, 0, "value"},
		{[]string{"check", "--json"}, 0, "value"},
		{[]string{"--json", "--help"}, 0, "value"},
		{[]string{"--json", "--version"}, 0, "value"},
		{[]string{"library", "--help", "--json"}, 0, "value"},
		{[]string{"nonsense", "--json"}, 2, "error"},
		{[]string{"build", "--json=invalid"}, 2, "error"},
		{[]string{"build", "--json=false", "--bad", "--json"}, 2, "error"},
		{[]string{"build", "--bad-flag", "--json"}, 2, "error"},
		{[]string{"build", "--config", "--json"}, 2, "error"},
		{[]string{"build", "--json", "--config", "missing.json"}, 1, "error"},
		{[]string{"local", "add", "group", "techs/go", "--json"}, 2, "error"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			out, diagnostic, code := runCLI(t, binary, dir, tc.args...)
			var response map[string]json.RawMessage
			if code != tc.code || diagnostic != "" || json.Unmarshal([]byte(out), &response) != nil || response[tc.field] == nil || string(response["ok"]) != map[bool]string{true: "true", false: "false"}[code == 0] {
				t.Fatalf("structured result: %d %q %q", code, out, diagnostic)
			}
		})
	}
}

// TestHumanCheckAndJSONStale reports pending updates without implying that check wrote them.
func TestHumanCheckAndJSONStale(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	if _, diagnostic, code := runCLI(t, binary, dir, "init"); code != 0 {
		t.Fatal(code, diagnostic)
	}
	out, diagnostic, code := runCLI(t, binary, dir, "check")
	if code != 1 || diagnostic != "" || !strings.Contains(out, "No files were changed.") || !strings.Contains(out, "Add: RULES.md") {
		t.Fatal(code, out, diagnostic)
	}
	out, diagnostic, code = runCLI(t, binary, dir, "check", "--json")
	var result struct {
		OK    bool
		Value struct{ Added []string }
		Error struct{ Kind string }
	}
	if code != 1 || diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil || result.OK || result.Error.Kind != "stale_output" || len(result.Value.Added) == 0 {
		t.Fatal(code, out, diagnostic)
	}
	for _, args := range [][]string{{"build", "--json", "--bad", "--json=false"}, {"--json=false", "--help"}, {"--json", "--json=false", "--help"}, {"build", "--config=--json"}, {"build", "--", "--json"}} {
		out, diagnostic, _ = runCLI(t, binary, dir, args...)
		if json.Valid([]byte(out)) || (out == "" && diagnostic == "") {
			t.Fatal(args, out, diagnostic)
		}
	}
}
