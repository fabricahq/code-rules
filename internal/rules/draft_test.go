// Check that a rule still marked as a draft is recognized in either marker's spelling.

package rules_test

import (
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestHasDraftMarker_RecognizesTheCodeRulesMarker that authoring wrote before Code Rules was renamed Syllabus.
func TestHasDraftMarker_RecognizesTheCodeRulesMarker(t *testing.T) {
	for document, want := range map[string]bool{
		"Guidance.\n" + rules.DraftMarker + "\n": true,
		"Guidance.\n<!-- code-rules:draft -->\n": true,
		"Guidance.\n":                            false,
	} {
		if got := rules.HasDraftMarker([]byte(document)); got != want {
			t.Errorf("HasDraftMarker(%q) = %v; want %v", document, got, want)
		}
	}
}
