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
