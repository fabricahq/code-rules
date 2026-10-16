// Verify rule selector parsing, matching, exclusion lookup, and group selection comparison and JSON form.

package rules_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fabricahq/syllabus/internal/rules"
)

// TestParseRuleSelection_ClassifiesEachSelector splits rules.include into the groups it imports in full and the
// individually selected rules, and rejects each malformed selector at its index with a message naming the fix.
func TestParseRuleSelection_ClassifiesEachSelector(t *testing.T) {
	for name, test := range map[string]struct {
		input  string
		groups rules.GroupSelection
		ids    []string
		// err is the whole message, or empty when the input is valid or, with rejected, fails ID validation.
		err      string
		rejected bool
	}{
		"everything":            {input: `["*"]`, groups: rules.GroupSelection{Pattern: "*", Groups: []string{}}, ids: []string{}},
		"every practice":        {input: `["practices/*"]`, groups: rules.GroupSelection{Pattern: "practices/*", Groups: []string{}}, ids: []string{}},
		"one group":             {input: `["techs/go/*"]`, groups: rules.GroupSelection{Groups: []string{"techs/go"}}, ids: []string{}},
		"one rule":              {input: `["techs/go/wrap-errors"]`, groups: rules.GroupSelection{Groups: []string{}}, ids: []string{"techs/go/wrap-errors"}},
		"nested rule":           {input: `["techs/go/errors/wrap"]`, groups: rules.GroupSelection{Groups: []string{}}, ids: []string{"techs/go/errors/wrap"}},
		"wildcards combine":     {input: `["techs/*", "practices/*"]`, groups: rules.GroupSelection{Pattern: "*", Groups: []string{}}, ids: []string{}},
		"mixed and sorted":      {input: `["techs/rust/*", "practices/testing/b", "techs/*", "practices/errors/*", "practices/testing/a"]`, groups: rules.GroupSelection{Pattern: "techs/*", Groups: []string{"practices/errors", "techs/rust"}}, ids: []string{"practices/testing/a", "practices/testing/b"}},
		"empty":                 {input: `[]`, groups: rules.GroupSelection{Groups: []string{}}, ids: []string{}},
		"not an array":          {input: `"*"`, err: `include: expected an array of selectors, such as ["techs/go/*", "practices/testing/verify-retry-limits"]`},
		"null":                  {input: `null`, err: `include: expected an array of selectors, such as ["techs/go/*", "practices/testing/verify-retry-limits"]`},
		"duplicate":             {input: `["techs/go/*", "techs/go/*"]`, err: "include: duplicate entries"},
		"inner star":            {input: `["techs/*/wrap-errors"]`, err: `include[0]: invalid selector "techs/*/wrap-errors": * is allowed only at the end, after a slash, such as techs/go/*, or alone as "*"`},
		"partial star":          {input: `["techs/go/wrap*"]`, err: `include[0]: invalid selector "techs/go/wrap*": * is allowed only at the end, after a slash, such as techs/go/*, or alone as "*"`},
		"double star":           {input: `["**"]`, err: `include[0]: invalid selector "**": * is allowed only at the end, after a slash, such as techs/go/*, or alone as "*"`},
		"bare group":            {input: `["techs/*", "techs/go"]`, err: `include[1]: "techs/go" is a group ID; select the whole group with techs/go/*`},
		"directory in a group":  {input: `["techs/go/errors/*"]`, err: `include[0]: invalid selector "techs/go/errors/*": a selector ending in /* selects a group, such as techs/go/*, or every group of one type, techs/* or practices/*`},
		"unknown root":          {input: `["tools/*"]`, rejected: true},
		"unknown root for rule": {input: `["tools/go/a"]`, rejected: true},
	} {
		t.Run(name, func(t *testing.T) {
			groups, ids, err := rules.ParseRuleSelection(json.RawMessage(test.input), "include")
			if test.rejected {
				if err == nil {
					t.Fatalf("accepted %s as %+v and %v", test.input, groups, ids)
				}
				return
			}
			if test.err != "" {
				if err == nil || err.Error() != test.err {
					t.Fatalf("got %v, want %s", err, test.err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(groups, test.groups) || !reflect.DeepEqual(ids, test.ids) {
				t.Fatalf("got %+v, %v, %v; want %+v, %v", groups, ids, err, test.groups, test.ids)
			}
		})
	}
}

// TestSelectorMatches_MatchesRulesWithinTheSelectorsScope matches a rule ID against itself, its group's selector,
// its type's selector, and "*", and refuses neighboring groups that share a name prefix.
func TestSelectorMatches_MatchesRulesWithinTheSelectorsScope(t *testing.T) {
	for _, test := range []struct {
		selector, id string
		want         bool
	}{
		{"*", "techs/go/wrap-errors", true},
		{"techs/*", "techs/go/wrap-errors", true},
		{"techs/*", "practices/testing/a", false},
		{"practices/*", "practices/testing/nested/a", true},
		{"techs/go/*", "techs/go/wrap-errors", true},
		{"techs/go/*", "techs/go/errors/wrap", true},
		{"techs/go/*", "techs/golang/wrap-errors", false},
		{"techs/go/wrap-errors", "techs/go/wrap-errors", true},
		{"techs/go/wrap-errors", "techs/go/wrap-errors-twice", false},
	} {
		if got := rules.SelectorMatches(test.selector, test.id); got != test.want {
			t.Errorf("SelectorMatches(%q, %q) = %v, want %v", test.selector, test.id, got, test.want)
		}
	}
}

// TestSource_Exclusion_PrefersTheNarrowestEntry returns the rule's own entry before its group's selector, its type's
// selector, and "*", and reports false for a rule no entry matches.
func TestSource_Exclusion_PrefersTheNarrowestEntry(t *testing.T) {
	source := rules.Source{Exclude: map[string]rules.Exclusion{
		"techs/go/a":   {Reason: "rule"},
		"techs/go/*":   {Reason: "group"},
		"practices/*":  {Reason: "practices"},
		"techs/rust/*": {Reason: "rust"},
	}}
	for id, want := range map[string]string{
		"techs/go/a":          "rule",
		"techs/go/b":          "group",
		"techs/go/nested/c":   "group",
		"practices/testing/a": "practices",
		"techs/rust/a":        "rust",
		"techs/python/a":      "",
	} {
		exclusion, ok := source.Exclusion(id)
		if ok != (want != "") || exclusion.Reason != want {
			t.Errorf("Exclusion(%q) = %+v, %v; want %q", id, exclusion, ok, want)
		}
	}
	source.Exclude["*"] = rules.Exclusion{Reason: "all"}
	if exclusion, ok := source.Exclusion("techs/python/a"); !ok || exclusion.Reason != "all" {
		t.Errorf("Exclusion with * = %+v, %v", exclusion, ok)
	}
}

// TestGroupSelection_Covers_JudgesByWildcardsAndListedGroups judges coverage by wildcards and listed groups, so a narrower or different selection is
// not covered.
func TestGroupSelection_Covers_JudgesByWildcardsAndListedGroups(t *testing.T) {
	for _, test := range []struct {
		selection, other rules.GroupSelection
		want             bool
	}{
		{rules.GroupSelection{Pattern: "*"}, rules.GroupSelection{Pattern: "techs/*", Groups: []string{"practices/testing"}}, true},
		{rules.GroupSelection{Pattern: "techs/*"}, rules.GroupSelection{Pattern: "*"}, false},
		{rules.GroupSelection{Pattern: "techs/*"}, rules.GroupSelection{Groups: []string{"techs/go"}}, true},
		{rules.GroupSelection{Pattern: "techs/*"}, rules.GroupSelection{Groups: []string{"practices/testing"}}, false},
		{rules.GroupSelection{Groups: []string{"techs/go", "techs/rust"}}, rules.GroupSelection{Groups: []string{"techs/go"}}, true},
		{rules.GroupSelection{Groups: []string{"techs/go"}}, rules.GroupSelection{Pattern: "techs/*"}, false},
		{rules.GroupSelection{Groups: []string{}}, rules.GroupSelection{Groups: []string{}}, true},
	} {
		if got := test.selection.Covers(test.other); got != test.want {
			t.Errorf("%+v.Covers(%+v) = %v, want %v", test.selection, test.other, got, test.want)
		}
	}
}

// TestGroupSelection_JSONRoundTrips writes a lone wildcard as a string and any other selection as a sorted array,
// and reads each back into an equal selection.
func TestGroupSelection_JSONRoundTrips(t *testing.T) {
	for want, selection := range map[string]rules.GroupSelection{
		`"*"`:                              {Pattern: "*", Groups: []string{}},
		`["practices/testing","techs/*"]`:  {Pattern: "techs/*", Groups: []string{"practices/testing"}},
		`["practices/testing","techs/go"]`: {Groups: []string{"practices/testing", "techs/go"}},
		`[]`:                               {Groups: []string{}},
	} {
		encoded, err := json.Marshal(selection)
		if err != nil || string(encoded) != want {
			t.Fatalf("encoded %+v as %s, %v; want %s", selection, encoded, err, want)
		}
		var decoded rules.GroupSelection
		if err := json.Unmarshal(encoded, &decoded); err != nil || !decoded.Equal(selection) {
			t.Fatalf("decoded %s as %+v, %v", encoded, decoded, err)
		}
	}
}
