// Exercise sync as a lockfile: restoring recorded versions, moving pinned rules, and offline checks of the result.

package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// secondRelease changes techs/go/errors to 1.1.0 and adds techs/go/extra in the sync fixture's library.
func secondRelease(t *testing.T, f *gitfixture.Fixture) {
	t.Helper()
	ctx := context.Background()
	files := map[string][]byte{"techs/go/errors.md": []byte(strings.Replace(projectRule, "Return errors to the caller.", "Return wrapped errors to the caller.", 1)), "techs/go/extra.md": []byte(projectRule)}
	if _, err := f.Commit(ctx, f.Worktree(), "Second release", files); err != nil {
		t.Fatal(err)
	}
	record := "formatVersion: 1\nrelease: 2\nrules:\n  techs/go/errors: 1.1.0\n  techs/go/extra: 1.0.0\nchanges:\n  techs/go/errors: {change: minor, from: 1.0.0, summaries: [Add wrapping.]}\n  techs/go/extra: {change: new, summaries: [Add the rule.]}\n"
	if err := f.Release(ctx, 2, record); err != nil {
		t.Fatal(err)
	}
}

// configure replaces the sync fixture's source fields, keeping its repository.
func configure(t *testing.T, options Options, f *gitfixture.Fixture, fields map[string]any) {
	t.Helper()
	fields["repository"] = f.Repository
	raw, _ := json.Marshal(map[string]any{"schemaVersion": 1, "sources": map[string]any{"team": fields}})
	root, err := openProject(context.Background(), options, false)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	writeFixture(t, root, configurationFile, string(raw))
}

// recordedVersions reads the project's source record and returns each rule's version@release.
func recordedVersions(t *testing.T, options Options) (snapshot, map[string]string) {
	t.Helper()
	root, err := openProject(context.Background(), options, false)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	state, err := readProject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	snapshots, err := decodeSnapshots(state.config, state.vendor.Files)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]string{}
	for id, rule := range snapshots["team"].Rules {
		result[id] = "unreleased"
		if rule.Version != nil {
			result[id] = rule.Version.String() + "@" + strconv.Itoa(rule.Release)
		}
	}
	return snapshots["team"], result
}

// requireCurrent checks offline, with no Git on PATH, that generated output matches the synced inputs.
func requireCurrent(t *testing.T, options Options) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	checked, err := Check(context.Background(), options)
	if err != nil || !checked.Current() {
		t.Fatalf("offline check: %+v %v", checked, err)
	}
}

// TestSync_RestoresRecordedVersionsAndMovesOnlyPinnedRules follows the lockfile across a new library release.
func TestSync_RestoresRecordedVersionsAndMovesOnlyPinnedRules(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	secondRelease(t, f)
	restored, err := Sync(ctx, options, git)
	if err != nil || len(restored.Added)+len(restored.Changed)+len(restored.Removed) != 0 {
		t.Fatalf("sync after a new library release changed %+v, %v", restored, err)
	}
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "pins": map[string]any{"techs/go/errors": map[string]string{"version": "1.1.0", "reason": "Adopt wrapping."}}})
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if _, got := recordedVersions(t, options); len(got) != 1 || got["techs/go/errors"] != "1.1.0@2" {
		t.Fatalf("pinned up: %v", got)
	}
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "rules": []string{"techs/go/extra"}, "pins": map[string]any{"techs/go/errors": map[string]string{"version": "1.0.0", "reason": "Not ready."}}})
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	record, got := recordedVersions(t, options)
	if len(got) != 2 || got["techs/go/errors"] != "1.0.0@1" || got["techs/go/extra"] != "1.0.0@2" || record.Release != 2 {
		t.Fatalf("pinned down with a new rule: %v, release %d", got, record.Release)
	}
	if string(record.Files["techs/go/errors.md"]) != projectRule {
		t.Fatalf("older rule's version 1.0.0 isn't stored at its library path: %q", record.Files["techs/go/errors.md"])
	}
	requireCurrent(t, options)
}

