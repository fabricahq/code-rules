// Check release tag message parsing against independent fixtures, including record consistency.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestReleaseMessageFixtures checks the notes, the parsed record, and exact diagnostics.
func TestReleaseMessageFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/release-records/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Tag, Message string
		Expected         struct {
			OK    bool
			Notes string
			Value json.RawMessage
			Error *struct{ Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			notes, got, err := rules.ParseReleaseMessage(test.Tag, []byte(test.Message))
			if !test.Expected.OK {
				var validation *rules.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var expected rules.ReleaseRecord
			if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
				t.Fatal(err)
			}
			if notes != test.Expected.Notes || !reflect.DeepEqual(got, expected) {
				t.Fatalf("got %q, %+v; want %q, %+v", notes, got, test.Expected.Notes, expected)
			}
		})
	}
}

// TestParseReleaseTagObject_SkipsHeadersAndSignature reads the record from a signed tag object's message only.
func TestParseReleaseTagObject_SkipsHeadersAndSignature(t *testing.T) {
	object := "object 0123456789012345678901234567890123456789\ntype commit\ntag release/2\ntagger Fixture <fixture@example.invalid> 0 +0000\n\nNotes.\n\n---\nrelease: 2\nrules: {}\n-----BEGIN PGP SIGNATURE-----\nU0lH\n-----END PGP SIGNATURE-----\n"
	notes, record, err := rules.ParseReleaseTagObject("release/2", []byte(object))
	if err != nil || notes != "Notes." || record.Release != 2 || len(record.Rules) != 0 {
		t.Fatalf("got %q, %+v, %v", notes, record, err)
	}
	if _, _, err := rules.ParseReleaseTagObject("release/3", []byte(object)); err == nil {
		t.Fatal("accepted a record for another library release")
	}
}
