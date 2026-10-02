// Verify that adding pins and exclusions to a source keeps authored presentation and refuses invalid edits.

package rules_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestEditConfigurationSource_AddsPinsAndExclusionsKeepingPresentation writes quoted pins into a new pins map
// and an exclusion into an existing flow-style map, keeping comments, order, and the other source unchanged.
func TestEditConfigurationSource_AddsPinsAndExclusionsKeepingPresentation(t *testing.T) {
	input := []byte(`# Project rules
schemaVersion: 1
sources:
  other:
    repository: https://example.invalid/other.git # untouched
    groups: '*'
  team:
    repository: https://example.invalid/team.git # the team library
    groups:
      - techs/go
    exclude: {techs/go/old: {reason: Not used here.}}
`)
	version, err := coderules.ParseRuleVersion("1.3.0", "version")
	if err != nil {
		t.Fatal(err)
	}
	out, err := rules.EditConfigurationSource(input, "team", rules.SourceEdit{
		Pins:    map[string]rules.Pin{"techs/go/b": {Version: version, Reason: "Waiting on #45."}, "techs/go/a": {Version: version, Reason: "yes"}},
		Exclude: map[string]rules.Exclusion{"techs/go/new": {Reason: "Covered by our own rule."}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"# Project rules\n", "https://example.invalid/other.git # untouched\n", "https://example.invalid/team.git # the team library\n",
		"    pins:\n      techs/go/a:\n        version: \"1.3.0\"\n        reason: 'yes'\n      techs/go/b:\n        version: \"1.3.0\"\n        reason: 'Waiting on #45.'\n",
		"    exclude:\n      techs/go/old: {reason: Not used here.}\n      techs/go/new:\n        reason: Covered by our own rule.\n"} {
		if !strings.Contains(string(out), text) {
			t.Fatalf("missing %q:\n%s", text, out)
		}
	}
	if strings.Index(string(out), "  other:") > strings.Index(string(out), "  team:") {
		t.Fatalf("reordered sources:\n%s", out)
	}
	config, err := rules.ParseConfigurationYAML(out)
	if err != nil {
		t.Fatal(err)
	}
	team := config.Sources[1]
	if team.Pins["techs/go/a"].Reason != "yes" || team.Pins["techs/go/b"].Version != version || team.Exclude["techs/go/new"].Reason != "Covered by our own rule." || len(team.Exclude) != 2 || len(config.Sources[0].Pins) != 0 {
		t.Fatalf("edited configuration %+v", config.Sources)
	}
}

// TestEditConfigurationSource_RemovesPinsAndAnEmptiedPinsMap removes one pin and keeps the other, then removes
// the last one with the pins map, keeping the rest of the source.
func TestEditConfigurationSource_RemovesPinsAndAnEmptiedPinsMap(t *testing.T) {
	input := []byte("schemaVersion: 1\nsources:\n  team:\n    repository: https://example.invalid/team.git # the team library\n    groups: '*'\n    pins:\n      techs/go/a:\n        version: \"1.0.0\"\n        reason: Kept.\n      techs/go/b:\n        version: \"2.0.0\"\n        reason: Also kept.\n")
	one, err := rules.EditConfigurationSource(input, "team", rules.SourceEdit{Unpin: []string{"techs/go/a"}})
	if err != nil || strings.Contains(string(one), "techs/go/a") || !strings.Contains(string(one), "    pins:\n      techs/go/b:\n") {
		t.Fatalf("%v:\n%s", err, one)
	}
	none, err := rules.EditConfigurationSource(one, "team", rules.SourceEdit{Unpin: []string{"techs/go/b"}, Exclude: map[string]rules.Exclusion{"techs/go/b": {Reason: "Forked."}}})
	if err != nil || strings.Contains(string(none), "pins") || !strings.Contains(string(none), "# the team library") || !strings.Contains(string(none), "    exclude:\n      techs/go/b:\n        reason: Forked.\n") {
		t.Fatalf("%v:\n%s", err, none)
	}
}

// TestEditConfigurationSource_RefusesEditsItCantApply leaves existing entries alone and rejects results that
// wouldn't validate, returning no bytes.
func TestEditConfigurationSource_RefusesEditsItCantApply(t *testing.T) {
	input := []byte("schemaVersion: 1\nsources:\n  team:\n    repository: https://example.invalid/team.git\n    groups: '*'\n    pins:\n      techs/go/a:\n        version: \"1.0.0\"\n        reason: Kept.\n  by-ref:\n    repository: https://example.invalid/ref.git\n    groups: '*'\n    ref: release/2\n")
	version, err := coderules.ParseRuleVersion("2.0.0", "version")
	if err != nil {
		t.Fatal(err)
	}
	for name, test := range map[string]struct {
		alias string
		edit  rules.SourceEdit
		where string
	}{
		"existing pin":          {"team", rules.SourceEdit{Pins: map[string]rules.Pin{"techs/go/a": {Version: version, Reason: "Newer."}}}, "sources.team.pins.techs/go/a"},
		"missing source":        {"absent", rules.SourceEdit{Exclude: map[string]rules.Exclusion{"techs/go/a": {Reason: "No."}}}, "sources.absent"},
		"pin with ref":          {"by-ref", rules.SourceEdit{Pins: map[string]rules.Pin{"techs/go/a": {Version: version, Reason: "No."}}}, "sources.by-ref"},
		"blank reason":          {"team", rules.SourceEdit{Exclude: map[string]rules.Exclusion{"techs/go/b": {Reason: " "}}}, "sources.team.exclude.techs/go/b"},
		"missing pin to remove": {"team", rules.SourceEdit{Unpin: []string{"techs/go/b"}}, "sources.team.pins.techs/go/b"},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := rules.EditConfigurationSource(input, test.alias, test.edit)
			var validation errs.ValidationError
			if out != nil || !errors.As(err, &validation) || !strings.HasPrefix(validation.ValidationLocation(), test.where) {
				t.Fatalf("got %s, %v", out, err)
			}
		})
	}
}