// TestSync_ExplicitSelectionImportsAGroupTheWildcardSnapshotLacked imports the rules of a group that a library
// release added after a wildcard sync, once the selection names it explicitly, while recorded rules keep their
// versions, and offline check agrees with the result.
func TestSync_ExplicitSelectionImportsAGroupTheWildcardSnapshotLacked(t *testing.T) {
	for _, test := range []struct {
		name   string
		groups []string
		want   map[string]string
	}{
		{"only the new group", []string{"practices/testing"}, map[string]string{"practices/testing/verify": "1.0.0@2"}},
		{"the new group and a recorded one", []string{"practices/testing", "techs/go"}, map[string]string{"techs/go/errors": "1.0.0@1", "practices/testing/verify": "1.0.0@2"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f, options, git := syncProject(t)
			ctx := context.Background()
			configure(t, options, f, map[string]any{"groups": "*"})
			if _, err := Sync(ctx, options, git); err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{
				"techs/go/errors.md":            []byte(strings.Replace(projectRule, "Return errors to the caller.", "Return wrapped errors to the caller.", 1)),
				"practices/testing/_group.yaml": []byte(`{"name":"Testing","description":"Testing guidance.","whenToRead":"When testing."}`),
				"practices/testing/verify.md":   []byte(projectRule),
			}
			if _, err := f.Commit(ctx, f.Worktree(), "Second release", files); err != nil {
				t.Fatal(err)
			}
			if err := f.Release(ctx, 2, "formatVersion: 1\nrelease: 2\nrules:\n  techs/go/errors: 1.1.0\n  practices/testing/verify: 1.0.0\nchanges:\n  techs/go/errors: {change: minor, from: 1.0.0, summaries: [Add wrapping.]}\n  practices/testing/verify: {change: new, summaries: [Add the rule.]}\n"); err != nil {
				t.Fatal(err)
			}
			configure(t, options, f, map[string]any{"groups": test.groups})
			if _, err := Sync(ctx, options, git); err != nil {
				t.Fatal(err)
			}
			record, got := recordedVersions(t, options)
			if !maps.Equal(got, test.want) || !slices.Equal(record.Groups, test.groups) {
				t.Fatalf("versions %v in groups %v, want %v in %v", got, record.Groups, test.want, test.groups)
			}
			requireCurrent(t, options)
		})
	}
}

