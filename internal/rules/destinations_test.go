// Check destination spans, escaping, and relative links through the public API that rewrites links in place.

package rules_test

import (
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestMarkdownDestinations_LocatesEachDestinationOnce spans every kind of destination, including empty ones,
// and lists a definition that reference links share once.
func TestMarkdownDestinations_LocatesEachDestinationOnce(t *testing.T) {
	text := "[a](one.md) ![b](<two words.png>) [c]() `[code](none)`\n\n[d][ref] [ref]\n\n[ref]: three.md\n"
	got, err := rules.MarkdownDestinations(text)
	want := []rules.MarkdownDestination{
		{Start: 4, End: 10, Value: "one.md"},
		{Start: 18, End: 31, Value: "two words.png"},
		{Start: 38, End: 38, Value: ""},
		{Start: 79, End: 87, Value: "three.md"},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v; want %+v", got, err, want)
	}
	for _, destination := range got {
		if span := text[destination.Start:destination.End]; span != destination.Value {
			t.Errorf("span %d:%d holds %q, not %q", destination.Start, destination.End, span, destination.Value)
		}
	}
}

// TestMarkdownBodyStart skips only a complete frontmatter envelope, after an optional byte order mark.
func TestMarkdownBodyStart(t *testing.T) {
	for _, test := range []struct {
		document string
		want     int
	}{
		{"---\ntitle: x\n---\nBody", 17},
		{"\ufeff---\r\ntitle: x\r\n---\r\nBody", 23},
		{"---\nunclosed", 0},
		{"Body only", 0},
	} {
		if got := rules.MarkdownBodyStart(test.document); got != test.want {
			t.Errorf("%q: got %d, want %d", test.document, got, test.want)
		}
	}
}

// TestRelativeLink_EscapesSegments links between root-relative paths with URL-escaped segments.
func TestRelativeLink_EscapesSegments(t *testing.T) {
	for _, test := range []struct{ from, to, want string }{
		{"techs/go/errors.md", "techs/go/assets/errors/a b.svg", "assets/errors/a%20b.svg"},
		{"techs/go/assets/errors/guide.md", "techs/go/assets/errors/diagrams/x.svg", "diagrams/x.svg"},
		{"techs/go/assets/errors/guide.md", "assets/x.svg", "../../../../assets/x.svg"},
	} {
		if got := rules.RelativeLink(test.from, test.to); got != test.want {
			t.Errorf("%s -> %s: got %q, want %q", test.from, test.to, got, test.want)
		}
	}
}

// TestEscapeDestination_KeepsOneDestination escapes characters that would end or split a destination.
func TestEscapeDestination_KeepsOneDestination(t *testing.T) {
	if got := rules.EscapeDestination(`a b(c)<d>&e\f`); got != "a%20b%28c%29%3Cd%3E&amp;e%5Cf" {
		t.Fatalf("got %q", got)
	}
}
