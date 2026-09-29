// Check change note parsing against independent fixtures, including every documented rejection.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

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
				var validation *rules.ValidationError
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

// TestValidateRuleID_AcceptsEveryLoadableRulePath keeps change notes able to name every rule the loader accepts.
func TestValidateRuleID_AcceptsEveryLoadableRulePath(t *testing.T) {
	for _, path := range []string{"practices/testing/verify-retry-limits.md", "practices/testing/example.md.md", "techs/react/hooks/test-in-isolation.md"} {
		if _, err := rules.GroupFromPath(path, "path"); err != nil {
			t.Fatalf("%s is not a loadable rule path: %v", path, err)
		}
		if err := rules.ValidateRuleID(strings.TrimSuffix(path, ".md"), "id"); err != nil {
			t.Errorf("rule %s has an ID change notes reject: %v", path, err)
		}
	}
}