// TestEditConfigurationSource_SetsTheBasedOnVersionOfAReplacement adds basedOn to a replacement that has none and
// replaces another's, writing the versions quoted and keeping comments, and refuses a rule the source doesn't
// replace.
func TestEditConfigurationSource_SetsTheBasedOnVersionOfAReplacement(t *testing.T) {
	input := []byte(`schemaVersion: 1
sources:
  team:
    repository: https://example.invalid/team.git
    groups: [techs/go]
    exclude:
      techs/go/a: # ours
        reason: Stricter.
        replacedBy: local/techs/go/a.md
      techs/go/b:
        reason: Ours.
        replacedBy: local/techs/go/b.md
        basedOn: "1.0.0" # forked
      techs/go/c:
        reason: Not used.
`)
	version := func(text string) coderules.RuleVersion {
		parsed, err := coderules.ParseRuleVersion(text, "version")
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	out, err := rules.EditConfigurationSource(input, "team", rules.SourceEdit{BasedOn: map[string]coderules.RuleVersion{"techs/go/a": version("1.2.0"), "techs/go/b": version("2.0.0")}})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"      techs/go/a: # ours\n        reason: Stricter.\n        replacedBy: local/techs/go/a.md\n        basedOn: \"1.2.0\"\n", "        basedOn: \"2.0.0\" # forked\n"} {
		if !strings.Contains(string(out), text) {
			t.Fatalf("missing %q:\n%s", text, out)
		}
	}
	config, err := rules.ParseConfigurationYAML(out)
	if err != nil {
		t.Fatal(err)
	}
	if based := config.Sources[0].Exclude["techs/go/b"].BasedOn; based == nil || *based != version("2.0.0") {
		t.Fatalf("basedOn %v", based)
	}
	for _, id := range []string{"techs/go/c", "techs/go/missing"} {
		var invalid errs.ValidationError
		if _, err := rules.EditConfigurationSource(input, "team", rules.SourceEdit{BasedOn: map[string]coderules.RuleVersion{id: version("1.0.0")}}); !errors.As(err, &invalid) || invalid.ValidationLocation() != "sources.team.exclude."+id {
			t.Errorf("%s: got %v", id, err)
		}
	}
}
