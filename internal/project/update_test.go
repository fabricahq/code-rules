// Exercise project update against a real library: previews that write nothing, decisions written to config.yaml
// with the output they select, exactly the previewed versions, and refusal after concurrent edits.

package project

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/rules"
)

// syncedProject syncs the sync fixture's project at release/1, then publishes release/2, in which
// techs/go/errors moves to 1.1.0 and techs/go/extra is new.
func syncedProject(t *testing.T) (Options, imports.Options, func(*testing.T)) {
	t.Helper()
	f, options, git := syncProject(t)
	if _, err := Sync(context.Background(), options, git); err != nil {
		t.Fatal(err)
	}
	secondRelease(t, f)
	thirdRelease := func(t *testing.T) {
		t.Helper()
		files := map[string][]byte{"techs/go/errors.md": []byte(strings.Replace(projectRule, "Return errors to the caller.", "Always return wrapped errors.", 1))}
		if _, err := f.Commit(context.Background(), f.Worktree(), "Third release", files); err != nil {
			t.Fatal(err)
		}
		record := "release: 3\nrules:\n  techs/go/errors: 2.0.0\n  techs/go/extra: 1.0.0\nchanges:\n  techs/go/errors: {change: major, from: 1.1.0, summary: Require wrapping.}\n"
		if err := f.Release(context.Background(), 3, record); err != nil {
			t.Fatal(err)
		}
	}
	return options, git, thirdRelease
}

