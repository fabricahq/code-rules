// Check release tag message parsing against independent fixtures, including record consistency.

package rules_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestReleaseMessageFixtures checks the notes, the parsed record, and exact diagnostics.
func TestReleaseMessageFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/release-records/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Tag, Message string
		Expected         struct {
			OK    bool
			Notes string
			Value json.RawMessage
			Error *struct{ Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			notes, got, err := rules.ParseReleaseMessage(test.Tag, []byte(test.Message))
			if !test.Expected.OK {
				var validation *rules.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var expected rules.ReleaseRecord
			if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
				t.Fatal(err)
			}
			if notes != test.Expected.Notes || !reflect.DeepEqual(got, expected) {
				t.Fatalf("got %q, %+v; want %q, %+v", notes, got, test.Expected.Notes, expected)
			}
		})
	}
}

// TestParseReleaseTagObject_SkipsHeadersAndSignature reads the record from a signed tag object's message only,
// whether the tag is signed with GPG or SSH.
func TestParseReleaseTagObject_SkipsHeadersAndSignature(t *testing.T) {
	for _, signature := range []string{"-----BEGIN PGP SIGNATURE-----\nU0lH\n-----END PGP SIGNATURE-----\n", "-----BEGIN SSH SIGNATURE-----\nU1NIU0lH\n-----END SSH SIGNATURE-----\n"} {
		object := "object 0123456789012345678901234567890123456789\ntype commit\ntag release/2\ntagger Fixture <fixture@example.invalid> 0 +0000\n\nNotes.\n\n---\nrelease: 2\nrules:\n  techs/go/a: 1.0.0\n" + signature
		notes, record, err := rules.ParseReleaseTagObject("release/2", []byte(object))
		if err != nil || notes != "Notes." || record.Release != 2 || len(record.Rules) != 1 {
			t.Fatalf("got %q, %+v, %v", notes, record, err)
		}
		if _, _, err := rules.ParseReleaseTagObject("release/3", []byte(object)); err == nil {
			t.Fatal("accepted a record for another library release")
		}
	}
}

// recordWith returns a release/2 record whose section, one of rules, changes, retired, or libraryFiles, has count
// entries, with the rules entries the changes need, up to the 10,000 rules allows.
func recordWith(section string, count int) []byte {
	var record strings.Builder
	record.WriteString("release: 2\n")
	if section == "rules" || section == "changes" {
		ruleCount := count
		if section == "changes" {
			ruleCount = min(count, 10_000)
		}
		record.WriteString("rules:\n")
		for i := range ruleCount {
			fmt.Fprintf(&record, "  techs/go/r%d: 1.0.0\n", i)
		}
	} else {
		record.WriteString("rules: {}\n")
	}
	switch section {
	case "changes":
		record.WriteString("changes:\n")
		for i := range count {
			fmt.Fprintf(&record, "  techs/go/r%d: {change: new, summary: Add.}\n", i)
		}
	case "retired":
		record.WriteString("retired:\n")
		for i := range count {
			fmt.Fprintf(&record, "  techs/go/r%d: {lastVersion: 1.0.0, summary: Gone.}\n", i)
		}
	case "libraryFiles":
		record.WriteString("libraryFiles:\n")
		for i := range count {
			fmt.Fprintf(&record, "  - assets/f%d.md\n", i)
		}
	}
	return []byte(record.String())
}

// TestParseReleaseRecord_LimitsEachCollection accepts each collection at its documented limit and refuses one
// more entry, naming the limit.
func TestParseReleaseRecord_LimitsEachCollection(t *testing.T) {
	for section, limit := range map[string]int{"rules": 10_000, "changes": 10_000, "retired": 10_000, "libraryFiles": 20_000} {
		t.Run(section, func(t *testing.T) {
			if _, err := rules.ParseReleaseRecord(recordWith(section, limit), "release/2"); err != nil {
				t.Fatalf("refused %d entries: %v", limit, err)
			}
			_, err := rules.ParseReleaseRecord(recordWith(section, limit+1), "release/2")
			var invalid *rules.ValidationError
			want := fmt.Sprintf("expected at most %d,000 entries", limit/1000)
			if !errors.As(err, &invalid) || invalid.Location != "release/2."+section || invalid.Problem != want {
				t.Fatalf("got %v; want %q", err, want)
			}
		})
	}
}

// TestParseReleaseRecord_RefusesDuplicateLibraryFilesInLargeLists finds a duplicate path at the end of the
// longest list the limit allows.
func TestParseReleaseRecord_RefusesDuplicateLibraryFilesInLargeLists(t *testing.T) {
	record := append(recordWith("libraryFiles", 19_999), "  - assets/f0.md\n"...)
	_, err := rules.ParseReleaseRecord(record, "release/2")
	var invalid *rules.ValidationError
	if !errors.As(err, &invalid) || invalid.Location != "release/2.libraryFiles[19999]" || !strings.Contains(invalid.Problem, "duplicate path") {
		t.Fatalf("got %v", err)
	}
}
