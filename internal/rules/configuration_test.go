// Verify complete configurations against independent public API expectations.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// projectedSource is a source as the fixtures describe it: its fields and, when it has a ref, the ref as GitRef
// parses and normalizes it.
type projectedSource struct {
	rules.Source
	ParsedRef *rules.GitRef `json:"parsedRef,omitempty"`
}

// projection returns config with each source's parsed ref, for comparison with a fixture's expected value.
func projection(config rules.Configuration) map[string][]projectedSource {
	sources := make([]projectedSource, 0, len(config.Sources))
	for _, source := range config.Sources {
		projected := projectedSource{Source: source}
		if ref, ok := source.GitRef(); ok {
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
				var validation *rules.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
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
			encoded, err := json.Marshal(projection(got))
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

// TestSourceSameRef matches a recorded ref to the source's ref by the revision they name, not their spelling.
func TestSourceSameRef(t *testing.T) {
	for _, test := range []struct {
		source, recorded string
		same             bool
	}{
		{"release/5", "release/5", true},
		{"release/5", "refs/tags/release/5", true},
		{"refs/tags/release/5", "release/5", true},
		{"ABCDEF0123456789ABCDEF0123456789ABCDEF01", "abcdef0123456789abcdef0123456789abcdef01", true},
		{"", "", true},
		{"release/5", "release/6", false},
		{"release/5", "", false},
		{"", "release/5", false},
		{"release/5", "refs/heads/release/5", false},
	} {
		if got := (rules.Source{Name: "team", Ref: test.source}).SameRef(test.recorded); got != test.same {
			t.Errorf("source ref %q, recorded %q: same %v, want %v", test.source, test.recorded, got, test.same)
		}
	}
}
