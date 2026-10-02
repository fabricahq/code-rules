// Check that rendered rules and groups are exactly what coderules reads back.

package rules_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestRenderRule_DraftHasOneBlankLineAfterItsMetadata starts the template's body right after one blank line.
func TestRenderRule_DraftHasOneBlankLineAfterItsMetadata(t *testing.T) {
	metadata := rules.RuleMetadata{Title: "Return errors", WhenToRead: "When returning errors.", Impact: "HIGH", ImpactDescription: "Callers decide."}
	draft, err := rules.RenderRule("techs/go/errors", metadata, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(draft), "\n---\n\n## Return errors\n") {
		t.Fatalf("draft:\n%s", draft)
	}
}

// TestRenderGroup_WritesTrimmedMetadataThatParsesBack keeps text YAML would read as another type, such as a date,
// as text, and trims surrounding whitespace.
func TestRenderGroup_WritesTrimmedMetadataThatParsesBack(t *testing.T) {
	data, err := rules.RenderGroup(coderules.GroupMetadata{Name: " Go ", Description: "2026-10-01", WhenToRead: "true"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := coderules.ParseGroupMetadata(data, "_group.yaml")
	want := coderules.GroupMetadata{Name: "Go", Description: "2026-10-01", WhenToRead: "true"}
	if err != nil || got != want {
		t.Fatalf("got %+v, %v from:\n%s", got, err, data)
	}
}

// TestRenderGroup_RefusesBlankText reports the blank field at its location in the group file.
func TestRenderGroup_RefusesBlankText(t *testing.T) {
	_, err := rules.RenderGroup(coderules.GroupMetadata{Name: "Go", Description: " \t", WhenToRead: "When editing Go."})
	var validation errs.ValidationError
	if !errors.As(err, &validation) || validation.ValidationLocation() != "_group.yaml.description" || validation.ValidationProblem() != "expected nonempty text" {
		t.Fatalf("got %v", err)
	}
}
