// Verify complete configurations against independent public API expectations.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/rules"
)

// projectedSource is a source as the fixtures describe it: its fields and, when it has a ref, the ref's canonical
// form.
type projectedSource struct {
	rules.Source
	ParsedRef *projectedRef `json:"parsedRef,omitempty"`
}

// configurationProjection returns config with each source's canonical ref, for comparison with a fixture's expected value.
func configurationProjection(config rules.Configuration) map[string][]projectedSource {
	sources := make([]projectedSource, 0, len(config.Sources))
	for _, source := range config.Sources {
		projected := projectedSource{Source: source}
		if !source.Ref.IsZero() {
			ref := projection(source.Ref)
			projected.ParsedRef = &ref
		}
		sources = append(sources, projected)
	}
	return map[string][]projectedSource{"sources": sources}
}

// TestConfigurationFixtures checks successful projections and precise error locations.
func TestConfigurationFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/configuration/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Input string
		Expected  struct {
			OK    bool
			Value json.RawMessage
			Error *struct{ Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		// Exercise each complete document rather than mirroring its parser helpers.
		t.Run(test.ID, func(t *testing.T) {
			got, err := rules.ParseConfiguration(json.RawMessage(test.Input))
			if !test.Expected.OK {
				var validation errs.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.ValidationLocation() != test.Expected.Error.Location {
					t.Fatalf("got %v; want %+v", err, test.Expected.Error)
				}
				if !reflect.DeepEqual(got, rules.Configuration{}) {
					t.Fatal("partial result on failure")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(configurationProjection(got))
			if err != nil {
				t.Fatal(err)
			}
			var actual, expected any
			if err := json.Unmarshal(encoded, &actual); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("got %s; want %s", encoded, test.Expected.Value)
			}
		})
	}
}

// TestParseConfigurationYAML_VersionOneShowsAValidVersionTwoExample rejects a version 1 configuration with a
// message whose example is itself a valid version 2 configuration selecting a group and a rule.
func TestParseConfigurationYAML_VersionOneShowsAValidVersionTwoExample(t *testing.T) {
	_, err := rules.ParseConfigurationYAML([]byte("schemaVersion: 1\nsources:\n  team:\n    repository: https://github.com/example/rules.git\n    groups: [techs/go]\n"))
	var validation errs.ValidationError
	if !errors.As(err, &validation) || validation.ValidationLocation() != "schemaVersion" {
		t.Fatalf("got %v, want a failure at schemaVersion", err)
	}
	start := strings.Index(validation.ValidationProblem(), "\n\nschemaVersion: 2\n")
	end := strings.LastIndex(validation.ValidationProblem(), "\n\n")
	if start < 0 || end <= start {
		t.Fatalf("no example in %q", validation.ValidationProblem())
	}
	example, err := rules.ParseConfigurationYAML([]byte(validation.ValidationProblem()[start+2 : end]))
	if err != nil {
		t.Fatal(err)
	}
	team := example.Sources[0]
	if !team.Groups.Equal(rules.GroupSelection{Groups: []string{"techs/go"}}) || !reflect.DeepEqual(team.Rules, []string{"practices/testing/verify-retry-limits"}) || len(team.Exclude) != 1 || len(team.Pins) != 1 {
		t.Fatalf("example parsed as %+v", team)
	}
}