// localRuleProject syncs the sync fixture's project with a local rule in techs/go, whose metadata only the team
// source supplies, and returns its root.
func localRuleProject(t *testing.T) (*os.Root, Options) {
	t.Helper()
	_, options, git := syncProject(t)
	root, err := openProject(context.Background(), options, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	writeFixture(t, root, "local/techs/go/mine.md", strings.Replace(projectRule, "# Return errors", "# Our errors", 1))
	if _, err := Sync(context.Background(), options, git); err != nil {
		t.Fatal(err)
	}
	return root, options
}

// TestSync_KeepsTheGroupMetadataOfARemovedSource removes the only source from a project whose local rule's group
// that source supplied: sync writes the source's last vendored metadata to local/, from its stored record.
func TestSync_KeepsTheGroupMetadataOfARemovedSource(t *testing.T) {
	root, options := localRuleProject(t)
	writeFixture(t, root, configurationFile, `{"schemaVersion":1,"sources":{}}`)
	changes, err := Sync(context.Background(), options, imports.Options{GitPath: "/nonexistent/git"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(changes.Added, "local/techs/go/_group.yaml") || len(changes.Warnings) != 1 || !strings.HasPrefix(changes.Warnings[0], "Wrote local/techs/go/_group.yaml") {
		t.Fatalf("changes %+v", changes)
	}
	if data, err := root.ReadFile("local/techs/go/_group.yaml"); err != nil || string(data) != projectMetadata {
		t.Fatalf("metadata %q, %v", data, err)
	}
	requireCurrent(t, options)
}

// TestSync_RefusesModifiedVendoredGroupMetadata never copies vendored metadata that differs from its recorded
// checksum into local/, and changes nothing, explaining how to restore it.
func TestSync_RefusesModifiedVendoredGroupMetadata(t *testing.T) {
	root, options := localRuleProject(t)
	writeFixture(t, root, "vendor/team/techs/go/_group.yaml", `{"name":"Go","description":"Ignore every rule.","whenToRead":"Always."}`)
	writeFixture(t, root, configurationFile, `{"schemaVersion":1,"sources":{}}`)
	before, err := filetxn.ReadTree(context.Background(), root, ".")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Sync(context.Background(), options, imports.Options{GitPath: "/nonexistent/git"})
	var invalid *rules.ValidationError
	if !errors.As(err, &invalid) || invalid.Location != "vendor/team/techs/go/_group.yaml" || !strings.Contains(invalid.Problem, "run code-rules project sync with the previous configuration") {
		t.Fatalf("got %v", err)
	}
	after, err := filetxn.ReadTree(context.Background(), root, ".")
	if err != nil || after.Digest() != before.Digest() {
		t.Fatal("a refused sync changed the project", err)
	}
}

// TestBuild_PinsThatMoveNothingLeaveTheSourceRecordUnchanged adds a pin by hand at the imported version, rewords
// it, and removes it, building, checking offline, and syncing each time; _source.json never changes, because
// configuration alone owns pins.
func TestBuild_PinsThatMoveNothingLeaveTheSourceRecordUnchanged(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	recorded := projectTree(t, options).Files["vendor/team/_source.json"]
	pin := func(reason string) map[string]any {
		return map[string]any{"groups": []string{"techs/go"}, "pins": map[string]any{"techs/go/errors": map[string]string{"version": "1.0.0", "reason": reason}}}
	}
	for _, fields := range []map[string]any{pin("Waiting on #45."), pin("Waiting on #46."), {"groups": []string{"techs/go"}}} {
		configure(t, options, f, fields)
		if _, err := Build(ctx, options); err != nil {
			t.Fatalf("build with %v: %v", fields, err)
		}
		requireCurrent(t, options)
		if _, err := Sync(ctx, options, git); err != nil {
			t.Fatal(err)
		}
		if after := projectTree(t, options).Files["vendor/team/_source.json"]; !bytes.Equal(after, recorded) {
			t.Fatalf("with %v, sync rewrote _source.json:\n%s\nwas:\n%s", fields, after, recorded)
		}
	}
}

// TestSync_RecordsTheRetirementOfANewlyPinnedRule the record didn't list, as it does for an exclusion: release/2
// adds extra and release/3 retires it after the project synced release/1; pinning extra warns, and sync records
// the retirement so offline checks accept the pin, and a second sync changes nothing.
func TestSync_RecordsTheRetirementOfANewlyPinnedRule(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	secondRelease(t, f)
	if _, err := f.Commit(ctx, f.Worktree(), "Retire extra", map[string][]byte{"techs/go/extra.md": nil}); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 3, "formatVersion: 1\nrelease: 3\nrules:\n  techs/go/errors: 1.1.0\nretired:\n  techs/go/extra: {lastVersion: 1.0.0, summaries: [No longer needed.]}\n"); err != nil {
		t.Fatal(err)
	}
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "pins": map[string]any{"techs/go/extra": map[string]string{"version": "1.0.0", "reason": "Keep it."}}})
	changes, err := Sync(ctx, options, git)
	if err != nil || len(changes.Warnings) != 1 || !strings.Contains(changes.Warnings[0], "techs/go/extra") {
		t.Fatalf("sync: %+v, %v", changes, err)
	}
	if record, _ := recordedVersions(t, options); !slices.Equal(record.RetiredRules, []string{"techs/go/extra"}) || record.Release != 1 {
		t.Fatalf("retired %v, shared files from release %d", record.RetiredRules, record.Release)
	}
	requireCurrent(t, options)
	recorded := projectTree(t, options).Files["vendor/team/_source.json"]
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if after := projectTree(t, options).Files["vendor/team/_source.json"]; !bytes.Equal(after, recorded) {
		t.Fatalf("a second sync rewrote _source.json:\n%s\nwas:\n%s", after, recorded)
	}
}

// TestSync_KeepsAPinnedRetiredRuleAtItsLastVersion: the project imports errors 1.1.0, the library then retires it,
// and a pin at 1.1.0 keeps it installed through sync, build, and an offline check, without changing the record.
func TestSync_KeepsAPinnedRetiredRuleAtItsLastVersion(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	secondRelease(t, f)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Commit(ctx, f.Worktree(), "Retire errors", map[string][]byte{"techs/go/errors.md": nil, "techs/go/assets/errors/data.bin": nil}); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 3, "formatVersion: 1\nrelease: 3\nrules:\n  techs/go/extra: 1.0.0\nretired:\n  techs/go/errors: {lastVersion: 1.1.0, summaries: [Covered by extra.]}\n"); err != nil {
		t.Fatal(err)
	}
	recorded := projectTree(t, options).Files["vendor/team/_source.json"]
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "pins": map[string]any{"techs/go/errors": map[string]string{"version": "1.1.0", "reason": "Keep it."}}})
	if _, err := Build(ctx, options); err != nil {
		t.Fatal(err)
	}
	requireCurrent(t, options)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if _, versions := recordedVersions(t, options); versions["techs/go/errors"] != "1.1.0@2" {
		t.Fatalf("versions %v", versions)
	}
	if after := projectTree(t, options).Files["vendor/team/_source.json"]; !bytes.Equal(after, recorded) {
		t.Fatalf("pinning the retired rule rewrote _source.json:\n%s\nwas:\n%s", after, recorded)
	}
	requireCurrent(t, options)
}

