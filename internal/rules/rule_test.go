// Verify complete rules through the public parser and independent shared cases.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestParseSharedExpectations(t *testing.T) {
	data, err := os.ReadFile("../../tests/migration/rules/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID       string
		Input    struct{ Text, Path, Source string }
		Expected struct {
			OK    bool
			Value rules.Rule
			Error *struct{ Name, Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			got, err := rules.Parse(test.Input.Text, test.Input.Path, test.Input.Source)
			if test.Expected.OK {
				if err != nil || !reflect.DeepEqual(got, test.Expected.Value) {
					t.Fatalf("got %#v, %v; want %#v", got, err, test.Expected.Value)
				}
				return
			}
			var validation *rules.ValidationError
			if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
				t.Fatalf("got %#v, %v; want %s", got, err, test.Expected.Error.Message)
			}
			if !reflect.DeepEqual(got, rules.Rule{}) {
				t.Fatalf("failure returned partial data: %#v", got)
			}
		})
	}
}

func TestParsePathBeforeDocument(t *testing.T) {
	_, err := rules.Parse("malformed", "../escape.md", "team")
	_, expected := rules.GroupFromPath("../escape.md", "team:../escape.md")
	if err == nil || err.Error() != expected.Error() {
		t.Fatalf("got %v; want path error %v", err, expected)
	}
}

func FuzzParse(f *testing.F) {
	f.Add("---\ntitle: Rule\nimpact: HIGH\nimpactDescription: Avoid failure.\nwhenToRead: When coding.\n---\nBody")
	f.Add("---\na: &a [*a]\n---\nBody")
	f.Add("---\ntitle: [\n---\nBody")
	f.Fuzz(func(t *testing.T, text string) {
		got, err := rules.Parse(text, "techs/go/example.md", "fuzz")
		if err != nil {
			var validation *rules.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("untyped failure: %v", err)
			}
			if !reflect.DeepEqual(got, rules.Rule{}) {
				t.Fatal("partial result on failure")
			}
			return
		}
		raw, err := rules.SplitDocument(text, "rule")
		if err != nil || got.Metadata != raw.Frontmatter || got.Body != raw.Body {
			t.Fatal("successful parse changed authored text")
		}
	})
}
