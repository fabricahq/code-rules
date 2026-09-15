// Test exact document boundaries against shared fixtures and the splitter's narrow contract.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestSplitDocumentSharedExpectations(t *testing.T) {
	data, err := os.ReadFile("../../tests/migration/rule-documents/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Input, Location string
		Expected            struct {
			OK    bool
			Value rules.DocumentText
			Error *struct{ Name, Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			got, err := rules.SplitDocument(test.Input, test.Location)
			if test.Expected.OK {
				if err != nil || got != test.Expected.Value {
					t.Fatalf("got %#v, %v; want %#v", got, err, test.Expected.Value)
				}
				return
			}
			var validation *rules.ValidationError
			if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
				t.Fatalf("got %#v, %v; want %s", got, err, test.Expected.Error.Message)
			}
			if got != (rules.DocumentText{}) {
				t.Fatalf("failed splitting returned partial text: %#v", got)
			}
		})
	}
}

func TestSplitDocumentDoesNotInterpretContent(t *testing.T) {
	// The full rule parser will validate these captures later. Splitting alone
	// must preserve them, including empty text, invalid YAML, and arbitrary bytes.
	for _, test := range []struct {
		name, input, frontmatter, body string
	}{
		{"empty body at EOF", "---\ntitle: Example\n---", "title: Example", ""},
		{"empty body after newline", "---\ntitle: Example\n---\r\n", "title: Example", ""},
		{"empty frontmatter", "---\n\n---\nBody", "", "Body"},
		{"invalid YAML", "---\ntitle: [\n---\nBody", "title: [", "Body"},
		{"whitespace body", "---\nx\n---\n \t\n", "x", " \t\n"},
		{"raw bytes", "---\n\xff\n---\n\xfe", "\xff", "\xfe"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := rules.SplitDocument(test.input, "rule")
			want := rules.DocumentText{Frontmatter: test.frontmatter, Body: test.body}
			if err != nil || got != want {
				t.Fatalf("got %#v, %v; want %#v", got, err, want)
			}
		})
	}
}
