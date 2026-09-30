// Exercise snapshot integrity and offline version checks through the encoding and decoding boundary.

package project

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Commits of the fixture's two library releases.
var (
	releaseOne = strings.Repeat("1", 40)
	releaseTwo = strings.Repeat("2", 40)
)

// snapshotConfig parses one source named team from its JSON fields.
func snapshotConfig(t *testing.T, fields string) rules.Configuration {
	t.Helper()
	config, err := rules.ParseConfiguration([]byte(`{"schemaVersion":1,"sources":{"team":{"repository":"https://github.com/acme/rules",` + fields + `}}}`))
	if err != nil {
		t.Fatal(err)
	}
	return config
}

// pinVersions returns each configured pin's version, as a snapshot records it.
func pinVersions(pins map[string]rules.Pin) map[string]rules.RuleVersion {
	versions := map[string]rules.RuleVersion{}
	for id, pin := range pins {
		versions[id] = pin.Version
	}
	return versions
}

// snapshotFixture supplies a source that follows rule versions: errors at 1.1.0 from library release 2, which
// supplies the library-wide files, and naming at 1.0.0 from library release 1.
// Files include original binary and CRLF bytes.
func snapshotFixture(t *testing.T) (rules.Configuration, map[string]snapshot) {
	t.Helper()
	config := snapshotConfig(t, `"groups":["techs/go"]`)
	one, two := rules.RuleVersion{Major: 1}, rules.RuleVersion{Major: 1, Minor: 1}
	return config, map[string]snapshot{"team": {Repository: config.Sources[0].Repository, Pins: map[string]rules.RuleVersion{}, RetiredRules: []string{}, Release: 2, Commit: releaseTwo, Groups: []string{"techs/go"}, Selection: config.Sources[0].Groups, RuleSelection: []string{},
		Rules: map[string]library.ImportedRule{"techs/go/errors": {Version: &two, Release: 2, Commit: releaseTwo}, "techs/go/naming": {Version: &one, Release: 1, Commit: releaseOne}},
		Files: map[string][]byte{
			"rule-library.yaml":    []byte(`{"formatVersion":1}`),
			"techs/go/_group.yaml": []byte(`{"name":"Go","description":"Go rules","whenToRead":"When editing Go."}`),
			"techs/go/errors.md":   []byte(projectRule),
			"techs/go/naming.md":   []byte(projectRule),
			"assets/image.bin":     {0, 255, 10, 128}, "LICENSE": []byte("Terms\r\nPreserved\r\n"), "empty.txt": {},
		}}}
}

// requireSync checks that decoding fails with a validation error that tells the user to sync.
func requireSync(t *testing.T, got map[string]snapshot, err error) {
	t.Helper()
	var validation *rules.ValidationError
	if got != nil || !errors.As(err, &validation) || !strings.HasPrefix(validation.Location, "vendor/team/_source.json") || !strings.Contains(validation.Problem, "run code-rules project sync") {
		t.Fatalf("got %v, %v; want a failure that asks for sync", got, err)
	}
}

// TestSnapshotsRoundTripOwnsExactBytes verifies deterministic records and independent ownership.
func TestSnapshotsRoundTripOwnsExactBytes(t *testing.T) {
	config, input := snapshotFixture(t)
	encoded, err := encodeSnapshots(config, input)
	if err != nil {
		t.Fatal(err)
	}
	again, err := encodeSnapshots(config, input)
	if err != nil || !reflect.DeepEqual(encoded, again) {
		t.Fatalf("unstable encoding: %v", err)
	}
	got, err := decodeSnapshots(config, encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, input) {
		t.Fatalf("round trip: %#v", got)
	}
	got["team"].Files["LICENSE"][0] = 'X'
	if input["team"].Files["LICENSE"][0] == 'X' || encoded["team/LICENSE"][0] == 'X' {
		t.Fatal("returned bytes alias input")
	}
	encoded["team/assets/image.bin"][0] = 9
	if input["team"].Files["assets/image.bin"][0] != 0 {
		t.Fatal("encoded bytes alias input")
	}
}

