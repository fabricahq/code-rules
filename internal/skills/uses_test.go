// Test that ResolveUses installs every skill a selection uses, transitively, and rejects missing uses and cycles.

package skills_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/skills"
)

func TestResolveUsesInstallsEverySkillTheSelectionUses(t *testing.T) {
	included := func(usedBy ...string) skills.Installed {
		return skills.Installed{Included: true, UsedBy: append([]string{}, usedBy...)}
	}
	used := func(usedBy ...string) skills.Installed { return skills.Installed{UsedBy: usedBy} }
	tests := map[string]struct {
		included []string
		uses     map[string][]string
		want     map[string]skills.Installed
	}{
		"nothing included": {nil, map[string][]string{"skills/a": {"skills/b"}, "skills/b": nil}, map[string]skills.Installed{}},
		"a skill that uses none": {
			[]string{"skills/a"}, map[string][]string{"skills/a": nil, "skills/b": nil},
			map[string]skills.Installed{"skills/a": included()},
		},
		"transitive uses": {
			[]string{"skills/a"}, map[string][]string{"skills/a": {"skills/b"}, "skills/b": {"skills/c"}, "skills/c": nil, "skills/d": nil},
			map[string]skills.Installed{"skills/a": included(), "skills/b": used("skills/a"), "skills/c": used("skills/b")},
		},
		"a skill two others use": {
			[]string{"skills/a", "skills/b"}, map[string][]string{"skills/a": {"skills/c"}, "skills/b": {"skills/c"}, "skills/c": nil},
			map[string]skills.Installed{"skills/a": included(), "skills/b": included(), "skills/c": used("skills/a", "skills/b")},
		},
		"an included skill another uses": {
			[]string{"skills/a", "skills/b"}, map[string][]string{"skills/a": {"skills/b"}, "skills/b": nil},
			map[string]skills.Installed{"skills/a": included(), "skills/b": included("skills/a")},
		},
		"a diamond": {
			[]string{"skills/a"}, map[string][]string{"skills/a": {"skills/c", "skills/b"}, "skills/b": {"skills/d"}, "skills/c": {"skills/d"}, "skills/d": nil},
			map[string]skills.Installed{"skills/a": included(), "skills/b": used("skills/a"), "skills/c": used("skills/a"), "skills/d": used("skills/b", "skills/c")},
		},
		"a use listed twice": {
			[]string{"skills/a"}, map[string][]string{"skills/a": {"skills/b", "skills/b"}, "skills/b": nil},
			map[string]skills.Installed{"skills/a": included(), "skills/b": used("skills/a")},
		},
		"a cycle the selection doesn't reach": {
			[]string{"skills/a"}, map[string][]string{"skills/a": nil, "skills/b": {"skills/c"}, "skills/c": {"skills/b"}},
			map[string]skills.Installed{"skills/a": included()},
		},
		"a missing use the selection doesn't reach": {
			[]string{"skills/a"}, map[string][]string{"skills/a": nil, "skills/b": {"skills/gone"}},
			map[string]skills.Installed{"skills/a": included()},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := skills.ResolveUses(test.included, test.uses)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestResolveUsesRejectsAMissingUse(t *testing.T) {
	tests := map[string]struct {
		included []string
		uses     map[string][]string
		want     skills.MissingUseError
	}{
		"an included skill's use": {
			[]string{"skills/a"}, map[string][]string{"skills/a": {"skills/how"}},
			skills.MissingUseError{Skill: "skills/a", Missing: "skills/how", Via: "skills/a"},
		},
		"a used skill's use": {
			[]string{"skills/a"}, map[string][]string{"skills/a": {"skills/b"}, "skills/b": {"skills/how"}},
			skills.MissingUseError{Skill: "skills/b", Missing: "skills/how", Via: "skills/a"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := skills.ResolveUses(test.included, test.uses)
			var missing *skills.MissingUseError
			if !errors.As(err, &missing) || *missing != test.want {
				t.Fatalf("got %v, want %#v", err, test.want)
			}
		})
	}
}

func TestResolveUsesRejectsSkillsThatUseEachOtherInACycle(t *testing.T) {
	tests := map[string]struct {
		included []string
		uses     map[string][]string
		want     []string
	}{
		"a skill that uses itself": {[]string{"skills/a"}, map[string][]string{"skills/a": {"skills/a"}}, []string{"skills/a", "skills/a"}},
		"two skills": {
			[]string{"skills/a"}, map[string][]string{"skills/a": {"skills/b"}, "skills/b": {"skills/a"}},
			[]string{"skills/a", "skills/b", "skills/a"},
		},
		"a cycle below the included skill": {
			[]string{"skills/a"}, map[string][]string{"skills/a": {"skills/b"}, "skills/b": {"skills/c"}, "skills/c": {"skills/b"}},
			[]string{"skills/b", "skills/c", "skills/b"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := skills.ResolveUses(test.included, test.uses)
			var cycle *skills.UseCycleError
			if !errors.As(err, &cycle) || !reflect.DeepEqual(cycle.Cycle, test.want) {
				t.Fatalf("got %v, want cycle %v", err, test.want)
			}
		})
	}
}

func TestResolveUsesLeavesItsInputsUnchanged(t *testing.T) {
	uses := map[string][]string{"skills/a": {"skills/c", "skills/b"}, "skills/b": nil, "skills/c": nil}
	if _, err := skills.ResolveUses([]string{"skills/a"}, uses); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(uses["skills/a"], []string{"skills/c", "skills/b"}) {
		t.Fatalf("uses changed: %v", uses)
	}
}
