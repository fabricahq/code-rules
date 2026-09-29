// Verify project add library records the documented selection and revision fields without fetching.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
)

// TestLibrarySelectionFlags records groups, rules, and an optional exact ref, and rejects invalid selections without writes.
func TestLibrarySelectionFlags(t *testing.T) {
	binary := buildCLI(t)
	repository := "https://example.invalid/rules.git"
	for _, tc := range []struct {
		name string
		args []string
		code int
		want map[string]any
	}{
		{"groups without ref", []string{"--groups", "techs/go"}, 0, map[string]any{"repository": repository, "groups": []any{"techs/go"}}},
		{"tag ref", []string{"--ref", "release/5", "--groups", "techs/go"}, 0, map[string]any{"repository": repository, "groups": []any{"techs/go"}, "ref": "release/5"}},
		{"rules only", []string{"--rules", "techs/go/wrap-errors", "--rules", "practices/testing/verify-retry-limits"}, 0,
			map[string]any{"repository": repository, "rules": []any{"practices/testing/verify-retry-limits", "techs/go/wrap-errors"}}},
		{"groups and rules", []string{"--groups", "*", "--rules", "techs/go/wrap-errors"}, 0, map[string]any{"repository": repository, "groups": "*", "rules": []any{"techs/go/wrap-errors"}}},
		{"no selection", nil, 2, nil},
		{"version range", []string{"--ref", ">= 1.2.0, < 2.0.0", "--groups", "techs/go"}, 2, nil},
		{"branch", []string{"--ref", "refs/heads/main", "--groups", "techs/go"}, 2, nil},
		{"group as rule", []string{"--rules", "techs/go"}, 1, nil},
		{"duplicate rules", []string{"--rules", "techs/go/a", "--rules", "techs/go/a"}, 1, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if out, diagnostic, code := runCLI(t, binary, dir, "project", "init"); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
			before := projectFileContents(t, dir)
			args := append([]string{"project", "add", "library", "team", "--repository", repository, "--non-interactive", "--json"}, tc.args...)
			out, diagnostic, code := runCLI(t, binary, dir, args...)
			var result response
			if diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil || code != tc.code {
				t.Fatal(code, out, diagnostic)
			}
			if tc.want == nil {
				if result.OK || result.Error == nil || !reflect.DeepEqual(before, projectFileContents(t, dir)) {
					t.Fatal("invalid selection must fail without writes", out)
				}
				return
			}
			data, err := os.ReadFile(filepath.Join(dir, ".code-rules/config.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			var config struct{ Sources map[string]map[string]any }
			if err := yaml.Unmarshal(data, &config); err != nil || !reflect.DeepEqual(config.Sources["team"], tc.want) {
				t.Fatal(string(data), err)
			}
			if _, err := os.Stat(filepath.Join(dir, ".code-rules/vendor")); !os.IsNotExist(err) {
				t.Fatal("add library fetched files", err)
			}
		})
	}
}
