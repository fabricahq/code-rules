// Validate release selection, annotated tags, and failure boundaries independently of Git.

package rules_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fabricahq/code-rules/internal/rules"
	"os"
	"strings"
	"testing"
)

// TestVersionSelectionFixtures checks deterministic highest-version and alias behavior.
func TestVersionSelectionFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/version-selection/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Constraint, AvailableGitTags string
		Expected                         struct {
			Code  rules.VersionSelectionKind
			Value rules.VersionSelection
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		// Exercise each independently authored tag listing and selection expectation.
		t.Run(test.ID, func(t *testing.T) {
			constraint, err := rules.ParseVersionConstraint(test.Constraint, "constraint")
			if err != nil {
				t.Fatal(err)
			}
			got, err := rules.SelectReleaseTag(test.AvailableGitTags, constraint)
			if test.Expected.Code != "" {
				var selection *rules.VersionSelectionError
				if !errors.As(err, &selection) || selection.Kind != test.Expected.Code || got != (rules.VersionSelection{}) {
					t.Fatalf("got %+v, %v; want %s", got, err, test.Expected.Code)
				}
				return
			}
			if err != nil || got != test.Expected.Value {
				t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Value)
			}
		})
	}
}

// TestVersionSelectionLimits checks exact tag-count bounds and an unusable zero constraint.
func TestVersionSelectionLimits(t *testing.T) {
	constraint, err := rules.ParseVersionConstraint(">= 1.0.0", "constraint")
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for i := range 20_000 {
		fmt.Fprintf(&text, "%s\trefs/tags/ordinary-%d\n", strings.Repeat("a", 40), i)
	}
	_, err = rules.SelectReleaseTag(text.String(), constraint)
	var selection *rules.VersionSelectionError
	if !errors.As(err, &selection) || selection.Kind != rules.VersionNotFound {
		t.Fatalf("exact limit: %v", err)
	}
	text.WriteString(strings.Repeat("a", 40) + "\trefs/tags/another\n")
	_, err = rules.SelectReleaseTag(text.String(), constraint)
	if !errors.As(err, &selection) || selection.Kind != rules.TagLimitExceeded {
		t.Fatalf("over limit: %v", err)
	}
	if _, err := rules.SelectReleaseTag("", rules.VersionConstraint{}); err == nil {
		t.Fatal("accepted zero constraint")
	}
	_, err = rules.SelectReleaseTag(strings.Repeat("\n", 8*1024*1024+1), constraint)
	if !errors.As(err, &selection) || selection.Kind != rules.TagLimitExceeded {
		t.Fatalf("byte limit: %v", err)
	}
}
