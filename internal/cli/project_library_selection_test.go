// Verify one CLI ref input preserves literal refs and records version ranges without fetching.

package cli

import (
	"encoding/json"
	"go.yaml.in/yaml/v4"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLibraryRefSelection(t *testing.T) {
	binary := buildCLI(t)
	for _, tc := range []struct{ input, field, value string }{
		{"v1.2.3", "ref", "v1.2.3"},
		{">= 1.2.0, < 2.0.0", "version", ">= 1.2.0, < 2.0.0"},
		{"refs/heads/main", "", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			dir := t.TempDir()
			if out, diagnostic, code := runCLI(t, binary, dir, "project", "init"); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
			before := projectFileContents(t, dir)
			out, diagnostic, code := runCLI(t, binary, dir, "project", "add", "library", "team", "--repository", "https://example.invalid/rules.git", "--ref", tc.input, "--groups", "techs/go", "--non-interactive", "--json")
			var result response
			if diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil {
				t.Fatal(code, out, diagnostic)
			}
			if tc.field == "" {
				if code != 2 || result.OK || result.Error == nil || !reflect.DeepEqual(before, projectFileContents(t, dir)) {
					t.Fatal("invalid ref must fail without writes", code, out)
				}
				return
			}
			if code != 0 || !result.OK {
				t.Fatal(code, out)
			}
			data, err := os.ReadFile(filepath.Join(dir, ".code-rules/config.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			var config struct{ Sources map[string]map[string]any }
			if err := yaml.Unmarshal(data, &config); err != nil {
				t.Fatal(err)
			}
			source := config.Sources["team"]
			other := "ref"
			if tc.field == "ref" {
				other = "version"
			}
			if source[tc.field] != tc.value || source[other] != nil {
				t.Fatal("wrong revision selection", string(data))
			}
			if _, err := os.Stat(filepath.Join(dir, ".code-rules/vendor")); !os.IsNotExist(err) {
				t.Fatal("add library fetched files", err)
			}
		})
	}
}
