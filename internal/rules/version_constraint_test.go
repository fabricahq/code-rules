// Exercise native constraints against independent fixtures and public API boundaries.
package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestVersionConstraintsSharedExpectations pins native syntax and matching semantics.
func TestVersionConstraintsSharedExpectations(t *testing.T) {
	data, err := os.ReadFile("../../tests/migration/version-constraints/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Operation, Location string
		Input                   json.RawMessage
		Expected                struct {
			OK    bool
			Value json.RawMessage
			Error *struct{ Name, Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		// Compare each result with authored expectations, including exact diagnostics.
		t.Run(test.ID, func(t *testing.T) {
			var got any
			var err error
			switch test.Operation {
			case "versionConstraint":
				var text string
				if err := json.Unmarshal(test.Input, &text); err != nil {
					t.Fatal(err)
				}
				var constraint rules.VersionConstraint
				constraint, err = rules.ParseVersionConstraint(text, test.Location)
				got = constraint.String()
				if err != nil && !reflect.DeepEqual(constraint, rules.VersionConstraint{}) {
					t.Fatal("returned partial constraint")
				}
			case "versionMatch":
				var input struct{ Constraint, Version string }
				if err := json.Unmarshal(test.Input, &input); err != nil {
					t.Fatal(err)
				}
				var constraint rules.VersionConstraint
				constraint, err = rules.ParseVersionConstraint(input.Constraint, test.Location+".constraint")
				if err == nil {
					got, err = constraint.Matches(input.Version, test.Location+".version")
				}
			default:
				t.Fatalf("unhandled operation %s", test.Operation)
			}
			if !test.Expected.OK {
				var validation *rules.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
					t.Fatalf("got %v; want %+v", err, test.Expected.Error)
				}
				return
			}
			var expected any
			if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
				t.Fatal(err)
			}
			if err != nil || !reflect.DeepEqual(got, expected) {
				t.Fatalf("got %#v, %v; want %#v", got, err, expected)
			}
		})
	}
}

// TestVersionConstraintInvalidMatch distinguishes invalid inputs from valid nonmatches.
func TestVersionConstraintInvalidMatch(t *testing.T) {
	parsed, err := rules.ParseVersionConstraint(" >= 1.2.0, < 2.0.0 ", "constraint")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		constraint rules.VersionConstraint
		tag        string
	}{
		{"zero constraint", rules.VersionConstraint{}, "1.2.3"},
		{"incomplete version", parsed, "1.2"},
	} {
		// Verify failed calls return false and a diagnostic owned by the caller.
		t.Run(test.name, func(t *testing.T) {
			got, err := test.constraint.Matches(test.tag, "release.version")
			var validation *rules.ValidationError
			if got || !errors.As(err, &validation) || validation.Location != "release.version" {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
	for _, tag := range []string{"1.2.3", "2.0.0", "1.2.3"} {
		got, err := parsed.Matches(tag, "release")
		if err != nil || got != (tag == "1.2.3") {
			t.Fatalf("%s: %v, %v", tag, got, err)
		}
	}
	if parsed.String() != " >= 1.2.0, < 2.0.0 " {
		t.Fatal("matching changed original text")
	}
}