// projectTree reads every file of the project's Code Rules directory.
func projectTree(t *testing.T, options Options) *filetxn.Tree {
	t.Helper()
	root, err := openProject(context.Background(), options, false)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	tree, err := filetxn.ReadTree(context.Background(), root, ".")
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// TestUpdate_PreviewWritesNothingAndApplyInstallsIt previews a minor change and a new rule, then applies both.
func TestUpdate_PreviewWritesNothingAndApplyInstallsIt(t *testing.T) {
	options, git, _ := syncedProject(t)
	before := projectTree(t, options)
	plan, err := PlanUpdate(context.Background(), options, git, nil)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := plan.Preview(nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := preview.Sources[0].Rules
	if preview.Applied || !preview.Moves() || len(rows) != 2 || rows[0].ID != "techs/go/errors" || rows[0].Change != imports.UpdateMinor || rows[1].ID != "techs/go/extra" || rows[1].Change != imports.UpdateNew {
		t.Fatalf("preview %+v", preview)
	}
	if after := projectTree(t, options); after.Digest() != before.Digest() {
		t.Fatal("planning and previewing changed the project")
	}
	applied, err := plan.Apply(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || slices.Contains(applied.Changed, "config.yaml") || !slices.Contains(applied.Changed, "vendor/team/techs/go/errors.md") || !slices.Contains(applied.Added, "vendor/team/techs/go/extra.md") {
		t.Fatalf("applied %+v", applied.FileChanges)
	}
	if _, got := recordedVersions(t, options); !reflect.DeepEqual(got, map[string]string{"techs/go/errors": "1.1.0@2", "techs/go/extra": "1.0.0@2"}) {
		t.Fatalf("versions %v", got)
	}
	requireCurrent(t, options)
}

// TestUpdate_DecisionsWritePinsAndExclusionsWithTheOutput keeps a changed rule with a pin and excludes the new
// rule, writing both to config.yaml in the same update.
func TestUpdate_DecisionsWritePinsAndExclusionsWithTheOutput(t *testing.T) {
	options, git, _ := syncedProject(t)
	plan, err := PlanUpdate(context.Background(), options, git, nil)
	if err != nil {
		t.Fatal(err)
	}
	decisions := []UpdateDecision{{Source: "team", Rule: "techs/go/errors", Keep: true, Reason: "Waiting on #45."}, {Source: "team", Rule: "techs/go/extra", Reason: "Not for this project."}}
	applied, err := plan.Apply(context.Background(), decisions)
	if err != nil {
		t.Fatal(err)
	}
	rows := applied.Sources[0].Rules
	if rows[0].Decision != "keep" || rows[0].Reason != "Waiting on #45." || rows[1].Decision != "exclude" || applied.Moves() != true || !slices.Contains(applied.Changed, "config.yaml") {
		t.Fatalf("applied %+v", applied)
	}
	tree := projectTree(t, options)
	config, err := rules.ParseConfigurationYAML(tree.Files["config.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	team := config.Sources[0]
	if team.Pins["techs/go/errors"].Version.String() != "1.0.0" || team.Pins["techs/go/errors"].Reason != "Waiting on #45." || team.Exclude["techs/go/extra"].Reason != "Not for this project." {
		t.Fatalf("configuration %s", tree.Files["config.yaml"])
	}
	if !strings.Contains(string(tree.Files["config.yaml"]), `version: "1.0.0"`) {
		t.Fatalf("pin version isn't quoted:\n%s", tree.Files["config.yaml"])
	}
	// The excluded rule is still imported so the project can review its changes, but agents don't read it.
	if _, got := recordedVersions(t, options); !reflect.DeepEqual(got, map[string]string{"techs/go/errors": "1.0.0@1", "techs/go/extra": "1.0.0@2"}) {
		t.Fatalf("versions %v", got)
	}
	if _, ok := tree.Files["generated/rules/team/techs/go/extra.md"]; ok {
		t.Fatal("generated an excluded rule")
	}
	requireCurrent(t, options)
}

// TestUpdate_AppliesThePreviewedVersionsAfterANewerLibraryRelease ignores a library release published after the
// preview.
func TestUpdate_AppliesThePreviewedVersionsAfterANewerLibraryRelease(t *testing.T) {
	options, git, thirdRelease := syncedProject(t)
	plan, err := PlanUpdate(context.Background(), options, git, nil)
	if err != nil {
		t.Fatal(err)
	}
	thirdRelease(t)
	if _, err := plan.Apply(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if _, got := recordedVersions(t, options); got["techs/go/errors"] != "1.1.0@2" {
		t.Fatalf("versions %v", got)
	}
}

// TestUpdate_RefusesAProjectChangedAfterThePreview writes nothing when configuration changed in between.
func TestUpdate_RefusesAProjectChangedAfterThePreview(t *testing.T) {
	options, git, _ := syncedProject(t)
	plan, err := PlanUpdate(context.Background(), options, git, nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := openProject(context.Background(), options, false)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	data, err := root.ReadFile(configurationFile)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, configurationFile, string(data)+"\n")
	before := projectTree(t, options)
	_, err = plan.Apply(context.Background(), nil)
	projectCode(t, err, "concurrent-change")
	if after := projectTree(t, options); after.Digest() != before.Digest() {
		t.Fatal("a refused update changed the project")
	}
}

// TestUpdate_RejectsDecisionsThePreviewDoesntOffer names the rule in each failure and writes nothing.
func TestUpdate_RejectsDecisionsThePreviewDoesntOffer(t *testing.T) {
	options, git, _ := syncedProject(t)
	plan, err := PlanUpdate(context.Background(), options, git, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := projectTree(t, options)
	for name, decisions := range map[string][]UpdateDecision{
		"keep a new rule":       {{Source: "team", Rule: "techs/go/extra", Keep: true, Reason: "No."}},
		"exclude a moved rule":  {{Source: "team", Rule: "techs/go/errors", Reason: "No."}},
		"unknown rule":          {{Source: "team", Rule: "techs/go/missing", Keep: true, Reason: "No."}},
		"blank reason":          {{Source: "team", Rule: "techs/go/errors", Keep: true, Reason: " "}},
		"two decisions":         {{Source: "team", Rule: "techs/go/errors", Keep: true, Reason: "A."}, {Source: "team", Rule: "techs/go/errors", Keep: true, Reason: "B."}},
		"another source's rule": {{Source: "other", Rule: "techs/go/errors", Keep: true, Reason: "No."}},
	} {
		t.Run(name, func(t *testing.T) {
			_, previewErr := plan.Preview(decisions)
			_, applyErr := plan.Apply(context.Background(), decisions)
			var validation *rules.ValidationError
			for _, err := range []error{previewErr, applyErr} {
				if !errors.As(err, &validation) || validation.Location != decisions[len(decisions)-1].Source+":"+decisions[len(decisions)-1].Rule {
					t.Fatalf("got %v", err)
				}
			}
		})
	}
	if after := projectTree(t, options); after.Digest() != before.Digest() {
		t.Fatal("a rejected decision changed the project")
	}
}
