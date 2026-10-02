// Check which rule, if any, owns a library path, and that every loadable rule path has a valid rule ID.

package librarytree_test

import (
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/librarytree"
)

// TestVersionedRule_NamesTheRuleThatOwnsAFile covers rule files, owned assets, and library-wide files.
func TestVersionedRule_NamesTheRuleThatOwnsAFile(t *testing.T) {
	for file, want := range map[string]string{
		"techs/go/errors.md":                    "techs/go/errors",
		"techs/go/nested/errors.md":             "techs/go/nested/errors",
		"techs/go/assets/errors/diagram.png":    "techs/go/errors",
		"techs/go/assets/errors/deep/notes.md":  "techs/go/errors",
		"techs/go/nested/assets/errors/data.md": "techs/go/nested/errors",
		"techs/go/_group.yaml":                  "",
		"techs/go/README.md":                    "",
		"techs/go/assets/loose.png":             "",
		"assets/shared.md":                      "",
		"rule-library.yaml":                     "",
		"LICENSE.md":                            "",
		"changes/one.yaml":                      "",
		"techs/go/LICENSE.md":                   "",
	} {
		got, ok := librarytree.VersionedRule(file)
		if got != want || ok != (want != "") {
			t.Errorf("VersionedRule(%q) = %q, %v; want %q", file, got, ok, want)
		}
	}
}

// TestValidateRuleID_AcceptsEveryLoadableRulePath keeps change notes able to name every rule the loader accepts.
func TestValidateRuleID_AcceptsEveryLoadableRulePath(t *testing.T) {
	for _, path := range []string{"practices/testing/verify-retry-limits.md", "practices/testing/example.md.md", "techs/react/hooks/test-in-isolation.md"} {
		if _, err := librarytree.GroupFromPath(path, "path"); err != nil {
			t.Fatalf("%s is not a loadable rule path: %v", path, err)
		}
		if err := librarytree.ValidateRuleID(strings.TrimSuffix(path, ".md"), "id"); err != nil {
			t.Errorf("rule %s has an ID change notes reject: %v", path, err)
		}
	}
}
