// Verify YAML configuration semantics and refusal of ambiguous or unsupported syntax.

package rules_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
	"go.yaml.in/yaml/v4"
)

func TestYAMLConfigurationMatchesValidSchemaFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/configuration/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Input string
		Expected  struct{ OK bool }
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		if !tc.Expected.OK {
			continue
		}
		t.Run(tc.ID, func(t *testing.T) {
			var value any
			if err := json.Unmarshal([]byte(tc.Input), &value); err != nil {
				t.Fatal(err)
			}
			encoded, err := yaml.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			want, err := rules.ParseConfiguration([]byte(tc.Input))
			if err != nil {
				t.Fatal(err)
			}
			got, err := rules.ParseConfigurationYAML(encoded)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("YAML differs: %s\n%+v\n%v", encoded, got, err)
			}
		})
	}
}

func TestYAMLConfigurationRejectsAmbiguousInput(t *testing.T) {
	for name, input := range map[string]string{
		"empty": "", "nonobject": "[]", "duplicate": "schemaVersion: 1\nschemaVersion: 1\nsources: {}",
		"unknown": "schemaVersion: 1\nsources: {}\nextra: true", "multiple": "schemaVersion: 1\nsources: {}\n---\nschemaVersion: 1\nsources: {}",
		"anchor": "schemaVersion: 1\nsources: &sources {}", "alias": "schemaVersion: 1\nsources: *sources",
		"tag": "schemaVersion: !!int 1\nsources: {}", "merge": "schemaVersion: 1\nsources: {<<: {}}",
		"nonstringkey": "schemaVersion: 1\nsources: {123: {}}", "null": "schemaVersion: 1\nsources: null",
		"infinity": "schemaVersion: .inf\nsources: {}", "utf8": "schemaVersion: 1\nsources: {}\n#\xff",
		"deep":           "schemaVersion: 1\nsources: " + strings.Repeat("[", 40) + strings.Repeat("]", 40),
		"duplicatealias": "schemaVersion: 1\nsources: {team: {}, team: {}}",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := rules.ParseConfigurationYAML([]byte(input))
			if err == nil || !reflect.DeepEqual(got, rules.Configuration{}) {
				t.Fatalf("accepted invalid input: %+v, %v", got, err)
			}
		})
	}
}

// TestAppendConfigurationSource preserves authored presentation while validating the edited document.
func TestAppendConfigurationSource(t *testing.T) {
	input := []byte(`# Project guidance
schemaVersion: 1
sources:
  existing:
    repository: 'https://github.com/acme/existing.git' # keep the selected library
    version: '>= 0.1.0, < 0.2.0'
    groups: '*'
    exclude: {}
    replace: {}
`)
	source := rules.Source{
		Repository: "https://github.com/acme/added.git", Ref: "v0.1.0",
		Groups:  rules.GroupSelection{Pattern: "*"},
		Exclude: map[string]string{}, Replace: map[string]rules.Replacement{},
	}
	out, err := rules.AppendConfigurationSource(input, "added", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, retained := range []string{"# Project guidance", "'https://github.com/acme/existing.git' # keep the selected library", "version: '>= 0.1.0, < 0.2.0'", "groups: '*'"} {
		if !strings.Contains(string(out), retained) {
			t.Fatalf("lost authored presentation %q:\n%s", retained, out)
		}
	}
	if strings.Index(string(out), "  existing:") > strings.Index(string(out), "  added:") {
		t.Fatalf("reordered existing source:\n%s", out)
	}
	got, err := rules.ParseConfigurationYAML(out)
	if err != nil || len(got.Sources) != 2 || got.Sources[0].Name != "added" || got.Sources[0].Repository != source.Repository {
		t.Fatalf("invalid appended source: %+v, %v", got, err)
	}
	if out, err := rules.AppendConfigurationSource(input, "existing", source); err == nil || out != nil {
		t.Fatalf("accepted duplicate source: %s, %v", out, err)
	}
}

// TestAppendConfigurationSourceValidatesInputFirst preserves errors from the existing document.
func TestAppendConfigurationSourceValidatesInputFirst(t *testing.T) {
	for _, input := range []string{"schemaVersion: 1", "schemaVersion: 1\nsources: []", "schemaVersion: 1\nsources: {}\nsources: {}"} {
		_, want := rules.ParseConfigurationYAML([]byte(input))
		out, got := rules.AppendConfigurationSource([]byte(input), "invalid alias", rules.Source{})
		if want == nil || got == nil || got.Error() != want.Error() || out != nil {
			t.Fatalf("input error lost: output %s, got %v, want %v", out, got, want)
		}
	}
}
