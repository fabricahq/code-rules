// Check release notes and tag messages rendered from release records, with the expected text inline.

package library

import (
	"os"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// version returns a rule version for test records.
func version(major, minor, patch int) *rules.RuleVersion {
	return &rules.RuleVersion{Major: major, Minor: minor, Patch: patch}
}

// guideRecord is the library release the version rules guide shows as a GitHub Release page.
func guideRecord() rules.ReleaseRecord {
	return rules.ReleaseRecord{
		Release: 4,
		Rules: map[string]rules.RuleVersion{
			"practices/code-design/organize-code-by-feature": *version(1, 1, 0),
			"practices/testing/verify-backoff":               *version(1, 3, 0),
			"practices/testing/verify-retries":               *version(1, 0, 0),
			"practices/testing/verify-retry-limits":          *version(2, 0, 0),
			"techs/react/prefer-server-components":           *version(1, 4, 0),
			"techs/react/test-hooks-in-isolation":            *version(2, 2, 0),
		},
		Changes: map[string]rules.RecordedChange{
			"practices/testing/verify-retry-limits": {Change: rules.ChangeMajor, From: version(1, 3, 0), Summary: "Require a test at the limit for every retry policy."},
			"practices/testing/verify-retries":      {Change: rules.ChangeNew, Summary: "Add a broader rule about testing retries."},
			"techs/react/test-hooks-in-isolation":   {Change: rules.ChangeMinor, From: version(2, 1, 0), Summary: "Add an example for custom hooks."},
		},
		Retired: map[string]rules.RetiredRule{
			"practices/testing/check-retry-backoff": {LastVersion: *version(1, 2, 0), ReplacedBy: "practices/testing/verify-retries", Summary: "Covered by the broader rule about testing retries."},
		},
		LibraryFiles: []string{"practices/testing/_group.yaml"},
	}
}

// guideNotes is the example GitHub Release page in the version rules guide.
const guideNotes = "Library release 4 changes 4 rules:\n1 major, 1 minor, 1 new, and 1 retired.\n\n## Major changes\n\nCode that complied with the previous rule version could fail\nthe new one, so review these before updating.\n\n- **practices/testing/verify-retry-limits** `1.3.0` → `2.0.0`\n  Require a test at the limit for every retry policy.\n\n## Minor changes\n\n- **techs/react/test-hooks-in-isolation** `2.1.0` → `2.2.0`\n  Add an example for custom hooks.\n\n## New rules\n\n- **practices/testing/verify-retries** `1.0.0`\n  Add a broader rule about testing retries.\n\n## Retired rules\n\n- **practices/testing/check-retry-backoff**, last version `1.2.0`\n  Covered by the broader rule about testing retries.\n  Replaced by **practices/testing/verify-retries**.\n\nThis library release also updates shared files, such as group\ndescriptions or shared assets.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/code-design/organize-code-by-feature | 1.1.0 |\n| practices/testing/verify-backoff | 1.3.0 |\n| practices/testing/verify-retries | 1.0.0 |\n| practices/testing/verify-retry-limits | 2.0.0 |\n| techs/react/prefer-server-components | 1.4.0 |\n| techs/react/test-hooks-in-isolation | 2.2.0 |\n\n</details>"

// TestRenderReleaseNotes_MatchesTheGuideExample keeps the generated page and the guide's example identical.
func TestRenderReleaseNotes_MatchesTheGuideExample(t *testing.T) {
	if notes := renderReleaseNotes(guideRecord()); notes != guideNotes {
		t.Fatalf("notes:\n%s\nwant:\n%s", notes, guideNotes)
	}
	guide, err := os.ReadFile("../../docs/src/content/docs/guides/version-rules.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(guide), "```md\n"+guideNotes+"\n```") {
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
		record rules.ReleaseRecord
		want   string
	}{
		{
			name: "rules and library-wide files",
			record: rules.ReleaseRecord{
				Release:      7,
				Rules:        map[string]rules.RuleVersion{"practices/testing/a": *version(1, 1, 0)},
				Changes:      map[string]rules.RecordedChange{"practices/testing/a": {Change: rules.ChangeMinor, From: version(1, 0, 0), Summary: "Add an example."}},
				LibraryFiles: []string{"practices/testing/_group.yaml"},
			},
			want: "Library release 7 changes 1 rule:\n1 minor.\n\n## Minor changes\n\n- **practices/testing/a** `1.0.0` → `1.1.0`\n  Add an example.\n\nThis library release also updates shared files, such as group\ndescriptions or shared assets.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.1.0 |\n\n</details>",
		},
		{
			name: "rules without library-wide files: one patch with two notes",
			record: rules.ReleaseRecord{
				Release: 3,
				Rules:   map[string]rules.RuleVersion{"practices/testing/a": *version(1, 0, 1)},
				Changes: map[string]rules.RecordedChange{"practices/testing/a": {Change: rules.ChangePatch, From: version(1, 0, 0), Summary: "Fix a typo.\nClarify an example."}},
			},
			want: "Library release 3 changes 1 rule:\n1 patch.\n\n## Patch changes\n\n- **practices/testing/a** `1.0.0` → `1.0.1`\n  Fix a typo.\n  Clarify an example.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.0.1 |\n\n</details>",
		},
		{
			name: "two kinds and a retirement without a replacement",
			record: rules.ReleaseRecord{
				Release: 5,
				Rules:   map[string]rules.RuleVersion{"practices/testing/a": *version(1, 10, 0), "practices/testing/b": *version(1, 5, 0)},
				Changes: map[string]rules.RecordedChange{
					"practices/testing/b": {Change: rules.ChangeMinor, From: version(1, 4, 0), Summary: "Add a Go example."},
					"practices/testing/a": {Change: rules.ChangeMinor, From: version(1, 9, 0), Summary: "Add a Python example."},
				},
				Retired: map[string]rules.RetiredRule{"practices/testing/c": {LastVersion: *version(3, 1, 4), Summary: "Agents shouldn't add these comments."}},
			},
			want: "Library release 5 changes 3 rules:\n2 minor and 1 retired.\n\n## Minor changes\n\n- **practices/testing/a** `1.9.0` → `1.10.0`\n  Add a Python example.\n- **practices/testing/b** `1.4.0` → `1.5.0`\n  Add a Go example.\n\n## Retired rules\n\n- **practices/testing/c**, last version `3.1.4`\n  Agents shouldn't add these comments.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.10.0 |\n| practices/testing/b | 1.5.0 |\n\n</details>",
		},
		{
			name: "first library release, which adds its library-wide files without the shared-files sentence",
			record: rules.ReleaseRecord{
				Release:      1,
				LibraryFiles: []string{"practices/testing/_group.yaml", "rule-library.yaml"},
				Rules:        map[string]rules.RuleVersion{"practices/testing/a": rules.FirstRuleVersion, "techs/go/b": rules.FirstRuleVersion},
				Changes:      map[string]rules.RecordedChange{"practices/testing/a": {Change: rules.ChangeNew, Summary: "Add the rule."}, "techs/go/b": {Change: rules.ChangeNew, Summary: "Add the rule."}},
			},
			want: "Library release 1 changes 2 rules:\n2 new.\n\n## New rules\n\n- **practices/testing/a** `1.0.0`\n  Add the rule.\n- **techs/go/b** `1.0.0`\n  Add the rule.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.0.0 |\n| techs/go/b | 1.0.0 |\n\n</details>",
		},
		{
			name: "library-wide files only",
			record: rules.ReleaseRecord{
				Release:      6,
				Rules:        map[string]rules.RuleVersion{"practices/testing/a": *version(1, 2, 3)},
				LibraryFiles: []string{"assets/diagram.svg"},
			},
			want: "Library release 6 changes no rules.\nIt updates shared files, such as group descriptions or shared assets.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.2.3 |\n\n</details>",
		},
		{
			name:   "library-wide files in a library without rules",
			record: rules.ReleaseRecord{Release: 1, Rules: map[string]rules.RuleVersion{}, LibraryFiles: []string{"rule-library.yaml"}},
			want:   "Library release 1 changes no rules.\nIt updates shared files, such as group descriptions or shared assets.",
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
	record := rules.ReleaseRecord{
		Release: 2,
		Rules:   map[string]rules.RuleVersion{"practices/testing/c": rules.FirstRuleVersion, "practices/testing/a": *version(2, 0, 0)},
		Changes: map[string]rules.RecordedChange{
			"practices/testing/c": {Change: rules.ChangeNew, Summary: "Replace b with c."},
			"practices/testing/a": {Change: rules.ChangeMajor, From: version(1, 0, 0), Summary: "Fix a typo.\nReplace b with c."},
		},
		Retired: map[string]rules.RetiredRule{"practices/testing/b": {LastVersion: rules.FirstRuleVersion, ReplacedBy: "practices/testing/c", Summary: "Replace b with c."}},
	}
	message, err := releaseMessage("Notes.\n\n---\n\nMore notes.", record)
	if err != nil {
		t.Fatal(err)
	}
	want := "Notes.\n\n---\n\nMore notes.\n---\nrelease: 2\nrules:\n  practices/testing/a: 2.0.0\n  practices/testing/c: 1.0.0\nchanges:\n  practices/testing/a:\n    change: major\n    from: 1.0.0\n    summary: |-\n      Fix a typo.\n      Replace b with c.\n  practices/testing/c:\n    change: new\n    summary: Replace b with c.\nretired:\n  practices/testing/b:\n    lastVersion: 1.0.0\n    replacedBy: practices/testing/c\n    summary: Replace b with c.\n"
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
	record := rules.ReleaseRecord{
		Release:      2,
		Rules:        map[string]rules.RuleVersion{"practices/testing/a": *version(1, 1, 0)},
		Changes:      map[string]rules.RecordedChange{"practices/testing/a": {Change: rules.ChangeMajor, From: version(1, 0, 0), Summary: "Tighten a."}},
		LibraryFiles: []string{"practices/testing/a.md"},
	}
	if _, err := releaseMessage("Notes.", record); err == nil {
		t.Fatal("expected an inconsistent record to fail")
	}
}
