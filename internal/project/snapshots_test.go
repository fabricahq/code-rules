// Exercise snapshot integrity through the encoding and decoding boundary.

package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// snapshotFixture supplies one empty group with original binary and CRLF files.
func snapshotFixture(t *testing.T) (rules.Configuration, map[string]snapshot) {
	t.Helper()
	config, err := rules.ParseConfiguration([]byte(`{"schemaVersion":1,"sources":{"team":{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":{},"replace":{}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	return config, map[string]snapshot{"team": {Repository: config.Sources[0].Repository, Ref: "v1.0.0", Commit: strings.Repeat("a", 40), Groups: []string{"techs/go"}, Selection: config.Sources[0].Groups, Files: map[string][]byte{
		"rule-library.json":    []byte(`{"formatVersion":1}`),
		"techs/go/_group.json": []byte(`{"name":"Go","description":"Go rules","whenToRead":"When editing Go."}`),
		"assets/image.bin":     {0, 255, 10, 128}, "LICENSE": []byte("Terms\r\nPreserved\r\n"), "empty.txt": {},
	}}}
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
			var validation *rules.ValidationError
			if got != nil || !errors.As(err, &validation) {
				t.Fatalf("accepted stale repository address: %v, %v", got, err)
			}
			if validation.Location != "team/_source.json" || !strings.Contains(validation.Problem, "run code-rules project sync") {
				t.Fatalf("missing source location or recovery instruction: %v", err)
			}
		})
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

// TestSnapshotRecordRelationships validates identity and selection rather than trusting hashes alone.
func TestSnapshotRecordRelationships(t *testing.T) {
	cases := []struct {
		name  string
		alter func(map[string]any)
	}{
		{"new format", func(r map[string]any) { r["formatVersion"] = 2 }},
		{"unknown", func(r map[string]any) { r["future"] = true }},
		{"wrong case", func(r map[string]any) { r["Repository"] = r["repository"]; delete(r, "repository") }},
		{"null optional", func(r map[string]any) { r["resolvedTag"] = nil }},
		{"invalid commit", func(r map[string]any) { r["resolvedCommit"] = "main" }},
		{"changed repository", func(r map[string]any) { r["repository"] = "https://github.com/acme/other" }},
		{"changed ref", func(r map[string]any) { r["ref"] = "v2.0.0" }},
		{"changed selection", func(r map[string]any) { r["groupSelection"] = "*" }},
		{"missing group", func(r map[string]any) { r["groups"] = []string{} }},
		{"duplicate group", func(r map[string]any) { r["groups"] = []string{"techs/go", "techs/go"} }},
		{"bad hash", func(r map[string]any) { r["files"].(map[string]any)["LICENSE"] = "123" }},
		{"reserved", func(r map[string]any) { r["files"].(map[string]any)["_source.json"] = strings.Repeat("a", 64) }},
		{"traversal", func(r map[string]any) { r["files"].(map[string]any)["../outside"] = strings.Repeat("a", 64) }},
		{"missing manifest", func(r map[string]any) { delete(r["files"].(map[string]any), "rule-library.json") }},
		{"missing metadata", func(r map[string]any) { delete(r["files"].(map[string]any), "techs/go/_group.json") }},
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

// TestSnapshotLegacySelectionAndEmptySources preserves version-1 omission and local-only behavior.
func TestSnapshotLegacySelectionAndEmptySources(t *testing.T) {
	c, s := snapshotFixture(t)
	v, e := encodeSnapshots(c, s)
	if e != nil {
		t.Fatal(e)
	}
	var record map[string]any
	if e = json.Unmarshal(v["team/_source.json"], &record); e != nil {
		t.Fatal(e)
	}
	delete(record, "groupSelection")
	v["team/_source.json"], e = json.Marshal(record)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := decodeSnapshots(c, v); e != nil {
		t.Fatal(e)
	}
	c.Sources[0].Groups = rules.GroupSelection{Pattern: "*"}
	if _, e := decodeSnapshots(c, v); e == nil {
		t.Fatal("legacy explicit groups satisfied a wildcard")
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

// TestSnapshotVersionIdentity preserves readable constraints and verifies the selected release.
func TestSnapshotVersionIdentity(t *testing.T) {
	c, s := snapshotFixture(t)
	c.Sources[0].Ref = ""
	c.Sources[0].ParsedRef = nil
	c.Sources[0].Version = ">= 1.0.0, < 2.0.0"
	item := s["team"]
	item.Ref = ""
	item.Version = c.Sources[0].Version
	item.Tag = "v1.2.3"
	item.ResolvedVersion = "1.2.3"
	s["team"] = item
	v, e := encodeSnapshots(c, s)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(v["team/_source.json"], []byte(`>= 1.0.0, < 2.0.0`)) {
		t.Fatal("constraint escaped")
	}
	if _, e := decodeSnapshots(c, v); e != nil {
		t.Fatal(e)
	}
	item.ResolvedVersion = "1.2.4"
	s["team"] = item
	if got, e := encodeSnapshots(c, s); e == nil || got != nil {
		t.Fatal("accepted different release")
	}
	item.ResolvedVersion = "2.0.0"
	item.Tag = "v2.0.0"
	s["team"] = item
	if got, e := encodeSnapshots(c, s); e == nil || got != nil {
		t.Fatal("accepted release outside constraint")
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

// TestSnapshotWildcardAndCommitIdentity verifies resolved selections and immutable commit requests.
func TestSnapshotWildcardAndCommitIdentity(t *testing.T) {
	config, snapshots := snapshotFixture(t)
	config.Sources[0].Groups = rules.GroupSelection{Pattern: "practices/*"}
	item := snapshots["team"]
	item.Selection = config.Sources[0].Groups
	snapshots["team"] = item
	if got, err := encodeSnapshots(config, snapshots); got != nil || err == nil {
		t.Fatal("accepted technology group for practices wildcard")
	}
	config, snapshots = snapshotFixture(t)
	config.Sources[0].Ref = strings.Repeat("b", 40)
	ref, err := rules.ParseGitRef(config.Sources[0].Ref, "ref")
	if err != nil {
		t.Fatal(err)
	}
	config.Sources[0].ParsedRef = &ref
	item = snapshots["team"]
	item.Ref = config.Sources[0].Ref
	snapshots["team"] = item
	if got, err := encodeSnapshots(config, snapshots); got != nil || err == nil {
		t.Fatal("accepted mismatched commit")
	}
}
