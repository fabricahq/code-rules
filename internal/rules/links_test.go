// Check source-relative destinations and CommonMark integration through the public link API.

package rules_test

import (
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestMarkdownTargets covers CommonMark dependencies without treating code or metadata as links.
func TestMarkdownTargets(t *testing.T) {
	for _, test := range []struct {
		document string
		want     []string
	}{
		{"[self]()", []string{"techs/go/r.md"}},
		{"---\n[x](a.md)", []string{"techs/go/a.md"}},
		{"\ufeff---\nfield: '[skip](secret)'\n---\n[x](a.md)", []string{"techs/go/a.md"}},
		{"[one](assets/r/a\\(b\\).png) ![pic][ref]\n\n[ref]: </assets/a%20b.png>\n[unused]: other.md\n\n`[code](none)`", []string{"assets/a b.png", "techs/go/assets/r/a(b).png", "techs/go/other.md"}},
	} {
		got, err := rules.MarkdownTargets(test.document, "techs/go/r.md")
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Errorf("%q: %v, %v", test.document, got, err)
		}
	}
}

// TestRelativeTarget distinguishes remote links, safe normalization, fragments, and escapes.
func TestRelativeTarget(t *testing.T) {
	for _, test := range []struct {
		url, target, suffix string
		local, invalid      bool
	}{
		{"../other.md#part", "techs/other.md", "#part", true, false},
		{"/assets/a%20b.png?raw=1#x", "assets/a b.png", "?raw=1#x", true, false},
		{"#part", "techs/go/errors.md", "#part", true, false},
		{"https://example.com/x", "", "", false, false},
		{"../../../outside", "", "", false, true},
		{"%2e%2e/%2e%2e/%2e%2e/outside", "", "", false, true},
		{"%5csecret", "", "", false, true},
		{"%zz", "", "", false, true},
	} {
		target, suffix, local, err := rules.RelativeTarget(test.url, "techs/go/errors.md")
		if (err != nil) != test.invalid || target != test.target || suffix != test.suffix || local != test.local {
			t.Errorf("%s: %s %s %v %v", test.url, target, suffix, local, err)
		}
	}
}
