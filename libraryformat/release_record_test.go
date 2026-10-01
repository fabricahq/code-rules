// Check release tag message parsing against independent fixtures, including record consistency.

package libraryformat_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/authored"
	"github.com/fabricahq/code-rules/libraryformat"
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
			// Error's Kind is "unsupported" for a record format newer than the parser reads, "notReleaseTag" for a tag
			// name that isn't release/<number>, and empty for a validation error.
			Error *struct{ Message, Location, Kind string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.ID, func(t *testing.T) {
			notes, got, err := libraryformat.ParseReleaseMessage(test.Tag, []byte(test.Message))
			if !test.Expected.OK && test.Expected.Error.Kind == "unsupported" {
				var unsupported *libraryformat.UnsupportedReleaseRecordError
				if !errors.As(err, &unsupported) || err.Error() != test.Expected.Error.Message || unsupported.Location != test.Expected.Error.Location {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Error)
				}
				return
			}
			if !test.Expected.OK && test.Expected.Error.Kind == "notReleaseTag" {
				if !errors.Is(err, libraryformat.ErrNotReleaseTag) || err.Error() != test.Expected.Error.Message {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Error)
				}
				return
			}
			if !test.Expected.OK {
				var validation *authored.ValidationError
				if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
					t.Fatalf("got %+v, %v; want %+v", got, err, test.Expected.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var expected libraryformat.ReleaseRecord
			if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
				t.Fatal(err)
			}
			if notes != test.Expected.Notes || !reflect.DeepEqual(got, expected) {
				t.Fatalf("got %q, %+v; want %q, %+v", notes, got, test.Expected.Notes, expected)
			}
		})
	}
}

// recordWith returns a release/2 record whose section, one of rules, changes, retired, or libraryFiles, has count
// entries, with the rules entries the changes need, up to the 10,000 rules allows.
func recordWith(section string, count int) []byte {
	var record strings.Builder
	record.WriteString("formatVersion: 1\nrelease: 2\n")
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
			fmt.Fprintf(&record, "  techs/go/r%d: {change: new, summaries: [Add.]}\n", i)
		}
	case "retired":
		record.WriteString("retired:\n")
		for i := range count {
			fmt.Fprintf(&record, "  techs/go/r%d: {lastVersion: 1.0.0, summaries: [Gone.]}\n", i)
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
			if _, err := libraryformat.ParseReleaseRecord(recordWith(section, limit), "release/2"); err != nil {
				t.Fatalf("refused %d entries: %v", limit, err)
			}
			_, err := libraryformat.ParseReleaseRecord(recordWith(section, limit+1), "release/2")
			var invalid *authored.ValidationError
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
	_, err := libraryformat.ParseReleaseRecord(record, "release/2")
	var invalid *authored.ValidationError
	if !errors.As(err, &invalid) || invalid.Location != "release/2.libraryFiles[19999]" || !strings.Contains(invalid.Problem, "duplicate path") {
		t.Fatalf("got %v", err)
	}
}

