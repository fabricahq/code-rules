// Exercise choosing rule versions from real release tags: newest, pinned, recorded, retired, and ref imports.

package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// versionedRule is a valid rule whose body names its version, so tests can tell which version's file arrived.
func versionedRule(version string) []byte {
	return []byte("---\ntitle: Rule\nimpact: HIGH\nimpactDescription: Matters.\nwhenToRead: Always.\n---\n# Rule\n\nVersion " + version + " text.\n")
}

// groupMetadata is valid group metadata.
var groupMetadata = []byte(`{"name":"Group","description":"Rules.","whenToRead":"Always."}`)

// history is a library whose release history the tests share:
//
//   - release/1: techs/go/a and techs/go/b at 1.0.0, with a's own asset; practices/testing/c at 1.0.0.
//   - release/2: a minor to 1.1.0, new techs/go/d at 1.0.0.
//   - release/3: b retired, replaced by d; a major to 2.0.0.
//
// Commits after release/3 are unreleased.
type history struct {
	fixture *gitfixture.Fixture
	commits map[int]string
}

// newHistory publishes the shared library history to a fixture repository.
func newHistory(t *testing.T) history {
	t.Helper()
	f := newLibraryFixture(t, map[string][]byte{
		"rule-library.yaml":             []byte(`{"formatVersion":1}`),
		"techs/go/_group.yaml":          groupMetadata,
		"techs/go/a.md":                 versionedRule("a 1.0.0"),
		"techs/go/assets/a/diagram.bin": {1, 0},
		"techs/go/b.md":                 versionedRule("b 1.0.0"),
		"practices/testing/_group.yaml": groupMetadata,
		"practices/testing/c.md":        versionedRule("c 1.0.0"),
	})
	h := history{fixture: f, commits: map[int]string{}}
	h.release(t, 1, nil, "release: 1\nrules:\n  techs/go/a: 1.0.0\n  techs/go/b: 1.0.0\n  practices/testing/c: 1.0.0\nchanges:\n  techs/go/a: {change: new, summary: Add the rule.}\n  techs/go/b: {change: new, summary: Add the rule.}\n  practices/testing/c: {change: new, summary: Add the rule.}\n")
	h.release(t, 2, map[string][]byte{"techs/go/a.md": versionedRule("a 1.1.0"), "techs/go/assets/a/diagram.bin": {1, 1}, "techs/go/d.md": versionedRule("d 1.0.0")},
		"release: 2\nrules:\n  techs/go/a: 1.1.0\n  techs/go/b: 1.0.0\n  techs/go/d: 1.0.0\n  practices/testing/c: 1.0.0\nchanges:\n  techs/go/a: {change: minor, from: 1.0.0, summary: Add an example.}\n  techs/go/d: {change: new, summary: Add the rule.}\n")
	h.release(t, 3, map[string][]byte{"techs/go/a.md": versionedRule("a 2.0.0"), "techs/go/b.md": nil},
		"release: 3\nrules:\n  techs/go/a: 2.0.0\n  techs/go/d: 1.0.0\n  practices/testing/c: 1.0.0\nchanges:\n  techs/go/a: {change: major, from: 1.1.0, summary: Require more.}\nretired:\n  techs/go/b: {lastVersion: 1.0.0, replacedBy: techs/go/d, summary: Covered by d.}\n")
	return h
}

// release commits files to the fixture, when any, and publishes the library release number with record.
func (h history) release(t *testing.T, number int, files map[string][]byte, record string) {
	t.Helper()
	ctx := context.Background()
	commit, err := h.fixture.Commit(ctx, h.fixture.Worktree(), fmt.Sprintf("Release %d", number), files)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.fixture.Release(ctx, number, record); err != nil {
		t.Fatal(err)
	}
	h.commits[number] = commit
}