// TestSnapshotRecordFormat writes the documented format 2 fields, omitting pins, ref, and ruleSelection when empty.
func TestSnapshotRecordFormat(t *testing.T) {
	config, input := snapshotFixture(t)
	encoded, err := encodeSnapshots(config, input)
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]json.RawMessage
	if err := json.Unmarshal(encoded["team/_source.json"], &record); err != nil {
		t.Fatal(err)
	}
	for field, want := range map[string]string{
		"formatVersion": "2", "release": "2", "resolvedCommit": `"` + releaseTwo + `"`, "groupSelection": `["techs/go"]`, "groups": `["techs/go"]`,
		"rules": `{"techs/go/errors":{"version":"1.1.0","release":2,"commit":"` + releaseTwo + `"},"techs/go/naming":{"version":"1.0.0","release":1,"commit":"` + releaseOne + `"}}`,
	} {
		var compact strings.Builder
		for _, line := range strings.Split(string(record[field]), "\n") {
			compact.WriteString(strings.TrimSpace(line))
		}
		if got := strings.ReplaceAll(compact.String(), `": `, `":`); got != want {
			t.Errorf("%s = %s, want %s", field, got, want)
		}
	}
	for _, field := range []string{"pins", "exclude", "ref", "ruleSelection"} {
		if _, ok := record[field]; ok {
			t.Errorf("wrote empty %s", field)
		}
	}
}

// TestSnapshotRepositoryAddressChangeRequiresSync rejects stale authored provenance even for the same repository.
func TestSnapshotRepositoryAddressChangeRequiresSync(t *testing.T) {
	for _, address := range []string{"git@github.com:acme/rules.git", "https://github.com/acme/rules.git", "https://github.com/ACME/rules"} {
		t.Run(address, func(t *testing.T) {
			config, snapshots := snapshotFixture(t)
			vendor, err := encodeSnapshots(config, snapshots)
			if err != nil {
				t.Fatal(err)
			}
			config.Sources[0].Repository = address
			got, err := decodeSnapshots(config, vendor)
			requireSync(t, got, err)
		})
	}
}

// TestSnapshotChangedSelectionOrRefRequiresSync rejects configuration the record doesn't describe.
func TestSnapshotChangedSelectionOrRefRequiresSync(t *testing.T) {
	for name, fields := range map[string]string{
		"added group":            `"groups":["techs/go","techs/rust"]`,
		"wildcard":               `"groups":"*"`,
		"individual rule":        `"groups":["techs/go"],"rules":["practices/testing/limits"]`,
		"ref":                    `"groups":["techs/go"],"ref":"release/2"`,
		"pin to another version": `"groups":["techs/go"],"pins":{"techs/go/errors":{"version":"1.0.0","reason":"Not ready."}}`,
		"pin of a rule it lacks": `"groups":["techs/go"],"pins":{"techs/go/other":{"version":"1.0.0","reason":"Keep."}}`,
	} {
		t.Run(name, func(t *testing.T) {
			config, snapshots := snapshotFixture(t)
			vendor, err := encodeSnapshots(config, snapshots)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decodeSnapshots(snapshotConfig(t, fields), vendor)
			requireSync(t, got, err)
		})
	}
}

