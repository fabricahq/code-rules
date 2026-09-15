// Check the public rules API against migration expectations and ownership guarantees.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestSharedExpectations(t *testing.T) {
	data, err := os.ReadFile("../../tests/migration/identities/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID        string
		Operation string
		Input     json.RawMessage
		Location  string
		Expected  struct {
			OK    bool
			Value any
			Error *struct{ Name, Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			var value any
			var err error
			switch test.Operation {
			case "groupID", "ruleGroup":
				var text string
				if err := json.Unmarshal(test.Input, &text); err != nil {
					t.Fatal(err)
				}
				if test.Operation == "groupID" {
					value = text
					err = rules.ValidateGroupID(text, test.Location)
				} else {
					value, err = rules.GroupFromPath(text, test.Location)
				}
			case "selection":
				var selection rules.GroupSelection
				selection, err = rules.ParseGroupSelection(test.Input, test.Location)
				if selection.Pattern != "" {
					value = selection.Pattern
				} else {
					value = selection.Groups
				}
			default:
				t.Fatalf("unknown operation %q", test.Operation)
			}
			if !test.Expected.OK {
				var validation *rules.ValidationError
				if !errors.As(err, &validation) {
					t.Fatalf("want ValidationError, got %v", err)
				}
				if err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
					t.Fatalf("got %q at %q; want %q at %q", err.Error(), validation.Location, test.Expected.Error.Message, test.Expected.Error.Location)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			var decoded any
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded, test.Expected.Value) {
				t.Fatalf("got %#v; want %#v", decoded, test.Expected.Value)
			}
		})
	}
}

func TestSelectionOwnsItsResult(t *testing.T) {
	input := json.RawMessage(`["techs/go","practices/testing"]`)
	before := string(input)
	first, err := rules.ParseGroupSelection(input, "groups")
	if err != nil {
		t.Fatal(err)
	}
	first.Groups[0] = "changed"
	second, err := rules.ParseGroupSelection(input, "groups")
	if err != nil {
		t.Fatal(err)
	}
	if string(input) != before || second.Groups[0] != "practices/testing" {
		t.Fatal("parsing mutated input or shared results")
	}
}

func TestMalformedSelectionJSON(t *testing.T) {
	for _, input := range []string{`[`, `[] true`} {
		_, err := rules.ParseGroupSelection(json.RawMessage(input), "groups")
		if err == nil || err.Error() != "groups: expected a groups JSON value" {
			t.Fatalf("%q: got %v", input, err)
		}
	}
}
