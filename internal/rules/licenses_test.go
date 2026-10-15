// Check pure parsing of a configuration's library section, exact diagnostics, and deterministic retained-path ordering.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestLicenseFixtures checks the license declarations of complete configurations against independent expectations.
// A configuration without a library section isn't a library, which notLibrary expects.
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
			OK         bool
			NotLibrary bool
			Value      json.RawMessage
			Error      *struct{ Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			library, err := rules.ParseLibraryManifest([]byte(test.Input.Manifest), test.Location)
			if !test.Expected.OK {
				var validation errs.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.ValidationLocation() != test.Expected.Error.Location || library != nil {
					t.Fatalf("got %+v, %v; want %+v", library, err, test.Expected.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if (library == nil) != test.Expected.NotLibrary {
				t.Fatalf("got library %+v; want a library: %v", library, !test.Expected.NotLibrary)
			}
			var got *rules.LicenseDeclaration
			if library != nil {
				got = library.License
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

// TestPublishedManifest keeps only a configuration's library section, in canonical JSON, and publishes nothing for
// a configuration that isn't a library's.
func TestPublishedManifest(t *testing.T) {
	text := "# Comments stay in the repository.\nschemaVersion: 2\nsources:\n  upstream:\n    repository: https://github.com/acme/upstream.git\n    rules:\n      include: ['techs/*']\nlibrary:\n  license:\n    spdxExpression: MIT\n    file: LICENSE.md\n    notices: []\n"
	published, err := rules.PublishedManifest([]byte(text), "config.yaml")
	want := `{"library":{"license":{"file":"LICENSE.md","notices":[],"spdxExpression":"MIT"}},"schemaVersion":2,"sources":{}}` + "\n"
	if err != nil || string(published) != want {
		t.Fatalf("got %s, %v; want %s", published, err, want)
	}
	if again, err := rules.PublishedManifest(published, "config.yaml"); err != nil || string(again) != want {
		t.Fatalf("republished %s, %v", again, err)
	}
	if none, err := rules.PublishedManifest([]byte("schemaVersion: 2\nsources: {}\n"), "config.yaml"); err != nil || none != nil {
		t.Fatalf("got %s, %v for a project's configuration", none, err)
	}
	if _, err := rules.PublishedManifest([]byte("schemaVersion: 2\nsources: {}\nlibrary:\n  skills: {}\n"), "config.yaml"); err == nil {
		t.Fatal("published a library section with skill levels")
	}
}

// TestLicenseDeclaredInTheLegacyDirectory reads a license file declared in .code-rules/assets/, as a library release
// from before Code Rules was renamed Syllabus declares it, as the shared asset it is.
func TestLicenseDeclaredInTheLegacyDirectory(t *testing.T) {
	text := "schemaVersion: 2\nsources: {}\nlibrary:\n  license:\n    spdxExpression: MIT\n    file: .code-rules/assets/LICENSE.md\n    notices: [.code-rules/assets/NOTICE.md]\n"
	library, err := rules.ParseLibraryManifest([]byte(text), "config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := library.License; !reflect.DeepEqual(got.Files, []string{"assets/LICENSE.md"}) || !reflect.DeepEqual(got.AttributionFiles, []string{"assets/NOTICE.md"}) {
		t.Fatalf("got %+v; want the shared assets", got)
	}
}
