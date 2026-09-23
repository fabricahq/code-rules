// Check pure manifest parsing, exact diagnostics, and deterministic retained-path ordering.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestLicenseFixtures checks complete manifest declarations against independent expectations.
func TestLicenseFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/licenses/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Location string
		Input        struct {
			Manifest string
		}
		Expected struct {
			OK    bool
			Value json.RawMessage
			Error *struct{ Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			got, err := rules.ParseLibraryLicense([]byte(test.Input.Manifest), test.Location)
			if !test.Expected.OK {
				var validation *rules.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location || got != nil {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var expected *rules.LicenseDeclaration
			if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("got %+v; want %+v", got, expected)
			}
		})
	}
}

// TestLicensePaths checks unique paths and Unicode sorting without modifying the declaration.
func TestLicensePaths(t *testing.T) {
	declaration := &rules.LicenseDeclaration{Files: []string{"\ue000", "😀", "😀"}, AttributionFiles: []string{"a"}}
	paths := rules.LicensePaths(declaration)
	if !reflect.DeepEqual(paths, []string{"a", "😀", "\ue000"}) {
		t.Fatal(paths)
	}
	if !reflect.DeepEqual(declaration.Files, []string{"\ue000", "😀", "😀"}) {
		t.Fatal("modified license declaration")
	}
	if paths := rules.LicensePaths(nil); len(paths) != 0 {
		t.Fatalf("absent license paths: %v", paths)
	}
}
