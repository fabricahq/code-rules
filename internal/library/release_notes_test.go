// Check release notes and tag messages rendered from release records, with the expected text inline.

package library

import (
	"os"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/coderules"
)

// version returns a rule version for test records.
func version(major, minor, patch int) *coderules.RuleVersion {
	return &coderules.RuleVersion{Major: major, Minor: minor, Patch: patch}
}

// guideRecord is the library release the version rules guide shows as a GitHub Release page.
func guideRecord() coderules.ReleaseRecord {
	return coderules.ReleaseRecord{
		Release: 4,
		Rules: map[string]coderules.RuleVersion{
			"practices/code-design/organize-code-by-feature": *version(1, 1, 0),
			"practices/testing/verify-backoff":               *version(1, 3, 0),
			"practices/testing/verify-retries":               *version(1, 0, 0),
			"practices/testing/verify-retry-limits":          *version(2, 0, 0),
			"techs/react/prefer-server-components":           *version(1, 4, 0),
			"techs/react/test-hooks-in-isolation":            *version(2, 2, 0),
		},
		Changes: map[string]coderules.RecordedChange{
			"practices/testing/verify-retry-limits": {Change: coderules.ChangeMajor, From: version(1, 3, 0), Summaries: []string{"Require a test at the limit for every retry policy."}},
			"practices/testing/verify-retries":      {Change: coderules.ChangeNew, Summaries: []string{"Add a broader rule about testing retries."}},
			"techs/react/test-hooks-in-isolation":   {Change: coderules.ChangeMinor, From: version(2, 1, 0), Summaries: []string{"Add an example for custom hooks."}},
		},
		Retired: map[string]coderules.RetiredRule{
			"practices/testing/check-retry-backoff": {LastVersion: *version(1, 2, 0), ReplacedBy: "practices/testing/verify-retries", Summaries: []string{"Covered by the broader rule about testing retries."}},
		},
		LibraryFiles: []string{"practices/testing/_group.yaml"},
	}
}

// guideNotes is the example GitHub Release page in the version rules guide.
const guideNotes = "Library release 4 changes 4 rules: 1 new, 1 major, 1 minor, and 1 retired.\n\n## New rules\n\n- **practices/testing/verify-retries** `1.0.0`\n  - Add a broader rule about testing retries.\n\n## Major changes\n\nCode that complied with the previous rule version could fail the new one, so review these before updating.\n\n- **practices/testing/verify-retry-limits** `1.3.0` → `2.0.0`\n  - Require a test at the limit for every retry policy.\n\n## Minor changes\n\n- **techs/react/test-hooks-in-isolation** `2.1.0` → `2.2.0`\n  - Add an example for custom hooks.\n\n## Retired rules\n\n- **practices/testing/check-retry-backoff**, last version `1.2.0`, replaced by **practices/testing/verify-retries**\n  - Covered by the broader rule about testing retries.\n\nThis library release also updates shared files, such as group descriptions or shared assets.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/code-design/organize-code-by-feature | 1.1.0 |\n| practices/testing/verify-backoff | 1.3.0 |\n| practices/testing/verify-retries | 1.0.0 |\n| practices/testing/verify-retry-limits | 2.0.0 |\n| techs/react/prefer-server-components | 1.4.0 |\n| techs/react/test-hooks-in-isolation | 2.2.0 |\n\n</details>"

// TestRenderReleaseNotes_MatchesTheGuideExample keeps the generated page and the guide's example identical.
func TestRenderReleaseNotes_MatchesTheGuideExample(t *testing.T) {
	if notes := renderReleaseNotes(guideRecord()); notes != guideNotes {
		t.Fatalf("notes:\n%s\nwant:\n%s", notes, guideNotes)
	}
	guide, err := os.ReadFile("../../docs/src/content/docs/guides/version-rules.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(guide), "```md wrap\n"+guideNotes+"\n```") {
		t.Fatal("the version rules guide's example GitHub Release page differs from the rendered notes; update both together")
	}
}

