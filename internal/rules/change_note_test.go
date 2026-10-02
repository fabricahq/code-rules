// Check change note parsing against independent fixtures, including every documented rejection.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/decode"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestChangeNoteFixtures checks parsed notes and exact diagnostics.
func TestChangeNoteFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/change-notes/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Location, Input string
		Expected            struct {
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
			got, err := rules.ParseChangeNote([]byte(test.Input), test.Location)
			if !test.Expected.OK {
				var validation *decode.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var expected rules.ChangeNote
			if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("got %+v; want %+v", got, expected)
			}
		})
	}
}

// TestLargerChange_PicksTheLargestVersionChange resolves several pending notes on one rule.
func TestLargerChange_PicksTheLargestVersionChange(t *testing.T) {
	for _, test := range []struct{ a, b, want coderules.Change }{
		{coderules.ChangePatch, coderules.ChangeMajor, coderules.ChangeMajor},
		{coderules.ChangeMajor, coderules.ChangeMinor, coderules.ChangeMajor},
		{coderules.ChangeMinor, coderules.ChangePatch, coderules.ChangeMinor},
		{coderules.ChangePatch, coderules.ChangePatch, coderules.ChangePatch},
	} {
		if got := rules.LargerChange(test.a, test.b); got != test.want {
			t.Errorf("larger of %s and %s: got %s, want %s", test.a, test.b, got, test.want)
		}
	}
}
