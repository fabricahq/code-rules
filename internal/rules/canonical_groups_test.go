// Check that the canonical group list parser accepts only strict, sorted, uniquely named entries.

package rules_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestParseCanonicalGroups(t *testing.T) {
	input := "# Canonical groups\n" +
		"practices/testing:\n  name: Testing\n  description: >\n    What to test\n    and how.\n" +
		"techs/go:\n  name: Go\n  description: The Go language.\n"
	got, err := rules.ParseCanonicalGroups([]byte(input), "canonical-groups.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := []rules.CanonicalGroup{
		{ID: "practices/testing", Name: "Testing", Description: "What to test and how."},
		{ID: "techs/go", Name: "Go", Description: "The Go language."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v; want %+v", got, want)
	}
}

func TestParseCanonicalGroupsRejectsInvalidLists(t *testing.T) {
	const goEntry = "techs/go:\n  name: Go\n  description: The Go language.\n"
	const testingEntry = "practices/testing:\n  name: Testing\n  description: Tests.\n"
	for _, test := range []struct {
		name, input, location, problem string
	}{
		{"empty list", "{}", "list", "expected at least one group"},
		{"not a mapping", "- techs/go\n", "list", "expected a mapping"},
		{"several documents", goEntry + "---\n" + goEntry, "list", "expected one YAML document"},
		{"duplicate ID", goEntry + goEntry, "list.techs/go", "duplicate field"},
		{"alias", "techs/go: &go\n  name: Go\n  description: Go.\ntechs/golang: *go\n", "list.techs/go", "anchors, aliases, and explicit tags are not supported"},
		{"invalid ID", "techs/Go:\n  name: Go\n  description: Go.\n", "list.techs/Go", "invalid group ID"},
		{"unknown kind", "tools/go:\n  name: Go\n  description: Go.\n", "list.tools/go", "invalid group ID"},
		{"reserved ID", "techs/assets:\n  name: Assets\n  description: Assets.\n", "list.techs/assets", "reserved group name"},
		{"unsorted", goEntry + testingEntry, "list.practices/testing", `"practices/testing" must come before "techs/go"`},
		{"entry not a mapping", "techs/go: Go\n", "list.techs/go", "expected an object"},
		{"unknown field", "techs/go:\n  name: Go\n  description: Go.\n  icon: go.svg\n", "list.techs/go", "unknown field icon"},
		{"alias field", "techs/go:\n  name: Go\n  description: Go.\n  aliases: [techs/golang]\n", "list.techs/go", "unknown field aliases"},
		{"missing name", "techs/go:\n  description: Go.\n", "list.techs/go.name", "expected nonempty text"},
		{"blank name", "techs/go:\n  name: ' '\n  description: Go.\n", "list.techs/go.name", "expected nonempty text"},
		{"non-text name", "techs/go:\n  name: 1\n  description: Go.\n", "list.techs/go.name", "expected nonempty text"},
		{"missing description", "techs/go:\n  name: Go\n", "list.techs/go.description", "expected nonempty text"},
		{"multiline description", "techs/go:\n  name: Go\n  description: |\n    The Go\n    language.\n", "list.techs/go.description", "expected one line"},
		{"multiline name", "techs/go:\n  name: \"G\\no\"\n  description: Go.\n", "list.techs/go.name", "expected one line"},
		{"duplicate name", testingEntry + "techs/go:\n  name: testing\n  description: Go.\n", "list.techs/go.name", `"testing" is already the name of "practices/testing"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := rules.ParseCanonicalGroups([]byte(test.input), "list")
			var validation *rules.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("want ValidationError, got %v", err)
			}
			if validation.Location != test.location || !strings.Contains(validation.Problem, test.problem) {
				t.Fatalf("got %q at %q; want a problem containing %q at %q", validation.Problem, validation.Location, test.problem, test.location)
			}
			if got != nil {
				t.Fatalf("failed parsing returned a partial list: %+v", got)
			}
		})
	}
}
