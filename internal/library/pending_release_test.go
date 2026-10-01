// Check that planned library releases are records the release record parser accepts.

package library

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestPlan_FirstLibraryReleaseAddsEveryRuleWithASummary gives each rule "Add the rule.", since the record requires a summary.
func TestPlan_FirstLibraryReleaseAddsEveryRuleWithASummary(t *testing.T) {
	plan, err := libraryChanges{history: releaseHistory{}, current: []string{"practices/testing/a", "practices/testing/b"}}.plan()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"practices/testing/a", "practices/testing/b"} {
		if change := plan.changes[id]; change.Change != coderules.ChangeNew || !slices.Equal(change.Summaries, []string{"Add the rule."}) || plan.versions[id] != coderules.FirstRuleVersion {
			t.Fatalf("%s: %+v at %s", id, change, plan.versions[id])
		}
	}
	// The record is encoded as code-rules library release writes it into a tag.
	encoded, err := encodeReleaseRecord(plan.record(nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coderules.ParseReleaseRecord(encoded, "release/1"); err != nil {
		t.Fatalf("the parser rejects the planned first library release: %v\n%s", err, encoded)
	}
}

// TestPendingRule_NamesVersionsAsTheReleaseRecordDoes gives changed rules from and to, and a retired rule its
// lastVersion, with every note's summary, in JSON.
func TestPendingRule_NamesVersionsAsTheReleaseRecordDoes(t *testing.T) {
	from, to := coderules.FirstRuleVersion, coderules.RuleVersion{Major: 1, Minor: 1}
	record := coderules.ReleaseRecord{Release: 2, Rules: map[string]coderules.RuleVersion{"practices/testing/a": to, "practices/testing/c": coderules.FirstRuleVersion},
		Changes: map[string]coderules.RecordedChange{"practices/testing/a": {Change: coderules.ChangeMinor, From: &from, Summaries: []string{"Add an example.", "Fix a typo."}}, "practices/testing/c": {Change: coderules.ChangeNew, Summaries: []string{"Add c."}}},
		Retired: map[string]coderules.RetiredRule{"practices/testing/b": {LastVersion: from, ReplacedBy: "practices/testing/c", Summaries: []string{"Replaced by c."}}}}
	encoded, err := json.Marshal(releaseRules(record))
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"id":"practices/testing/a","change":"minor","from":"1.0.0","to":"1.1.0","summaries":["Add an example.","Fix a typo."]},` +
		`{"id":"practices/testing/b","change":"retired","lastVersion":"1.0.0","replacedBy":"practices/testing/c","summaries":["Replaced by c."]},` +
		`{"id":"practices/testing/c","change":"new","to":"1.0.0","summaries":["Add c."]}]`
	if string(encoded) != want {
		t.Fatalf("got %s\nwant %s", encoded, want)
	}
}

// TestPlan_LaterLibraryReleaseIsAValidRecord joins summaries and advances versions consistently with the parser.
func TestPlan_LaterLibraryReleaseIsAValidRecord(t *testing.T) {
	latest := &publishedRelease{number: 1, record: coderules.ReleaseRecord{Release: 1, Rules: map[string]coderules.RuleVersion{"practices/testing/a": coderules.FirstRuleVersion, "practices/testing/b": coderules.FirstRuleVersion}}}
	changes := libraryChanges{
		history: releaseHistory{latest: latest},
		current: []string{"practices/testing/a", "practices/testing/c"},
		pending: []pendingNote{
			{path: "changes/one.yaml", note: rules.ChangeNote{Summary: "Fix a typo.", Rules: map[string]rules.NoteChange{"practices/testing/a": {Change: coderules.ChangePatch}}}},
			{path: "changes/two.yaml", note: rules.ChangeNote{Summary: "Replace b with c.", Rules: map[string]rules.NoteChange{"practices/testing/a": {Change: coderules.ChangeMajor}, "practices/testing/b": {Change: coderules.ChangeRetired, ReplacedBy: "practices/testing/c"}, "practices/testing/c": {Change: coderules.ChangeNew}}}},
		},
	}
	plan, err := changes.plan()
	if err != nil {
		t.Fatal(err)
	}
	if a := plan.changes["practices/testing/a"]; a.Change != coderules.ChangeMajor || !slices.Equal(a.Summaries, []string{"Fix a typo.", "Replace b with c."}) || plan.versions["practices/testing/a"] != (coderules.RuleVersion{Major: 2}) {
		t.Fatalf("%+v at %s", a, plan.versions["practices/testing/a"])
	}
	encoded, err := encodeReleaseRecord(plan.record(nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coderules.ParseReleaseRecord(encoded, "release/2"); err != nil {
		t.Fatalf("the parser rejects the planned library release: %v\n%s", err, encoded)
	}
}