// TestRenderReleaseNotes_CountsAndSectionsFollowTheChanges covers singular counts, two kinds, several entries
// and summaries per section, patch changes, the first library release, and the shared-files sentence: a closing
// sentence when rules and library-wide files both change, none for rules alone, and the opening line for
// library-wide files alone.
func TestRenderReleaseNotes_CountsAndSectionsFollowTheChanges(t *testing.T) {
	for _, test := range []struct {
		name   string
		record coderules.ReleaseRecord
		want   string
	}{
		{
			name: "rules and library-wide files",
			record: coderules.ReleaseRecord{
				Release:      7,
				Rules:        map[string]coderules.RuleVersion{"practices/testing/a": *version(1, 1, 0)},
				Changes:      map[string]coderules.RecordedChange{"practices/testing/a": {Change: coderules.ChangeMinor, From: version(1, 0, 0), Summaries: []string{"Add an example."}}},
				LibraryFiles: []string{"practices/testing/_group.yaml"},
			},
			want: "Library release 7 changes 1 rule: 1 minor.\n\n## Minor changes\n\n- **practices/testing/a** `1.0.0` → `1.1.0`\n  - Add an example.\n\nThis library release also updates shared files, such as group descriptions or shared assets.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.1.0 |\n\n</details>",
		},
		{
			name: "rules without library-wide files: one patch with two notes",
			record: coderules.ReleaseRecord{
				Release: 3,
				Rules:   map[string]coderules.RuleVersion{"practices/testing/a": *version(1, 0, 1)},
				Changes: map[string]coderules.RecordedChange{"practices/testing/a": {Change: coderules.ChangePatch, From: version(1, 0, 0), Summaries: []string{"Fix a typo.", "Clarify an example."}}},
			},
			want: "Library release 3 changes 1 rule: 1 patch.\n\n## Patch changes\n\n- **practices/testing/a** `1.0.0` → `1.0.1`\n  - Fix a typo.\n  - Clarify an example.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.0.1 |\n\n</details>",
		},
		{
			name: "two kinds and a retirement without a replacement",
			record: coderules.ReleaseRecord{
				Release: 5,
				Rules:   map[string]coderules.RuleVersion{"practices/testing/a": *version(1, 10, 0), "practices/testing/b": *version(1, 5, 0)},
				Changes: map[string]coderules.RecordedChange{
					"practices/testing/b": {Change: coderules.ChangeMinor, From: version(1, 4, 0), Summaries: []string{"Add a Go example."}},
					"practices/testing/a": {Change: coderules.ChangeMinor, From: version(1, 9, 0), Summaries: []string{"Add a Python example."}},
				},
				Retired: map[string]coderules.RetiredRule{"practices/testing/c": {LastVersion: *version(3, 1, 4), Summaries: []string{"Agents shouldn't add these comments."}}},
			},
			want: "Library release 5 changes 3 rules: 2 minor and 1 retired.\n\n## Minor changes\n\n- **practices/testing/a** `1.9.0` → `1.10.0`\n  - Add a Python example.\n- **practices/testing/b** `1.4.0` → `1.5.0`\n  - Add a Go example.\n\n## Retired rules\n\n- **practices/testing/c**, last version `3.1.4`\n  - Agents shouldn't add these comments.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.10.0 |\n| practices/testing/b | 1.5.0 |\n\n</details>",
		},
		{
			name: "first library release, which adds its library-wide files without the shared-files sentence",
			record: coderules.ReleaseRecord{
				Release:      1,
				LibraryFiles: []string{"practices/testing/_group.yaml", "rule-library.yaml"},
				Rules:        map[string]coderules.RuleVersion{"practices/testing/a": coderules.FirstRuleVersion, "techs/go/b": coderules.FirstRuleVersion},
				Changes:      map[string]coderules.RecordedChange{"practices/testing/a": {Change: coderules.ChangeNew, Summaries: []string{"Add the rule."}}, "techs/go/b": {Change: coderules.ChangeNew, Summaries: []string{"Add the rule."}}},
			},
			want: "Library release 1 publishes 2 rules.\n\n## New rules\n\n- **practices/testing/a** `1.0.0`\n- **techs/go/b** `1.0.0`\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.0.0 |\n| techs/go/b | 1.0.0 |\n\n</details>",
		},
		{
			name: "library-wide files only",
			record: coderules.ReleaseRecord{
				Release:      6,
				Rules:        map[string]coderules.RuleVersion{"practices/testing/a": *version(1, 2, 3)},
				LibraryFiles: []string{"assets/diagram.svg"},
			},
			want: "Library release 6 changes no rules. It updates shared files, such as group descriptions or shared assets.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.2.3 |\n\n</details>",
		},
		{
			name:   "library-wide files in a library without rules",
			record: coderules.ReleaseRecord{Release: 1, Rules: map[string]coderules.RuleVersion{}, LibraryFiles: []string{"rule-library.yaml"}},
			want:   "Library release 1 publishes no rules, only shared files, such as group descriptions or shared assets.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if notes := renderReleaseNotes(test.record); notes != test.want {
				t.Fatalf("notes:\n%s\nwant:\n%s", notes, test.want)
			}
		})
	}
}

