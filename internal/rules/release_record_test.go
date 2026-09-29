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
