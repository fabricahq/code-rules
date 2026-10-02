// Check that adding a fork's attribution keeps a rule's other text, metadata, and earlier attribution.

package rules_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/rules"
)

// forkAttribution is the entry each case adds; its description needs YAML quoting.
var forkAttribution = coderules.Attribution{URL: "https://github.com/acme/rules/blob/0123/techs/go/errors.md", Description: "Forked from version 1.2.0: see #4"}

// TestAddRuleAttribution_AppendsWithoutChangingExistingBytes adds the field after a block mapping's existing
// text, in the document's line endings, so every earlier byte, including comments and quoting, stays.
func TestAddRuleAttribution_AppendsWithoutChangingExistingBytes(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		frontmatter := strings.ReplaceAll("title: \"Wrap errors\"  # kept\nimpact: HIGH\nimpactDescription: >\n  Folded\n  text.\nwhenToRead: When returning errors.", "\n", newline)
		body := strings.ReplaceAll("---\nWrap [errors](assets/errors/a.md).\n", "\n", newline)
		got, err := rules.AddRuleAttribution("---"+newline+frontmatter+newline+body, "techs/go/errors.md", forkAttribution)
		want := "---" + newline + frontmatter + newline + strings.ReplaceAll("attribution:\n  - url: https://github.com/acme/rules/blob/0123/techs/go/errors.md\n    description: 'Forked from version 1.2.0: see #4'\n", "\n", newline) + body
		if err != nil || got != want {
			t.Fatalf("newline %q: got %q, %v; want %q", newline, got, err, want)
		}
	}
}

// TestAddRuleAttribution_KeepsExistingEntriesFirst appends to block and flow attribution lists, and to flow
// frontmatter, keeping the other metadata and comments.
func TestAddRuleAttribution_KeepsExistingEntriesFirst(t *testing.T) {
	existing := coderules.Attribution{URL: "https://example.com/source.md", Description: "Adapted from the source."}
	for _, frontmatter := range []string{
		"title: Wrap\nimpact: HIGH\n# The upstream source.\nattribution:\n  - url: https://example.com/source.md\n    description: Adapted from the source.\nimpactDescription: Matters.\nwhenToRead: Always.",
		"title: Wrap\nimpact: HIGH\nimpactDescription: Matters.\nwhenToRead: Always.\nattribution: [{\"url\": \"https://example.com/source.md\", \"description\": \"Adapted from the source.\"}]",
		"{title: Wrap, impact: HIGH, impactDescription: Matters., whenToRead: Always., attribution: [{url: \"https://example.com/source.md\", description: Adapted from the source.}]}",
	} {
		document := "---\n" + frontmatter + "\n---\nBody.\n"
		got, err := rules.AddRuleAttribution(document, "techs/go/errors.md", forkAttribution)
		if err != nil {
			t.Fatalf("%q: %v", frontmatter, err)
		}
		rule, err := coderules.ParseRule(got, "techs/go/errors.md", "local")
		if err != nil || rule.Title != "Wrap" || rule.WhenToRead != "Always." || !reflect.DeepEqual(rule.Attribution, []coderules.Attribution{existing, forkAttribution}) || !strings.HasSuffix(got, "\n---\nBody.\n") {
			t.Fatalf("%q: got %q, %+v, %v", frontmatter, got, rule, err)
		}
		if strings.Contains(frontmatter, "#") && !strings.Contains(got, "# The upstream source.") {
			t.Fatalf("%q: comment lost: %q", frontmatter, got)
		}
	}
}

// TestAddRuleAttribution_RewritesWhenAppendingWouldChangeAValue keeps a final block scalar's value when appending
// after it would give it another line break.
func TestAddRuleAttribution_RewritesWhenAppendingWouldChangeAValue(t *testing.T) {
	document := "---\ntitle: Wrap\nimpact: HIGH\nimpactDescription: Matters.\nwhenToRead: |+\n  Always.\n---\nBody.\n"
	original, err := coderules.ParseRule(document, "techs/go/errors.md", "local")
	if err != nil {
		t.Fatal(err)
	}
	got, err := rules.AddRuleAttribution(document, "techs/go/errors.md", forkAttribution)
	if err != nil {
		t.Fatal(err)
	}
	rule, err := coderules.ParseRule(got, "techs/go/errors.md", "local")
	if err != nil || rule.WhenToRead != original.WhenToRead || !reflect.DeepEqual(rule.Attribution, []coderules.Attribution{forkAttribution}) {
		t.Fatalf("got %q, %+v, %v", got, rule, err)
	}
}

// TestAddRuleAttribution_RejectsInvalidInput refuses a document that isn't a rule and a URL that attribution
// doesn't accept.
func TestAddRuleAttribution_RejectsInvalidInput(t *testing.T) {
	valid := "---\ntitle: Wrap\nimpact: HIGH\nimpactDescription: Matters.\nwhenToRead: Always.\n---\nBody.\n"
	for _, test := range []struct {
		name, document string
		entry          coderules.Attribution
	}{
		{"not a rule", "No frontmatter.\n", forkAttribution},
		{"SSH URL", valid, coderules.Attribution{URL: "ssh://git@example.com/rules.git", Description: "Forked."}},
	} {
		if got, err := rules.AddRuleAttribution(test.document, "techs/go/errors.md", test.entry); err == nil {
			t.Errorf("%s: got %q, want an error", test.name, got)
		}
	}
}