// TestSnapshotPinsThatMoveNothingNeedNoSync accepts a pin at the recorded version, a changed reason, and a removed pin.
func TestSnapshotPinsThatMoveNothingNeedNoSync(t *testing.T) {
	config, snapshots := snapshotFixture(t)
	pinned := snapshotConfig(t, `"groups":["techs/go"],"pins":{"techs/go/naming":{"version":"1.0.0","reason":"Waiting on #45."}}`)
	item := snapshots["team"]
	item.Pins = pinVersions(pinned.Sources[0].Pins)
	snapshots["team"] = item
	vendor, err := encodeSnapshots(pinned, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	for name, fields := range map[string]string{
		"same pin":       `"groups":["techs/go"],"pins":{"techs/go/naming":{"version":"1.0.0","reason":"Waiting on #45."}}`,
		"changed reason": `"groups":["techs/go"],"pins":{"techs/go/naming":{"version":"1.0.0","reason":"Still waiting."}}`,
		"added pin":      `"groups":["techs/go"],"pins":{"techs/go/naming":{"version":"1.0.0","reason":"Keep."},"techs/go/errors":{"version":"1.1.0","reason":"Keep."}}`,
		"removed pin":    `"groups":["techs/go"]`,
	} {
		if _, err := decodeSnapshots(snapshotConfig(t, fields), vendor); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	_ = config
}

// TestSnapshotRetiredEntriesRecordedAtSyncNeedNoSync accepts a pin or rules entry that named a retired rule then.
func TestSnapshotRetiredEntriesRecordedAtSyncNeedNoSync(t *testing.T) {
	config := snapshotConfig(t, `"groups":["techs/go"],"rules":["practices/testing/retired"],"pins":{"techs/go/gone":{"version":"1.0.0","reason":"Keep."}}`)
	_, snapshots := snapshotFixture(t)
	item := snapshots["team"]
	item.Pins, item.RuleSelection = pinVersions(config.Sources[0].Pins), config.Sources[0].Rules
	snapshots["team"] = item
	vendor, err := encodeSnapshots(config, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeSnapshots(config, vendor); err != nil {
		t.Fatal(err)
	}
}

// TestSnapshotExclusionOfAnImportedOrRetiredRuleNeedsNoSync accepts any exclusion naming a rule the snapshot imports
// or a rule its record lists as retired, however the exclusions changed since the record was written, because the
// record holds facts about the library, not the configuration's exclusions.
func TestSnapshotExclusionOfAnImportedOrRetiredRuleNeedsNoSync(t *testing.T) {
	config, snapshots := snapshotFixture(t)
	item := snapshots["team"]
	item.RetiredRules = []string{"techs/go/gone"}
	snapshots["team"] = item
	vendor, err := encodeSnapshots(config, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	for name, fields := range map[string]string{
		"imported and retired":   `"groups":["techs/go"],"exclude":{"techs/go/gone":{"reason":"Retired."},"techs/go/errors":{"reason":"Not for us."}}`,
		"retired only":           `"groups":["techs/go"],"exclude":{"techs/go/gone":{"reason":"Delete me."}}`,
		"new imported exclusion": `"groups":["techs/go"],"exclude":{"techs/go/naming":{"reason":"Not for us.","replacedBy":"local/techs/go/naming.md"}}`,
		"no exclusions":          `"groups":["techs/go"]`,
	} {
		if _, err := decodeSnapshots(snapshotConfig(t, fields), vendor); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestSnapshotExclusionOfAnUnknownRuleRequiresSync refuses an exclusion whose rule isn't imported and that sync
// didn't record, such as a typo, so the intended rule doesn't stay active unnoticed.
func TestSnapshotExclusionOfAnUnknownRuleRequiresSync(t *testing.T) {
	config, snapshots := snapshotFixture(t)
	vendor, err := encodeSnapshots(config, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeSnapshots(snapshotConfig(t, `"groups":["techs/go"],"exclude":{"techs/go/erorrs":{"reason":"Typo."}}`), vendor)
	var validation *rules.ValidationError
	if got != nil || !errors.As(err, &validation) || validation.Location != "sources.team.exclude.techs/go/erorrs" || !strings.Contains(validation.Problem, "names no rule the library supplies") || !strings.Contains(validation.Problem, "run code-rules project sync") {
		t.Fatalf("got %v, %v; want a failure naming the entry that asks for sync", got, err)
	}
}

// TestSnapshotCorruptionReturnsNoPartialResult covers malformed records and altered inventories.
func TestSnapshotCorruptionReturnsNoPartialResult(t *testing.T) {
	cases := []struct {
		name  string
		alter func(map[string][]byte)
	}{
		{"missing record", func(v map[string][]byte) { delete(v, "team/_source.json") }},
		{"missing content", func(v map[string][]byte) { delete(v, "team/LICENSE") }},
		{"missing older rule", func(v map[string][]byte) { delete(v, "team/techs/go/naming.md") }},
		{"changed bytes", func(v map[string][]byte) { v["team/LICENSE"] = []byte("Terms\nPreserved\n") }},
		{"extra content", func(v map[string][]byte) { v["team/untracked"] = []byte("x") }},
		{"removed source", func(v map[string][]byte) { v["retired/_source.json"] = []byte("{}") }},
		{"invalid UTF8", func(v map[string][]byte) { v["team/_source.json"] = []byte{255} }},
		{"null", func(v map[string][]byte) { v["team/_source.json"] = []byte("null") }},
		{"trailing JSON", func(v map[string][]byte) { v["team/_source.json"] = append(v["team/_source.json"], []byte("{}")...) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, s := snapshotFixture(t)
			v, e := encodeSnapshots(c, s)
			if e != nil {
				t.Fatal(e)
			}
			tc.alter(v)
			got, e := decodeSnapshots(c, v)
			var validation *rules.ValidationError
			if got != nil || !errors.As(e, &validation) {
				t.Fatalf("got %v, %v", got, e)
			}
		})
	}
}

// TestSnapshotRecordRelationships validates identity, versions, and selection rather than trusting hashes alone.
func TestSnapshotRecordRelationships(t *testing.T) {
	rule := func(r map[string]any, id string) map[string]any {
		return r["rules"].(map[string]any)[id].(map[string]any)
	}
	cases := []struct {
		name  string
		alter func(map[string]any)
	}{
		{"format 1", func(r map[string]any) { r["formatVersion"] = 1 }},
		{"unknown", func(r map[string]any) { r["future"] = true }},
		{"wrong case", func(r map[string]any) { r["Repository"] = r["repository"]; delete(r, "repository") }},
		{"null optional", func(r map[string]any) { r["groupSelection"] = nil }},
		{"missing rules", func(r map[string]any) { delete(r, "rules") }},
		{"missing release", func(r map[string]any) { delete(r, "release") }},
		{"zero release", func(r map[string]any) { r["release"] = 0 }},
		{"older release", func(r map[string]any) { r["release"] = 1 }},
		{"invalid commit", func(r map[string]any) { r["resolvedCommit"] = "main" }},
		{"changed ref", func(r map[string]any) { r["ref"] = "v2.0.0" }},
		{"invalid ref", func(r map[string]any) { r["ref"] = "refs/heads/main" }},
		{"changed selection", func(r map[string]any) { r["groupSelection"] = "*" }},
		{"missing group", func(r map[string]any) { r["groups"] = []string{} }},
		{"duplicate group", func(r map[string]any) { r["groups"] = []string{"techs/go", "techs/go"} }},
		{"rule without release", func(r map[string]any) { rule(r, "techs/go/errors")["release"] = nil }},
		{"unreleased rule without ref", func(r map[string]any) {
			rule(r, "techs/go/errors")["release"] = nil
			rule(r, "techs/go/errors")["version"] = nil
		}},
		{"rule with invalid version", func(r map[string]any) { rule(r, "techs/go/errors")["version"] = "1.1" }},
		{"rule with another commit", func(r map[string]any) { rule(r, "techs/go/errors")["commit"] = releaseOne }},
		{"rule with unknown field", func(r map[string]any) { rule(r, "techs/go/errors")["notes"] = "x" }},
		{"invalid rule ID", func(r map[string]any) { r["rules"].(map[string]any)["Go"] = rule(r, "techs/go/errors") }},
		{"bad hash", func(r map[string]any) { r["files"].(map[string]any)["LICENSE"] = "123" }},
		{"reserved", func(r map[string]any) { r["files"].(map[string]any)["_source.json"] = strings.Repeat("a", 64) }},
		{"traversal", func(r map[string]any) { r["files"].(map[string]any)["../outside"] = strings.Repeat("a", 64) }},
		{"missing manifest", func(r map[string]any) { delete(r["files"].(map[string]any), "rule-library.yaml") }},
		{"missing metadata", func(r map[string]any) { delete(r["files"].(map[string]any), "techs/go/_group.yaml") }},
		{"pin with a reason", func(r map[string]any) {
			r["pins"] = map[string]any{"techs/go/errors": map[string]any{"version": "1.1.0", "reason": "Keep."}}
		}},
		{"pin with an invalid version", func(r map[string]any) { r["pins"] = map[string]any{"techs/go/errors": "1.1"} }},
		{"pin of an invalid rule ID", func(r map[string]any) { r["pins"] = map[string]any{"Go": "1.1.0"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, s := snapshotFixture(t)
			v, e := encodeSnapshots(c, s)
			if e != nil {
				t.Fatal(e)
			}
			var record map[string]any
			if e = json.Unmarshal(v["team/_source.json"], &record); e != nil {
				t.Fatal(e)
			}
			tc.alter(record)
			v["team/_source.json"], e = json.Marshal(record)
			if e != nil {
				t.Fatal(e)
			}
			got, e := decodeSnapshots(c, v)
			if got != nil || e == nil {
				t.Fatalf("accepted corrupt record: %v", got)
			}
		})
	}
}

// TestSnapshotFormatOneIsUnsupported tells the user to delete vendor/ and sync, for both sync and offline reads.
func TestSnapshotFormatOneIsUnsupported(t *testing.T) {
	config, _ := snapshotFixture(t)
	vendor := map[string][]byte{"team/_source.json": []byte(`{"formatVersion":1,"repository":"https://github.com/acme/rules","ref":"v1.0.0","resolvedCommit":"` + releaseOne + `","groups":["techs/go"],"files":{}}`)}
	for name, read := range map[string]func() error{
		"offline": func() error { _, err := decodeSnapshots(config, vendor); return err },
		"sync":    func() error { _, err := recordedSnapshots(config, vendor); return err },
	} {
		var validation *rules.ValidationError
		if err := read(); !errors.As(err, &validation) || !strings.Contains(validation.Problem, "delete .code-rules/vendor/ and run code-rules project sync") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestRecordedSnapshotsReadRecordsWithoutCheckingConfiguration returns records for sync to compare.
func TestRecordedSnapshotsReadRecordsWithoutCheckingConfiguration(t *testing.T) {
	config, snapshots := snapshotFixture(t)
	vendor, err := encodeSnapshots(config, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	vendor["team/LICENSE"] = []byte("Edited copy, which sync replaces.")
	changed := snapshotConfig(t, `"groups":"*"`)
	recorded, err := recordedSnapshots(changed, vendor)
	if err != nil {
		t.Fatal(err)
	}
	want := snapshots["team"]
	want.Files = nil
	if !reflect.DeepEqual(recorded["team"], want) {
		t.Fatalf("got %+v, want %+v", recorded["team"], want)
	}
	if recorded, err := recordedSnapshots(changed, map[string][]byte{}); err != nil || len(recorded) != 0 {
		t.Fatalf("new source: %v, %v", recorded, err)
	}
}

// TestSnapshotRefChecks verifies what offline checks can about ref: the commit a SHA names, the library release
// a release tag names, and no library release for any other tag.
func TestSnapshotRefChecks(t *testing.T) {
	one := rules.RuleVersion{Major: 1}
	for _, test := range []struct {
		name, ref string
		release   int
		commit    string
		rules     map[string]library.ImportedRule
		ok        bool
	}{
		{"commit", releaseTwo, 0, releaseTwo, map[string]library.ImportedRule{"techs/go/errors": {Commit: releaseTwo}}, true},
		{"other commit", releaseOne, 0, releaseTwo, map[string]library.ImportedRule{"techs/go/errors": {Commit: releaseTwo}}, false},
		{"library release", "release/2", 2, releaseTwo, map[string]library.ImportedRule{"techs/go/errors": {Version: &one, Release: 1, Commit: releaseOne}}, true},
		{"another library release", "release/3", 2, releaseTwo, map[string]library.ImportedRule{"techs/go/errors": {Version: &one, Release: 1, Commit: releaseOne}}, false},
		{"rule from a later library release", "release/1", 1, releaseOne, map[string]library.ImportedRule{"techs/go/errors": {Version: &one, Release: 2, Commit: releaseTwo}}, false},
		{"unreleased rule from a library release", "release/2", 2, releaseTwo, map[string]library.ImportedRule{"techs/go/errors": {Commit: releaseTwo}}, false},
		{"other tag", "candidate", 0, releaseTwo, map[string]library.ImportedRule{"techs/go/errors": {Version: &one, Release: 1, Commit: releaseOne}}, true},
		{"other tag with a library release", "candidate", 2, releaseTwo, map[string]library.ImportedRule{"techs/go/errors": {Version: &one, Release: 1, Commit: releaseOne}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := snapshotConfig(t, `"groups":["techs/go"],"ref":"`+test.ref+`"`)
			files := map[string][]byte{"rule-library.yaml": []byte(`{"formatVersion":1}`), "techs/go/_group.yaml": []byte(`{}`)}
			item := snapshot{Repository: config.Sources[0].Repository, Pins: map[string]rules.RuleVersion{}, Ref: gitRef(t, test.ref), Release: test.release, Commit: test.commit, Groups: []string{"techs/go"}, Selection: config.Sources[0].Groups, RuleSelection: []string{}, Rules: test.rules, Files: files}
			_, err := encodeSnapshots(config, map[string]snapshot{"team": item})
			if (err == nil) != test.ok {
				t.Fatalf("got %v, want ok %v", err, test.ok)
			}
		})
	}
}

// TestSnapshotRefWrittenAnotherWayNeedsNoSync accepts a record of release/2 for a source that now writes the same ref
// as refs/tags/release/2, and still asks for sync when the ref names another library release.
func TestSnapshotRefWrittenAnotherWayNeedsNoSync(t *testing.T) {
	recorded := snapshotConfig(t, `"groups":["techs/go"],"ref":"release/2"`)
	_, snapshots := snapshotFixture(t)
	item := snapshots["team"]
	item.Ref = gitRef(t, "release/2")
	vendor, err := encodeSnapshots(recorded, map[string]snapshot{"team": item})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeSnapshots(snapshotConfig(t, `"groups":["techs/go"],"ref":"refs/tags/release/2"`), vendor); err != nil {
		t.Fatal(err)
	}
	got, err := decodeSnapshots(snapshotConfig(t, `"groups":["techs/go"],"ref":"refs/tags/release/3"`), vendor)
	requireSync(t, got, err)
}

// TestSnapshotEmptySources encodes and decodes a local-only project and rejects a removed source's record.
func TestSnapshotEmptySources(t *testing.T) {
	c, s := snapshotFixture(t)
	v, e := encodeSnapshots(c, s)
	if e != nil {
		t.Fatal(e)
	}
	empty := rules.Configuration{Sources: []rules.Source{}}
	encoded, e := encodeSnapshots(empty, nil)
	if e != nil || len(encoded) != 0 {
		t.Fatalf("empty encode: %v", e)
	}
	decoded, e := decodeSnapshots(empty, encoded)
	if e != nil || len(decoded) != 0 {
		t.Fatalf("empty decode: %v", e)
	}
	if got, e := decodeSnapshots(empty, v); e == nil || got != nil {
		t.Fatal("accepted retired source")
	}
}

// TestSnapshotPortablePaths rejects ambiguous destination paths before encoding.
func TestSnapshotPortablePaths(t *testing.T) {
	for _, path := range []string{"../outside", "/absolute", "a\\b", "a:b", "a/./b", "empty.txt/child", "ASSETS/image.bin", "_source.json", "_SOURCE.json", "_source.json/child"} {
		t.Run(path, func(t *testing.T) {
			c, s := snapshotFixture(t)
			s["team"].Files[path] = []byte("x")
			if got, e := encodeSnapshots(c, s); e == nil || got != nil {
				t.Fatalf("accepted %s", path)
			}
		})
	}
}

// TestSnapshotsDoNotReturnEarlierSourcesOnFailure verifies atomic results across source inventories.
func TestSnapshotsDoNotReturnEarlierSourcesOnFailure(t *testing.T) {
	config, snapshots := snapshotFixture(t)
	source := config.Sources[0]
	source.Name = "second"
	config.Sources = append(config.Sources, source)
	snapshots["second"] = snapshots["team"]
	vendor, err := encodeSnapshots(config, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	delete(vendor, "second/LICENSE")
	if got, err := decodeSnapshots(config, vendor); got != nil || err == nil {
		t.Fatalf("partial result: %v, %v", got, err)
	}
	second := snapshots["second"]
	second.Commit = "invalid"
	snapshots["second"] = second
	if got, err := encodeSnapshots(config, snapshots); got != nil || err == nil {
		t.Fatalf("partial encoding: %v, %v", got, err)
	}
}

// TestSnapshotWildcardScope rejects a recorded group outside the selected wildcard's scope.
func TestSnapshotWildcardScope(t *testing.T) {
	config, snapshots := snapshotFixture(t)
	config.Sources[0].Groups = rules.GroupSelection{Pattern: "practices/*"}
	item := snapshots["team"]
	item.Selection = config.Sources[0].Groups
	snapshots["team"] = item
	if got, err := encodeSnapshots(config, snapshots); got != nil || err == nil {
		t.Fatal("accepted technology group for practices wildcard")
	}
}