// TestSync_KeepsRecordedRetirementsWhenOnlyAnExclusionChanged: the record lists retired errors and imports extra;
// the library then retires extra; an exclusion of errors, added by hand, builds and checks offline, and a sync that
// changes nothing else, although checking the exclusion reads the library's releases, writes _source.json byte for
// byte as before, extra's retirement included only when an update moves the source.
func TestSync_KeepsRecordedRetirementsWhenOnlyAnExclusionChanged(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	secondRelease(t, f)
	if _, err := f.Commit(ctx, f.Worktree(), "Retire errors", map[string][]byte{"techs/go/errors.md": nil, "techs/go/assets/errors/data.bin": nil}); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 3, "formatVersion: 1\nrelease: 3\nrules:\n  techs/go/extra: 1.0.0\nretired:\n  techs/go/errors: {lastVersion: 1.1.0, summaries: [Covered by extra.]}\n"); err != nil {
		t.Fatal(err)
	}
	root, err := openProject(ctx, options, false)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.RemoveAll("vendor"); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if record, versions := recordedVersions(t, options); !slices.Equal(record.RetiredRules, []string{"techs/go/errors"}) || versions["techs/go/extra"] != "1.0.0@2" {
		t.Fatalf("retired %v, versions %v", record.RetiredRules, versions)
	}
	if _, err := f.Commit(ctx, f.Worktree(), "Retire extra", map[string][]byte{"techs/go/extra.md": nil}); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 4, "formatVersion: 1\nrelease: 4\nrules: {}\nretired:\n  techs/go/extra: {lastVersion: 1.0.0, summaries: [No longer needed.]}\n"); err != nil {
		t.Fatal(err)
	}
	recorded := projectTree(t, options).Files["vendor/team/_source.json"]
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "exclude": map[string]any{"techs/go/errors": map[string]string{"reason": "Retired upstream."}}})
	if _, err := Build(ctx, options); err != nil {
		t.Fatal(err)
	}
	requireCurrent(t, options)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if after := projectTree(t, options).Files["vendor/team/_source.json"]; !bytes.Equal(after, recorded) {
		t.Fatalf("sync rewrote _source.json:\n%s\nwas:\n%s", after, recorded)
	}
}

// TestSync_RecordsTheRetirementOfANewlyExcludedRule the record didn't list: release/2 adds extra and release/3
// retires it after the project synced release/1; excluding extra without updating warns, and sync records the
// retirement so offline checks accept the exclusion, and a second sync changes nothing.
func TestSync_RecordsTheRetirementOfANewlyExcludedRule(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	secondRelease(t, f)
	if _, err := f.Commit(ctx, f.Worktree(), "Retire extra", map[string][]byte{"techs/go/extra.md": nil}); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 3, "formatVersion: 1\nrelease: 3\nrules:\n  techs/go/errors: 1.1.0\nretired:\n  techs/go/extra: {lastVersion: 1.0.0, summaries: [No longer needed.]}\n"); err != nil {
		t.Fatal(err)
	}
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "exclude": map[string]any{"techs/go/extra": map[string]string{"reason": "Retired upstream."}}})
	changes, err := Sync(ctx, options, git)
	if err != nil || len(changes.Warnings) != 1 || !strings.Contains(changes.Warnings[0], "techs/go/extra") {
		t.Fatalf("sync: %+v, %v", changes, err)
	}
	if record, _ := recordedVersions(t, options); !slices.Equal(record.RetiredRules, []string{"techs/go/extra"}) || record.Release != 1 {
		t.Fatalf("retired %v, shared files from release %d", record.RetiredRules, record.Release)
	}
	requireCurrent(t, options)
	recorded := projectTree(t, options).Files["vendor/team/_source.json"]
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if after := projectTree(t, options).Files["vendor/team/_source.json"]; !bytes.Equal(after, recorded) {
		t.Fatalf("a second sync rewrote _source.json:\n%s\nwas:\n%s", after, recorded)
	}
}

