// Exercise group metadata parsing against explicit shared expectations and ownership guarantees.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestGroupMetadataSharedExpectations(t *testing.T) {
	data, err := os.ReadFile("../../tests/migration/group-metadata/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Input, Location string
		Expected            struct {
			OK    bool
			Value rules.GroupMetadata
			Error *struct{ Name, Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			got, err := rules.ParseGroupMetadata(json.RawMessage(test.Input), test.Location)
			if test.Expected.OK {
				if err != nil || !reflect.DeepEqual(got, test.Expected.Value) {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Value)
				}
				return
			}
			var validation *rules.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("want ValidationError, got %v", err)
			}
			if err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
				t.Fatalf("got %q at %q; want %q at %q", err.Error(), validation.Location, test.Expected.Error.Message, test.Expected.Error.Location)
			}
			if !reflect.DeepEqual(got, rules.GroupMetadata{}) {
				t.Fatalf("failed parsing returned partial metadata: %+v", got)
			}
		})
	}
}

func TestGroupMetadataOwnsItsResult(t *testing.T) {
	input := json.RawMessage(`{"name":"Go","description":"Go rules","whenToRead":["When editing."]}`)
	before := string(input)
	first, err := rules.ParseGroupMetadata(input, "group")
	if err != nil {
		t.Fatal(err)
	}
	first.WhenToRead[0] = "changed"
	second, err := rules.ParseGroupMetadata(input, "group")
	if err != nil {
		t.Fatal(err)
	}
	if string(input) != before {
		t.Fatal("parsing mutated the input")
	}
	for i := range input {
		input[i] = ' '
	}
	if second.Name != "Go" || second.Description != "Go rules" || second.WhenToRead[0] != "When editing." {
		t.Fatalf("result shares storage with another result or the input: %+v", second)
	}
}
