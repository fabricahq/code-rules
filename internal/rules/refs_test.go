// Check exact ref classification and version-tag validation against shared expectations.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestRefsSharedExpectations checks values and typed errors through the public Go API.
func TestRefsSharedExpectations(t *testing.T) {
	data, err := os.ReadFile("../../tests/migration/refs/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Operation, Input, Location string
		Expected                       struct {
			OK    bool
			Value json.RawMessage
			Error *struct{ Name, Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		// Check each fixture independently so boundary failures identify their input.
		t.Run(test.ID, func(t *testing.T) {
			switch test.Operation {
			case "gitRef":
				got, err := rules.ParseGitRef(test.Input, test.Location)
				if !test.Expected.OK {
					var validation *rules.ValidationError
					if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
						t.Fatalf("got %v; want %s", err, test.Expected.Error.Message)
					}
					if got != (rules.GitRef{}) {
						t.Fatalf("failure returned partial data: %#v", got)
					}
					return
				}
				var expected rules.GitRef
				if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
					t.Fatal(err)
				}
				if err != nil || got != expected {
					t.Fatalf("got %#v, %v; want %#v", got, err, expected)
				}
			case "tagVersion":
				got, err := rules.TagVersion(test.Input, test.Location)
				if !test.Expected.OK {
					var validation *rules.ValidationError
					if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
						t.Fatalf("got %v; want %s", err, test.Expected.Error.Message)
					}
					if got != "" {
						t.Fatalf("failure returned partial version: %q", got)
					}
					return
				}
				var expected string
				if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
					t.Fatal(err)
				}
				if err != nil || got != expected {
					t.Fatalf("got %q, %v; want %q", got, err, expected)
				}
			default:
				t.Fatalf("unhandled operation: %s", test.Operation)
			}
		})
	}
}

// TestGitRefErrorLocation checks that the caller owns the diagnostic path.
func TestGitRefErrorLocation(t *testing.T) {
	_, err := rules.ParseGitRef("refs/heads/main", "config.sources.other.ref")
	var validation *rules.ValidationError
	if !errors.As(err, &validation) || validation.Location != "config.sources.other.ref" {
		t.Fatalf("unexpected error location: %v", err)
	}
}