// TestBuild_HandEditedExclusionsKeepTheSourceRecordCurrent adds and removes exclusions by hand, of an imported rule
// and of a retired one, and runs build, as a fork's next step says: check passes offline, and a sync afterward
// writes _source.json byte for byte as before.
func TestBuild_HandEditedExclusionsKeepTheSourceRecordCurrent(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	secondRelease(t, f)
	if _, err := f.Commit(ctx, f.Worktree(), "Retire extra", map[string][]byte{"techs/go/extra.md": nil}); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 3, "formatVersion: 1\nrelease: 3\nrules:\n  techs/go/errors: 1.1.0\nchanges: {}\nretired:\n  techs/go/extra: {lastVersion: 1.0.0, summaries: [Covered by errors.]}\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	for _, exclude := range []map[string]any{
		{"techs/go/errors": map[string]string{"reason": "Ours is stricter.", "replacedBy": "local/techs/go/errors.md"}},
		{"techs/go/extra": map[string]string{"reason": "Retired upstream."}},
		{},
	} {
		recorded := projectTree(t, options).Files["vendor/team/_source.json"]
		configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "exclude": exclude})
		if _, ok := exclude["techs/go/errors"]; ok {
			root, err := openProject(ctx, options, false)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, "local/techs/go/errors.md", projectRule)
			writeFixture(t, root, "local/techs/go/_group.yaml", projectMetadata)
			root.Close()
		}
		if _, err := Build(ctx, options); err != nil {
			t.Fatalf("build with exclusions %v: %v", exclude, err)
		}
		requireCurrent(t, options)
		if _, err := Sync(ctx, options, git); err != nil {
			t.Fatal(err)
		}
		if after := projectTree(t, options).Files["vendor/team/_source.json"]; !bytes.Equal(after, recorded) {
			t.Fatalf("with exclusions %v, sync rewrote _source.json:\n%s\nwas:\n%s", exclude, after, recorded)
		}
	}
}

// TestSync_RefToAnUnreleasedCommitWarnsAndStillChecks reports the source and its unreleased rules.
func TestSync_RefToAnUnreleasedCommitWarnsAndStillChecks(t *testing.T) {
	f, options, git := syncProject(t)
	commit, err := f.Commit(context.Background(), f.Worktree(), "Unreleased", map[string][]byte{"techs/go/errors.md": []byte(projectRule + "\nMore.\n")})
	if err != nil {
		t.Fatal(err)
	}
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "ref": commit})
	changes, err := Sync(context.Background(), options, git)
	if err != nil || len(changes.Warnings) != 1 || !strings.Contains(changes.Warnings[0], "Unreleased rules: techs/go/errors.") {
		t.Fatalf("warnings %v, %v", changes.Warnings, err)
	}
	if _, got := recordedVersions(t, options); got["techs/go/errors"] != "unreleased" {
		t.Fatalf("versions %v", got)
	}
	requireCurrent(t, options)
}

// TestSync_RefusesAFormatOneRecord asks the user to delete vendor/ and sync, without changing files.
func TestSync_RefusesAFormatOneRecord(t *testing.T) {
	_, options, git := syncProject(t)
	root, err := openProject(context.Background(), options, false)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	writeFixture(t, root, "vendor/team/_source.json", `{"formatVersion":1,"repository":"x","ref":"v1.0.0","resolvedCommit":"x","groups":[],"files":{}}`)
	before, err := filetxn.ReadTree(context.Background(), root, ".")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Sync(context.Background(), options, git)
	var validation *rules.ValidationError
	if !errors.As(err, &validation) || !strings.Contains(err.Error(), "delete .code-rules/vendor/ and run code-rules project sync") {
		t.Fatalf("got %v", err)
	}
	after, err := filetxn.ReadTree(context.Background(), root, ".")
	if err != nil || before.Digest() != after.Digest() {
		t.Fatal("refused sync changed project", err)
	}
}

