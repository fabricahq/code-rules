// Exercise the field rules of group metadata, decoded to JSON, against explicit shared expectations and ownership
// guarantees. The fixtures reach cases, such as repeated keys and escaped surrogates, that YAML decoding rules out
// before these rules apply, so they call the unexported stage directly.

package coderules

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/errs"
)

func TestGroupMetadataSharedExpectations(t *testing.T) {
	data, err := os.ReadFile("testdata/group-metadata/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Input, Location string
		Expected            struct {
			OK    bool
			Value GroupMetadata
			Error *struct{ Name, Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			got, err := groupMetadataFields(json.RawMessage(test.Input), test.Location)
			if test.Expected.OK {
				if err != nil || !reflect.DeepEqual(got, test.Expected.Value) {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Value)
				}
				return
			}
			var validation errs.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("want ValidationError, got %v", err)
			}
			if err.Error() != test.Expected.Error.Message || validation.ValidationLocation() != test.Expected.Error.Location {
				t.Fatalf("got %q at %q; want %q at %q", err.Error(), validation.ValidationLocation(), test.Expected.Error.Message, test.Expected.Error.Location)
			}
			if !reflect.DeepEqual(got, GroupMetadata{}) {
				t.Fatalf("failed parsing returned partial metadata: %+v", got)
			}
		})
	}
}

func TestGroupMetadataOwnsItsResult(t *testing.T) {
	input := json.RawMessage(`{"name":"Go","description":"Go rules","whenToRead":"When editing."}`)
	before := string(input)
	first, err := groupMetadataFields(input, "group")
	if err != nil {
		t.Fatal(err)
	}
	first.WhenToRead = "changed"
	second, err := groupMetadataFields(input, "group")
	if err != nil {
		t.Fatal(err)
	}
	if string(input) != before {
		t.Fatal("parsing mutated the input")
	}
	for i := range input {
		input[i] = ' '
	}
	if second.Name != "Go" || second.Description != "Go rules" || second.WhenToRead != "When editing." {
		t.Fatalf("result shares storage with another result or the input: %+v", second)
	}
}

func TestGroupMetadataRejectsInvalidUTF8(t *testing.T) {
	input := append([]byte(`{"name":"`), 0xff)
	input = append(input, []byte(`","description":"Go rules","whenToRead":"When editing."}`)...)
	got, err := groupMetadataFields(input, "group")
	var validation errs.ValidationError
	if !errors.As(err, &validation) || validation.ValidationLocation() != "group.name" || validation.ValidationProblem() != "expected valid Unicode text: invalid UTF-8 or unpaired surrogate escape" {
		t.Fatalf("want Unicode validation error at group.name, got %v", err)
	}
	if got != (GroupMetadata{}) {
		t.Fatalf("failed parsing returned partial metadata: %+v", got)
	}
}
