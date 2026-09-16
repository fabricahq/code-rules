// Check source-relative destinations and CommonMark integration through the public link API.

package rules_test

import (
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestMarkdownLinks ignores code/frontmatter and preserves source ranges for references and nested syntax.
func TestMarkdownLinks(t *testing.T) {
	document := "---\nfield: '[ignored](secret)'\n---\n[one](assets/a/file\\(1\\).png) ![pic][ref]\n\n[ref]: </assets/image%20one.png>\n[unused]: other.md\n\n`[code](none)`\n\n```md\n[code](none)\n```\n"
	got, err := rules.MarkdownLinks(document)
	if err != nil {
		t.Fatal(err)
	}
	urls := []string{}
	for _, link := range got {
		urls = append(urls, link.URL)
		if link.End <= link.Start || link.End > len(document) {
			t.Fatalf("bad range %+v", link)
		}
	}
	if !reflect.DeepEqual(urls, []string{"assets/a/file(1).png", "/assets/image%20one.png", "other.md"}) {
		t.Fatalf("destinations: %v", urls)
	}
	empty, err := rules.MarkdownLinks("plain text")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty: %v %v", empty, err)
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
