// Verify complete rules through the public parser and independent shared cases.

package rules_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestParseSharedExpectations checks exact rule values and typed, zero-result failures against shared fixtures.
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

// TestParsePathBeforeDocument checks that an invalid path fails before document parsing begins.
func TestParsePathBeforeDocument(t *testing.T) {
	_, err := rules.Parse("malformed", "../escape.md", "team")
	_, expected := rules.GroupFromPath("../escape.md", "team:../escape.md")
	if err == nil || err.Error() != expected.Error() {
		t.Fatalf("got %v; want path error %v", err, expected)
	}
}

// FuzzParse checks arbitrary documents for typed failures, zero partial results, and preserved text on success.
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

// TestParseManyEscapedScalars checks a large tag collection without changing its authored text.
func TestParseManyEscapedScalars(t *testing.T) {
	text := escapedTagDocument(2000)
	got, err := rules.Parse(text, "techs/go/example.md", "lab")
	if err != nil {
		t.Fatal(err)
	}
	original, err := rules.SplitDocument(text, "rule")
	if err != nil || got.Metadata != original.Frontmatter || got.Title != "🐹" {
		t.Fatal("large document lost its values or original text")
	}
}

// BenchmarkParseEscapedScalars measures scaling across documents containing many independently quoted escape pairs.
func BenchmarkParseEscapedScalars(b *testing.B) {
	for _, count := range []int{500, 1000, 2000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			text := escapedTagDocument(count)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := rules.Parse(text, "techs/go/example.md", "lab"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// escapedTagDocument creates distinct quoted tags for correctness and scaling checks.
func escapedTagDocument(count int) string {
	var text strings.Builder
	text.WriteString("---\ntitle: \"\\uD83D\\uDC39\"\nimpact: HIGH\nimpactDescription: Avoid failures\nwhenToRead: Editing\ntags:\n")
	for i := 0; i < count; i++ {
		fmt.Fprintf(&text, "  - \"tag-%d \\uD83D\\uDC39\"\n", i)
	}
	text.WriteString("---\nBody")
	return text.String()
}