// source parses a single source named team from its JSON fields, adding the fixture's repository.
func (h history) source(t *testing.T, fields string) rules.Configuration {
	t.Helper()
	raw := `{"schemaVersion":1,"sources":{"team":{"repository":` + quoteJSON(h.fixture.Repository) + `,` + fields + `}}}`
	config, err := rules.ParseConfiguration(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	return config
}

// sync imports the configuration with an optional recorded snapshot, as project sync does.
func (h history) sync(t *testing.T, config rules.Configuration, recorded *library.Snapshot) (Library, error) {
	t.Helper()
	previous := map[string]library.Snapshot{}
	if recorded != nil {
		previous["team"] = *recorded
	}
	result, err := ImportLibraries(context.Background(), config, previous, Options{GitPath: h.fixture.GitPath, Environment: h.fixture.Environment})
	return result["team"], err
}

// versions summarizes each imported rule as version@release, or "unreleased".
func versions(snapshot library.Snapshot) map[string]string {
	result := map[string]string{}
	for id, rule := range snapshot.Rules {
		if rule.Version == nil {
			result[id] = "unreleased"
			continue
		}
		result[id] = fmt.Sprintf("%s@%d", rule.Version, rule.Release)
	}
	return result
}

// quoteJSON quotes text as a JSON string.
func quoteJSON(text string) string {
	data, _ := json.Marshal(text)
	return string(data)
}

// TestImport_NewSourceGetsEachRulesNewestVersionFromItsLibraryRelease stores older rules under _releases/<n>/.
func TestImport_NewSourceGetsEachRulesNewestVersionFromItsLibraryRelease(t *testing.T) {
	h := newHistory(t)
	imported, err := h.sync(t, h.source(t, `"groups":["techs/go"]`), nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := imported.Snapshot
	if want := map[string]string{"techs/go/a": "2.0.0@3", "techs/go/d": "1.0.0@2"}; !reflect.DeepEqual(versions(snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(snapshot), want)
	}
	if snapshot.Release != 3 || snapshot.Commit != h.commits[3] || snapshot.Rules["techs/go/d"].Commit != h.commits[2] {
		t.Fatalf("snapshot %+v", snapshot)
	}
	want := []string{"_releases/2/techs/go/d.md", "rule-library.yaml", "techs/go/_group.yaml", "techs/go/a.md", "techs/go/assets/a/diagram.bin"}
	if got := slices.Sorted(maps.Keys(snapshot.Files)); !reflect.DeepEqual(got, want) {
		t.Fatalf("files %v, want %v", got, want)
	}
	if string(snapshot.Files["techs/go/a.md"]) != string(versionedRule("a 2.0.0")) || len(imported.Warnings) != 0 {
		t.Fatalf("a.md %q, warnings %v", snapshot.Files["techs/go/a.md"], imported.Warnings)
	}
}

// TestImport_RecordedVersionsStayWhenNewerOnesExist restores a snapshot without adopting newer versions.
func TestImport_RecordedVersionsStayWhenNewerOnesExist(t *testing.T) {
	h := newHistory(t)
	config := h.source(t, `"groups":["techs/go"]`)
	one, two := rules.RuleVersion{Major: 1}, rules.RuleVersion{Major: 1, Minor: 1}
	recorded := library.Snapshot{Repository: h.fixture.Repository, Release: 2, Commit: h.commits[2], Selection: config.Sources[0].Groups, Groups: []string{"techs/go"}, RuleSelection: []string{},
		Rules: map[string]library.ImportedRule{"techs/go/a": {Version: &two, Release: 2, Commit: h.commits[2]}, "techs/go/b": {Version: &one, Release: 1, Commit: h.commits[1]}}}
	imported, err := h.sync(t, config, &recorded)
	if err != nil {
		t.Fatal(err)
	}
	// b stays although release/3 retired it, and d, added to the group after the record, doesn't join.
	if want := map[string]string{"techs/go/a": "1.1.0@2", "techs/go/b": "1.0.0@1"}; !reflect.DeepEqual(versions(imported.Snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(imported.Snapshot), want)
	}
	if imported.Snapshot.Release != 2 || string(imported.Snapshot.Files["techs/go/a.md"]) != string(versionedRule("a 1.1.0")) || string(imported.Snapshot.Files["_releases/1/techs/go/b.md"]) != string(versionedRule("b 1.0.0")) {
		t.Fatalf("snapshot %+v", imported.Snapshot)
	}
}

// TestImport_PinsMoveRulesUpAndDown chooses the pinned version for added or changed pins only.
func TestImport_PinsMoveRulesUpAndDown(t *testing.T) {
	h := newHistory(t)
	down, err := h.sync(t, h.source(t, `"groups":["techs/go"],"pins":{"techs/go/a":{"version":"1.0.0","reason":"Not ready."}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"techs/go/a": "1.0.0@1", "techs/go/d": "1.0.0@2"}; !reflect.DeepEqual(versions(down.Snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(down.Snapshot), want)
	}
	// The library-wide files come from the newest library release among the imported rule versions.
	if down.Snapshot.Release != 2 || string(down.Snapshot.Files["_releases/1/techs/go/assets/a/diagram.bin"]) != string([]byte{1, 0}) {
		t.Fatalf("snapshot release %d, files %v", down.Snapshot.Release, slices.Sorted(maps.Keys(down.Snapshot.Files)))
	}
	up, err := h.sync(t, h.source(t, `"groups":["techs/go"],"pins":{"techs/go/a":{"version":"1.1.0","reason":"Ready for the example."}}`), &down.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"techs/go/a": "1.1.0@2", "techs/go/d": "1.0.0@2"}; !reflect.DeepEqual(versions(up.Snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(up.Snapshot), want)
	}
	// Removing the pin keeps the recorded version.
	unpinned, err := h.sync(t, h.source(t, `"groups":["techs/go"]`), &up.Snapshot)
	if err != nil || versions(unpinned.Snapshot)["techs/go/a"] != "1.1.0@2" {
		t.Fatalf("versions %v, %v", versions(unpinned.Snapshot), err)
	}
	_, err = h.sync(t, h.source(t, `"groups":["techs/go"],"pins":{"techs/go/a":{"version":"1.2.0","reason":"Typo."}}`), &up.Snapshot)
	requireCode(t, err, "version-not-found")
}

// TestImport_NewlySelectedRulesGetTheirNewestVersion leaves recorded rules and adds only the new selection.
func TestImport_NewlySelectedRulesGetTheirNewestVersion(t *testing.T) {
	h := newHistory(t)
	config := h.source(t, `"groups":["techs/go"]`)
	one := rules.RuleVersion{Major: 1}
	recorded := library.Snapshot{Repository: h.fixture.Repository, Release: 1, Commit: h.commits[1], Selection: config.Sources[0].Groups, Groups: []string{"techs/go"}, RuleSelection: []string{},
		Rules: map[string]library.ImportedRule{"techs/go/a": {Version: &one, Release: 1, Commit: h.commits[1]}}}
	imported, err := h.sync(t, h.source(t, `"groups":["techs/go","practices/testing"]`), &recorded)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"techs/go/a": "1.0.0@1", "practices/testing/c": "1.0.0@1"}; !reflect.DeepEqual(versions(imported.Snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(imported.Snapshot), want)
	}
	individual, err := h.sync(t, h.source(t, `"groups":["techs/go"],"rules":["techs/go/d"]`), &recorded)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"techs/go/a": "1.0.0@1", "techs/go/d": "1.0.0@2"}; !reflect.DeepEqual(versions(individual.Snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(individual.Snapshot), want)
	}
}

// TestImport_IndividuallySelectedRuleComesWithoutTheRestOfItsGroup brings its group metadata only.
func TestImport_IndividuallySelectedRuleComesWithoutTheRestOfItsGroup(t *testing.T) {
	h := newHistory(t)
	imported, err := h.sync(t, h.source(t, `"rules":["techs/go/d"]`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"techs/go/d": "1.0.0@2"}; !reflect.DeepEqual(versions(imported.Snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(imported.Snapshot), want)
	}
	if want := []string{"rule-library.yaml", "techs/go/_group.yaml", "techs/go/d.md"}; !reflect.DeepEqual(slices.Sorted(maps.Keys(imported.Snapshot.Files)), want) {
		t.Fatalf("files %v", slices.Sorted(maps.Keys(imported.Snapshot.Files)))
	}
	if imported.Snapshot.Release != 2 || len(imported.Snapshot.Groups) != 0 || !reflect.DeepEqual(imported.Snapshot.RuleSelection, []string{"techs/go/d"}) {
		t.Fatalf("snapshot %+v", imported.Snapshot)
	}
}

// TestImport_EntriesNamingRetiredRulesWarnAndUnknownOnesFail covers rules, pins, and exclusions.
func TestImport_EntriesNamingRetiredRulesWarnAndUnknownOnesFail(t *testing.T) {
	h := newHistory(t)
	for _, test := range []struct {
		name, fields, warning, location string
	}{
		{"retired rules entry", `"groups":["techs/go"],"rules":["techs/go/b"]`, "sources.team.rules names techs/go/b", ""},
		{"retired exclusion", `"groups":["techs/go"],"exclude":{"techs/go/b":{"reason":"Old."}}`, "sources.team.exclude names techs/go/b", ""},
		{"retired pin", `"groups":["techs/go"],"pins":{"techs/go/b":{"version":"1.0.0","reason":"Keep."}}`, "sources.team.pins names techs/go/b", ""},
		{"unknown rules entry", `"groups":["techs/go"],"rules":["techs/go/missing"]`, "", "sources.team.rules"},
		{"unknown exclusion", `"groups":["techs/go"],"exclude":{"techs/go/missing":{"reason":"Typo."}}`, "", "sources.team.exclude.techs/go/missing"},
		{"exclusion of an unselected rule", `"groups":["techs/go"],"exclude":{"practices/testing/c":{"reason":"Not selected."}}`, "", "sources.team.exclude.practices/testing/c"},
		{"unknown pin", `"groups":["techs/go"],"pins":{"techs/go/missing":{"version":"1.0.0","reason":"Typo."}}`, "", "sources.team.pins.techs/go/missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			imported, err := h.sync(t, h.source(t, test.fields), nil)
			if test.location != "" {
				var validation *rules.ValidationError
				if !errors.As(err, &validation) || validation.Location != test.location {
					t.Fatalf("got %v, want a failure at %s", err, test.location)
				}
				return
			}
			if err != nil || len(imported.Warnings) != 1 || !strings.HasPrefix(imported.Warnings[0], test.warning) {
				t.Fatalf("warnings %v, %v", imported.Warnings, err)
			}
			if _, ok := imported.Snapshot.Rules["techs/go/b"]; ok {
				t.Fatal("imported a retired rule")
			}
		})
	}
}

// TestImport_PinnedRuleTheLibraryRetiredKeepsImporting restores a pin recorded before the retirement.
func TestImport_PinnedRuleTheLibraryRetiredKeepsImporting(t *testing.T) {
	h := newHistory(t)
	pinned := `"groups":["techs/go"],"pins":{"techs/go/b":{"version":"1.0.0","reason":"Keep."}}`
	one := rules.RuleVersion{Major: 1}
	config := h.source(t, pinned)
	recorded := library.Snapshot{Repository: h.fixture.Repository, Pins: config.Sources[0].Pins, Release: 1, Commit: h.commits[1], Selection: config.Sources[0].Groups, Groups: []string{"techs/go"}, RuleSelection: []string{},
		Rules: map[string]library.ImportedRule{"techs/go/b": {Version: &one, Release: 1, Commit: h.commits[1]}}}
	// Selecting practices/testing makes this sync read the history, which knows b is retired.
	imported, err := h.sync(t, h.source(t, `"groups":["techs/go","practices/testing"],"pins":{"techs/go/b":{"version":"1.0.0","reason":"Keep."}}`), &recorded)
	if err != nil || versions(imported.Snapshot)["techs/go/b"] != "1.0.0@1" || len(imported.Warnings) != 0 {
		t.Fatalf("versions %v, warnings %v, %v", versions(imported.Snapshot), imported.Warnings, err)
	}
}

// TestImport_RefToALibraryReleaseImportsWhatItPublished records each rule's version as that release lists it.
func TestImport_RefToALibraryReleaseImportsWhatItPublished(t *testing.T) {
	h := newHistory(t)
	imported, err := h.sync(t, h.source(t, `"groups":["techs/go"],"ref":"release/2"`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"techs/go/a": "1.1.0@2", "techs/go/b": "1.0.0@1", "techs/go/d": "1.0.0@2"}; !reflect.DeepEqual(versions(imported.Snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(imported.Snapshot), want)
	}
	if imported.Snapshot.Release != 2 || imported.Snapshot.Commit != h.commits[2] || len(imported.Warnings) != 0 {
		t.Fatalf("snapshot %+v, warnings %v", imported.Snapshot, imported.Warnings)
	}
	_, err = h.sync(t, h.source(t, `"groups":["techs/go"],"ref":"release/9"`), nil)
	requireCode(t, err, "version-not-found")
}

// TestImport_RefToAnUnreleasedCommitWarnsAndRecordsNoVersionForChangedRules matches the rest to published versions.
func TestImport_RefToAnUnreleasedCommitWarnsAndRecordsNoVersionForChangedRules(t *testing.T) {
	h := newHistory(t)
	commit, err := h.fixture.Commit(context.Background(), h.fixture.Worktree(), "Unreleased change", map[string][]byte{"techs/go/d.md": versionedRule("d unreleased")})
	if err != nil {
		t.Fatal(err)
	}
	config := h.source(t, `"groups":["techs/go"],"ref":"`+commit+`"`)
	imported, err := h.sync(t, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"techs/go/a": "2.0.0@3", "techs/go/d": "unreleased"}; !reflect.DeepEqual(versions(imported.Snapshot), want) {
		t.Fatalf("versions %v, want %v", versions(imported.Snapshot), want)
	}
	if imported.Snapshot.Release != 0 || imported.Snapshot.Commit != commit || imported.Snapshot.Rules["techs/go/d"].Commit != commit {
		t.Fatalf("snapshot %+v", imported.Snapshot)
	}
	if len(imported.Warnings) != 1 || !strings.Contains(imported.Warnings[0], "Source team imports "+commit) || !strings.HasSuffix(imported.Warnings[0], "Unreleased rules: techs/go/d.") {
		t.Fatalf("warnings %v", imported.Warnings)
	}
	// Removing the ref keeps published versions and gives unreleased rules their newest version.
	removed, err := h.sync(t, h.source(t, `"groups":["techs/go"]`), &imported.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"techs/go/a": "2.0.0@3", "techs/go/d": "1.0.0@2"}; !reflect.DeepEqual(versions(removed.Snapshot), want) || len(removed.Warnings) != 0 {
		t.Fatalf("versions %v, want %v; warnings %v", versions(removed.Snapshot), want, removed.Warnings)
	}
}

// TestImport_RefKeepsItsRecordedCommitAfterTheTagMoves restores the commit a tag named when it was recorded.
func TestImport_RefKeepsItsRecordedCommitAfterTheTagMoves(t *testing.T) {
	h := newHistory(t)
	ctx := context.Background()
	if _, err := h.fixture.Command(ctx, "tag", "candidate", h.commits[1]); err != nil {
		t.Fatal(err)
	}
	config := h.source(t, `"groups":["techs/go"],"ref":"candidate"`)
	first, err := h.sync(t, config, nil)
	if err != nil || first.Snapshot.Commit != h.commits[1] {
		t.Fatalf("snapshot %+v, %v", first.Snapshot, err)
	}
	if _, err := h.fixture.Command(ctx, "tag", "--force", "candidate", h.commits[3]); err != nil {
		t.Fatal(err)
	}
	again, err := h.sync(t, config, &first.Snapshot)
	if err != nil || again.Snapshot.Commit != h.commits[1] || !reflect.DeepEqual(versions(again.Snapshot), versions(first.Snapshot)) {
		t.Fatalf("snapshot %+v, %v", again.Snapshot, err)
	}
}

// TestImport_WithoutLibraryReleasesFailsUnlessARefNamesARevision reports releases-not-found.
func TestImport_WithoutLibraryReleasesFailsUnlessARefNamesARevision(t *testing.T) {
	f := newLibraryFixture(t, libraryFiles())
	config := libraryConfig(t, f.Repository)
	config.Sources[0].Ref, config.Sources[0].ParsedRef = "", nil
	_, err := ImportLibraries(context.Background(), config, nil, Options{GitPath: f.GitPath, Environment: f.Environment})
	requireCode(t, err, "releases-not-found")
}

// TestImport_RejectsAReleaseTagWithoutARecord refuses a lightweight or unparseable release/<number> tag.
func TestImport_RejectsAReleaseTagWithoutARecord(t *testing.T) {
	for name, tag := range map[string][]string{"lightweight": {"tag", "release/4"}, "no record": {"tag", "--annotate", "--message", "Notes only.", "release/4"}} {
		t.Run(name, func(t *testing.T) {
			h := newHistory(t)
			if _, err := h.fixture.Command(context.Background(), tag...); err != nil {
				t.Fatal(err)
			}
			_, err := h.sync(t, h.source(t, `"groups":["techs/go"]`), nil)
			requireCode(t, err, "invalid-release-tag")
		})
	}
}

// TestImport_IgnoresOtherTagsUnderRelease reads only tags named release/<number>.
func TestImport_IgnoresOtherTagsUnderRelease(t *testing.T) {
	h := newHistory(t)
	for _, name := range []string{"release/04", "release/next", "v2"} {
		if _, err := h.fixture.Command(context.Background(), "tag", name); err != nil {
			t.Fatal(err)
		}
	}
	imported, err := h.sync(t, h.source(t, `"groups":["techs/go"]`), nil)
	if err != nil || imported.Snapshot.Release != 3 {
		t.Fatalf("snapshot %+v, %v", imported.Snapshot, err)
	}
}

// TestImport_FailsWhenARecordedLibraryReleaseMoved refuses to mix a recorded commit with a moved tag.
func TestImport_FailsWhenARecordedLibraryReleaseMoved(t *testing.T) {
	h := newHistory(t)
	config := h.source(t, `"groups":["techs/go"]`)
	first, err := h.sync(t, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	object, err := h.fixture.Command(context.Background(), "cat-file", "tag", "release/2")
	if err != nil {
		t.Fatal(err)
	}
	_, message, _ := strings.Cut(object, "\n\n")
	if _, err := h.fixture.Command(context.Background(), "tag", "--force", "--annotate", "--cleanup=verbatim", "--message", message+"\n", "release/2", h.commits[3]); err != nil {
		t.Fatal(err)
	}
	_, err = h.sync(t, h.source(t, `"groups":["techs/go","practices/testing"]`), &first.Snapshot)
	requireCode(t, err, "invalid-release-tag")
}

// TestImport_ReleaseTagListingLimit refuses more release tags than the documented listing limit.
func TestImport_ReleaseTagListingLimit(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "git")
	listing := `awk 'BEGIN { for (i = 1; i <= 20001; i++) printf "%040d\trefs/tags/release/%d\n", i, i }'`
	body := "#!/bin/sh\nfor arg in \"$@\"; do\n  if [ \"$arg\" = ls-remote ]; then " + listing + "; exit 0; fi\n  if [ \"$arg\" = --version ]; then echo 'git version 2.45.0'; exit 0; fi\ndone\nexit 0\n"
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	repo, err := openRepository(context.Background(), rules.Source{Name: "team", Repository: "https://example.invalid/rules.git"}, Options{GitPath: script})
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	_, err = repo.loadHistory(context.Background())
	requireCode(t, err, "limit-exceeded")
}
