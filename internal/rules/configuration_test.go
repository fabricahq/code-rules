// Verify complete configurations against independent public API expectations.

package rules_test

import (
	"encoding/json"
	"errors"
	"github.com/fabricahq/code-rules/internal/rules"
	"os"
	"reflect"
	"testing"
)

// TestConfigurationFixtures checks successful projections and precise error locations.
func TestConfigurationFixtures(t *testing.T) {
	data, err := os.ReadFile("../../tests/migration/configuration/cases.json")
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
			encoded, err := json.Marshal(got)
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
