// Check library declarations, exact diagnostics, and preservation of binary term files.

package rules_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/fabricahq/code-rules/internal/rules"
	"os"
	"reflect"
	"testing"
)

// TestLicenseFixtures checks complete manifest declarations against independent expectations.
func TestLicenseFixtures(t *testing.T) {
	data, err := os.ReadFile("../../tests/migration/licenses/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Location string
		Input        struct {
			Manifest string
			Paths    []string
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
		// Exercise the supplied file inventory without opening real files.
		t.Run(test.ID, func(t *testing.T) {
			files := map[string][]byte{"rule-library.json": []byte(test.Input.Manifest)}
			for _, path := range test.Input.Paths {
				files[path] = nil
			}
			got, err := rules.ReadLibraryLicense(files, test.Location)
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

// TestLicenseBytesAndPaths checks binary presence, exact CRLF preservation, and Unicode sorting.
func TestLicenseBytesAndPaths(t *testing.T) {
	license := []byte{0xff, 0, 13, 10}
	notice := []byte("Notice\r\n")
	files := map[string][]byte{"rule-library.json": []byte(`{"formatVersion":1,"license":{"file":"LICENSE","notices":["NOTICE"]}}`), "LICENSE": license, "NOTICE": notice}
	got, err := rules.ReadLibraryLicense(files, "library")
	if err != nil || !bytes.Equal(license, []byte{0xff, 0, 13, 10}) || !bytes.Equal(notice, []byte("Notice\r\n")) {
		t.Fatalf("bytes changed or read failed: %v", err)
	}
	if paths := rules.LicensePaths(got); !reflect.DeepEqual(paths, []string{"LICENSE", "NOTICE"}) {
		t.Fatal(paths)
	}
	paths := rules.LicensePaths(&rules.LicenseDeclaration{Files: []string{"\ue000", "😀", "😀"}, AttributionFiles: []string{"a"}})
	if !reflect.DeepEqual(paths, []string{"a", "😀", "\ue000"}) {
		t.Fatal(paths)
	}
	if _, err := rules.ReadLibraryLicense(nil, "missing"); err == nil {
		t.Fatal("accepted missing manifest")
	}
}