// TestReleaseMessage_WritesNotesThenARecordThatReadsBack puts the record after a --- line, sorted and without
// empty sections, and parses it back to the same record.
func TestReleaseMessage_WritesNotesThenARecordThatReadsBack(t *testing.T) {
	record := coderules.ReleaseRecord{
		Release: 2,
		Rules:   map[string]coderules.RuleVersion{"practices/testing/c": coderules.FirstRuleVersion, "practices/testing/a": *version(2, 0, 0)},
		Changes: map[string]coderules.RecordedChange{
			"practices/testing/c": {Change: coderules.ChangeNew, Summaries: []string{"Replace b with c."}},
			"practices/testing/a": {Change: coderules.ChangeMajor, From: version(1, 0, 0), Summaries: []string{"Fix a typo.", "Replace b with c."}},
		},
		Retired: map[string]coderules.RetiredRule{"practices/testing/b": {LastVersion: coderules.FirstRuleVersion, ReplacedBy: "practices/testing/c", Summaries: []string{"Replace b with c."}}},
	}
	message, err := releaseMessage("Notes.\n\n---\n\nMore notes.", record)
	if err != nil {
		t.Fatal(err)
	}
	want := "Notes.\n\n---\n\nMore notes.\n---\nformatVersion: 1\nrelease: 2\nrules:\n  practices/testing/a: 2.0.0\n  practices/testing/c: 1.0.0\nchanges:\n  practices/testing/a:\n    change: major\n    from: 1.0.0\n    summaries:\n      - Fix a typo.\n      - Replace b with c.\n  practices/testing/c:\n    change: new\n    summaries:\n      - Replace b with c.\nretired:\n  practices/testing/b:\n    lastVersion: 1.0.0\n    replacedBy: practices/testing/c\n    summaries:\n      - Replace b with c.\n"
	if string(message) != want {
		t.Fatalf("message:\n%s\nwant:\n%s", message, want)
	}
	record.LibraryFiles = []string{"practices/testing/_group.yaml", "assets/diagram.svg"}
	message, err = releaseMessage("Notes.", record)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(message), "libraryFiles:\n  - practices/testing/_group.yaml\n  - assets/diagram.svg\n") {
		t.Fatalf("library-wide files keep their order:\n%s", message)
	}
}

// TestReleaseMessage_RefusesARecordTheParserRejects never produces a tag message that projects can't read.
func TestReleaseMessage_RefusesARecordTheParserRejects(t *testing.T) {
	record := coderules.ReleaseRecord{
		Release:      2,
		Rules:        map[string]coderules.RuleVersion{"practices/testing/a": *version(1, 1, 0)},
		Changes:      map[string]coderules.RecordedChange{"practices/testing/a": {Change: coderules.ChangeMajor, From: version(1, 0, 0), Summaries: []string{"Tighten a."}}},
		LibraryFiles: []string{"practices/testing/a.md"},
	}
	if _, err := releaseMessage("Notes.", record); err == nil {
		t.Fatal("expected an inconsistent record to fail")
	}
}
