// Check that planned library releases are records the release record parser accepts.

package library

import (
	"encoding/json"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// recordFor returns the release record a plan describes, with no library-wide files.
func recordFor(plan releasePlan) rules.ReleaseRecord {
	return rules.ReleaseRecord{Release: plan.release, Rules: plan.versions, Changes: plan.changes, Retired: plan.retired, LibraryFiles: []string{}}
}

// TestPlan_FirstLibraryReleaseAddsEveryRuleWithASummary gives each rule "Add the rule.", since the record requires a summary.
func TestPlan_FirstLibraryReleaseAddsEveryRuleWithASummary(t *testing.T) {
	plan, err := libraryChanges{history: releaseHistory{}, current: []string{"practices/testing/a", "practices/testing/b"}}.plan()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"practices/testing/a", "practices/testing/b"} {
		if change := plan.changes[id]; change.Change != rules.ChangeNew || change.Summary != "Add the rule." || plan.versions[id] != rules.FirstRuleVersion {
			t.Fatalf("%s: %+v at %s", id, change, plan.versions[id])
		}
	}
	// JSON is YAML, so the parser reads the encoded record as it would a tag's.
	encoded, err := json.Marshal(recordFor(plan))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rules.ParseReleaseRecord(encoded, "release/1"); err != nil {
		t.Fatalf("the parser rejects the planned first library release: %v\n%s", err, encoded)
	}
}

// TestPlan_LaterLibraryReleaseIsAValidRecord joins summaries and advances versions consistently with the parser.
func TestPlan_LaterLibraryReleaseIsAValidRecord(t *testing.T) {
	latest := &publishedRelease{number: 1, record: rules.ReleaseRecord{Release: 1, Rules: map[string]rules.RuleVersion{"practices/testing/a": rules.FirstRuleVersion, "practices/testing/b": rules.FirstRuleVersion}}}
	changes := libraryChanges{
		history: releaseHistory{latest: latest},
		current: []string{"practices/testing/a", "practices/testing/c"},
		pending: []pendingNote{
			{path: "changes/one.yaml", note: rules.ChangeNote{Summary: "Fix a typo.", Rules: map[string]rules.NoteChange{"practices/testing/a": {Change: rules.ChangePatch}}}},
			{path: "changes/two.yaml", note: rules.ChangeNote{Summary: "Replace b with c.", Rules: map[string]rules.NoteChange{"practices/testing/a": {Change: rules.ChangeMajor}, "practices/testing/b": {Change: rules.ChangeRetired, ReplacedBy: "practices/testing/c"}, "practices/testing/c": {Change: rules.ChangeNew}}}},
		},
	}
	plan, err := changes.plan()
	if err != nil {
		t.Fatal(err)
	}
	if a := plan.changes["practices/testing/a"]; a.Change != rules.ChangeMajor || a.Summary != "Fix a typo.\nReplace b with c." || plan.versions["practices/testing/a"] != (rules.RuleVersion{Major: 2}) {
		t.Fatalf("%+v at %s", a, plan.versions["practices/testing/a"])
	}
	encoded, err := json.Marshal(recordFor(plan))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rules.ParseReleaseRecord(encoded, "release/2"); err != nil {
		t.Fatalf("the parser rejects the planned library release: %v\n%s", err, encoded)
	}
}