// TestProject_TellsWhetherToUpgradeOrResyncForAnotherSourceRecordFormat asks to upgrade Code Rules for a
// _source.json a newer Code Rules wrote, whose deletion would downgrade the project, and to delete vendor/ and sync
// for an older one, in offline build and check as in sync and update.
func TestProject_TellsWhetherToUpgradeOrResyncForAnotherSourceRecordFormat(t *testing.T) {
	operations := map[string]func(Options, imports.Options) error{
		"build": func(o Options, _ imports.Options) error { _, err := Build(context.Background(), o); return err },
		"check": func(o Options, _ imports.Options) error { _, err := Check(context.Background(), o); return err },
		"sync":  func(o Options, g imports.Options) error { _, err := Sync(context.Background(), o, g); return err },
		"update": func(o Options, g imports.Options) error {
			_, err := PlanUpdate(context.Background(), o, g, nil)
			return err
		},
	}
	for name, operation := range operations {
		for _, format := range []int{1, 3} {
			t.Run(name+"/format "+strconv.Itoa(format), func(t *testing.T) {
				_, options, git := syncProject(t)
				if _, err := Sync(context.Background(), options, git); err != nil {
					t.Fatal(err)
				}
				root, err := openProject(context.Background(), options, false)
				if err != nil {
					t.Fatal(err)
				}
				defer root.Close()
				record, err := root.ReadFile("vendor/team/_source.json")
				if err != nil {
					t.Fatal(err)
				}
				// A newer format can also hold fields and values this version can't decode, such as a number no
				// Go type holds.
				replacement := `"formatVersion": ` + strconv.Itoa(format)
				if format > 2 {
					replacement += `, "future": {"limit": 1e999, "rules": "another shape"}`
				}
				writeFixture(t, root, "vendor/team/_source.json", strings.Replace(string(record), `"formatVersion": 2`, replacement, 1))
				err = operation(options, git)
				var failure *filetxn.Error
				var invalid *rules.ValidationError
				switch {
				case format > 2 && (!errors.As(err, &failure) || failure.Code != "unsupported-source-record" || !strings.Contains(err.Error(), "upgrade Code Rules") || strings.Contains(err.Error(), "delete")):
					t.Fatalf("got %v; want unsupported-source-record asking to upgrade", err)
				case format < 2 && (!errors.As(err, &invalid) || !strings.Contains(err.Error(), "delete .code-rules/vendor/ and run code-rules project sync")):
					t.Fatalf("got %v; want the advice to delete vendor/ and sync", err)
				}
			})
		}
	}
}

// TestSync_ReportsCodesForMissingLibraryReleases surfaces releases-not-found before the first library release.
func TestSync_ReportsCodesForMissingLibraryReleases(t *testing.T) {
	f, err := gitfixture.New(context.Background(), map[string][]byte{"rule-library.yaml": []byte(`{"formatVersion":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	_, options, _ := syncProject(t)
	configure(t, options, f, map[string]any{"groups": "*"})
	_, err = Sync(context.Background(), options, imports.Options{GitPath: f.GitPath, Environment: f.Environment})
	var failure *imports.Error
	if !errors.As(err, &failure) || failure.Code != "releases-not-found" {
		t.Fatalf("got %v", err)
	}
}

// TestSync_StoresOlderRulesAtTheirLibraryPaths keeps an older rule's attachment beside the shared files it links
// to, so relative links in the vendored copy still resolve.
func TestSync_StoresOlderRulesAtTheirLibraryPaths(t *testing.T) {
	ctx := context.Background()
	f, err := gitfixture.New(ctx, map[string][]byte{
		"rule-library.yaml":               []byte(`{"formatVersion":1}`),
		"techs/go/_group.yaml":            []byte(projectMetadata),
		"techs/go/errors.md":              []byte(projectRule + "\n[Guide](assets/errors/guide.md)\n"),
		"techs/go/assets/errors/guide.md": []byte("[Shared](../../../../assets/shared.md)\n"),
		"assets/shared.md":                []byte("Shared explanation.\n"),
		"techs/go/naming.md":              []byte(projectRule),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if err := f.Release(ctx, 1, "formatVersion: 1\nrelease: 1\nrules:\n  techs/go/errors: 1.0.0\n  techs/go/naming: 1.0.0\nchanges:\n  techs/go/errors: {change: new, summaries: [Add the rule.]}\n  techs/go/naming: {change: new, summaries: [Add the rule.]}\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Commit(ctx, f.Worktree(), "Second release", map[string][]byte{"techs/go/naming.md": []byte(projectRule + "\nMore.\n")}); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 2, "formatVersion: 1\nrelease: 2\nrules:\n  techs/go/errors: 1.0.0\n  techs/go/naming: 1.1.0\nchanges:\n  techs/go/naming: {change: minor, from: 1.0.0, summaries: [Add more.]}\n"); err != nil {
		t.Fatal(err)
	}
	_, options, _ := syncProject(t)
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}})
	if _, err := Sync(ctx, options, imports.Options{GitPath: f.GitPath, Environment: f.Environment}); err != nil {
		t.Fatal(err)
	}
	record, versions := recordedVersions(t, options)
	if versions["techs/go/errors"] != "1.0.0@1" || record.Release != 2 {
		t.Fatalf("versions %v, release %d", versions, record.Release)
	}
	for _, file := range []string{"techs/go/errors.md", "techs/go/assets/errors/guide.md", "assets/shared.md"} {
		if _, ok := record.Files[file]; !ok {
			t.Errorf("%s isn't stored at its library path: %v", file, slices.Sorted(maps.Keys(record.Files)))
		}
	}
	requireCurrent(t, options)
}

// TestSync_RefusesARecordedVersionOrReleaseTheLibraryDoesntHave, as a hand edit or a merge resolution can leave in
// _source.json: a rule's version that its library release doesn't publish, a rule's library release, or the
// shared files' library release, that the library doesn't have. Sync writes nothing, and syncs again once the record
// is restored.
func TestSync_RefusesARecordedVersionOrReleaseTheLibraryDoesntHave(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	secondRelease(t, f)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	recorded := string(projectTree(t, options).Files["vendor/team/_source.json"])
	for _, test := range []struct{ name, from, to, location, problem string }{
		{"version", `"version": "1.1.0"`, `"version": "1.1.1"`, "vendor/team/_source.json.rules.techs/go/errors", "records version 1.1.1 from library release 2, which publishes version 1.1.0 of the rule"},
		{"rule's release", "\"version\": \"1.1.0\",\n      \"release\": 2", "\"version\": \"1.1.0\",\n      \"release\": 3", "vendor/team/_source.json.rules.techs/go/errors", "records library release 3, which the library doesn't have"},
		{"shared files' release", "\"release\": 2,\n  \"resolvedCommit\"", "\"release\": 99,\n  \"resolvedCommit\"", "vendor/team/_source.json.release", "records library release 99, which the library doesn't have"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !strings.Contains(recorded, test.from) {
				t.Fatalf("the record lacks %q:\n%s", test.from, recorded)
			}
			root, err := openProject(ctx, options, false)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, "vendor/team/_source.json", strings.Replace(recorded, test.from, test.to, 1))
			root.Close()
			requireEditedRecord(t, options)
			before := projectTree(t, options)
			_, err = Sync(ctx, options, git)
			var invalid *rules.ValidationError
			if !errors.As(err, &invalid) || invalid.Location != test.location || !strings.Contains(invalid.Problem, test.problem) || !strings.Contains(invalid.Problem, "restore vendor/team/_source.json") {
				t.Fatalf("got %v", err)
			}
			if after := projectTree(t, options); after.Digest() != before.Digest() {
				t.Fatal("a refused sync changed the project")
			}
			root, err = openProject(ctx, options, false)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, "vendor/team/_source.json", recorded)
			root.Close()
			if _, err := Sync(ctx, options, git); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestBuild_AnEquivalentRefSpellingLeavesTheSourceRecordUnchanged: changing ref: release/1 to refs/tags/release/1
// names the same revision, so build and an offline check pass, and sync keeps the recorded spelling byte for byte.
func TestBuild_AnEquivalentRefSpellingLeavesTheSourceRecordUnchanged(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "ref": "release/1"})
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	recorded := projectTree(t, options).Files["vendor/team/_source.json"]
	configure(t, options, f, map[string]any{"groups": []string{"techs/go"}, "ref": "refs/tags/release/1"})
	if _, err := Build(ctx, options); err != nil {
		t.Fatal(err)
	}
	requireCurrent(t, options)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if after := projectTree(t, options).Files["vendor/team/_source.json"]; !bytes.Equal(after, recorded) {
		t.Fatalf("sync rewrote _source.json:\n%s\nwas:\n%s", after, recorded)
	}
	requireCurrent(t, options)
}

// requireEditedRecord requires offline build and check to refuse team's source record as changed outside sync,
// pointing to sync, never to build.
func requireEditedRecord(t *testing.T, options Options) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	_, buildErr := Build(context.Background(), options)
	_, checkErr := Check(context.Background(), options)
	for name, err := range map[string]error{"build": buildErr, "check": checkErr} {
		var invalid *rules.ValidationError
		if !errors.As(err, &invalid) || invalid.Location != "vendor/team/_source.json" || !strings.Contains(invalid.Problem, "changed outside code-rules project sync") || !strings.Contains(invalid.Problem, "run code-rules project sync") || strings.Contains(invalid.Problem, "project build") {
			t.Errorf("offline %s: %v", name, err)
		}
	}
}

// TestSync_RecordsAgainARecordWithoutItsChecksum, which offline build and check refuse as changed outside sync,
// restoring the record byte for byte, since the library confirms what it records.
func TestSync_RecordsAgainARecordWithoutItsChecksum(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	secondRelease(t, f)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	recorded := string(projectTree(t, options).Files["vendor/team/_source.json"])
	start := strings.Index(recorded, "  \"checksum\": ")
	if start < 0 {
		t.Fatalf("the record has no checksum:\n%s", recorded)
	}
	end := start + strings.Index(recorded[start:], "\n") + 1
	root, err := openProject(ctx, options, false)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "vendor/team/_source.json", recorded[:start]+recorded[end:])
	root.Close()
	requireEditedRecord(t, options)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if after := string(projectTree(t, options).Files["vendor/team/_source.json"]); after != recorded {
		t.Fatalf("sync wrote:\n%s\nwant:\n%s", after, recorded)
	}
	requireCurrent(t, options)
}

// TestSync_SaysItRaisesARecordedSharedFilesReleaseOlderThanItsRules: a record edited to take the shared files from
// library release 1, while errors comes from release 2, is repaired, with a warning saying so.
func TestSync_SaysItRaisesARecordedSharedFilesReleaseOlderThanItsRules(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	secondRelease(t, f)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	recorded := string(projectTree(t, options).Files["vendor/team/_source.json"])
	root, err := openProject(ctx, options, false)
	if err != nil {
		t.Fatal(err)
	}
	one, err := f.Command(ctx, "rev-parse", "release/1^{commit}")
	if err != nil {
		t.Fatal(err)
	}
	record, _ := recordedVersions(t, options)
	from := "\"release\": 2,\n  \"resolvedCommit\": \"" + record.Commit
	if !strings.Contains(recorded, from) {
		t.Fatalf("the record lacks %q:\n%s", from, recorded)
	}
	writeFixture(t, root, "vendor/team/_source.json", strings.Replace(recorded, from, "\"release\": 1,\n  \"resolvedCommit\": \""+strings.TrimSpace(one), 1))
	root.Close()
	changes, err := Sync(ctx, options, git)
	if err != nil || !slices.ContainsFunc(changes.Warnings, func(w string) bool {
		return strings.Contains(w, "vendor/team/_source.json recorded library release 1 for the shared files, older than library release 2")
	}) {
		t.Fatalf("sync: %+v, %v", changes, err)
	}
	if record, _ := recordedVersions(t, options); record.Release != 2 {
		t.Fatalf("shared files from release %d", record.Release)
	}
}

// TestSourceRecord_AnUnreadableRecordSaysHowToImportTheSourceAgain, and what that does to unpinned rules.
func TestSourceRecord_AnUnreadableRecordSaysHowToImportTheSourceAgain(t *testing.T) {
	_, options, git := syncProject(t)
	ctx := context.Background()
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	recorded := string(projectTree(t, options).Files["vendor/team/_source.json"])
	root, err := openProject(ctx, options, false)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "vendor/team/_source.json", strings.Replace(recorded, "{\n", "{\n  \"pins\": {},\n", 1))
	root.Close()
	_, err = Sync(ctx, options, git)
	var invalid *rules.ValidationError
	if !errors.As(err, &invalid) || invalid.Location != "vendor/team/_source.json.pins" || !strings.Contains(invalid.Problem, "delete .code-rules/vendor/team/ and run code-rules project sync") || !strings.Contains(invalid.Problem, "newest versions") {
		t.Fatalf("got %v", err)
	}
}

// TestCheck_NamesAFileWithUnresolvedMergeConflicts, config.yaml or a source record, instead of failing to parse it.
func TestCheck_NamesAFileWithUnresolvedMergeConflicts(t *testing.T) {
	_, options, git := syncProject(t)
	ctx := context.Background()
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	tree := projectTree(t, options)
	conflicted := func(text string) string {
		return "<<<<<<< HEAD\n" + text + "=======\n" + text + ">>>>>>> theirs\n"
	}
	for _, test := range []struct{ file, location string }{
		{configurationFile, ".code-rules/config.yaml"},
		{"vendor/team/_source.json", "vendor/team/_source.json"},
	} {
		root, err := openProject(ctx, options, false)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, test.file, conflicted(string(tree.Files[test.file])))
		_, err = Check(ctx, options)
		var invalid *rules.ValidationError
		if !errors.As(err, &invalid) || invalid.Location != test.location || !strings.Contains(invalid.Problem, "unresolved merge conflicts") {
			t.Errorf("%s: %v", test.file, err)
		}
		writeFixture(t, root, test.file, string(tree.Files[test.file]))
		root.Close()
	}
}
