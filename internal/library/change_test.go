// Record change notes against real library history, and check that the notes satisfy library check.

package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fabricahq/code-rules/internal/rules"
)

// noteDay is the calendar day tests record notes on.
var noteDay = time.Date(2026, time.September, 29, 23, 30, 0, 0, time.UTC)

// recordChange plans and commits a note, returning its path relative to the library and its contents.
func recordChange(t *testing.T, options Options, request ChangeRequest) (string, string, error) {
	t.Helper()
	plan, err := PlanChange(context.Background(), request, options)
	if err != nil {
		return "", "", err
	}
	result, err := plan.Commit(context.Background(), request.Bump, request.Summary, noteDay)
	if err != nil {
		return "", "", err
	}
	if len(result.Files) != 1 {
		t.Fatalf("wrote %q, want one note", result.Files)
	}
	name, err := filepath.Rel(options.Directory, result.Files[0])
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.Files[0])
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(name), string(data), nil
}

// TestChange_WritesNotesThatLibraryCheckAccepts records each kind of change and checks the pending release.
func TestChange_WritesNotesThatLibraryCheckAccepts(t *testing.T) {
	for _, test := range []struct {
		name    string
		files   map[string]string
		request ChangeRequest
		note    string
		want    string
		rows    []string
	}{
		{name: "version change", files: map[string]string{"practices/testing/a.md": ruleText("Changed.")},
			request: ChangeRequest{IDs: []string{"practices/testing/a"}, Bump: rules.ChangeMinor, Summary: "Add a Python example of the retry-limit test."},
			note:    "changes/2026-09-29-a.yaml", want: "summary: Add a Python example of the retry-limit test.\nrules:\n  practices/testing/a: minor\n",
			rows: []string{"practices/testing/a minor 1.0.0 1.1.0"}},
		{name: "several rules", files: map[string]string{"practices/testing/a.md": ruleText("Changed."), "practices/testing/b.md": ruleText("Changed too.")},
			request: ChangeRequest{IDs: []string{"practices/testing/b", "practices/testing/a"}, Bump: rules.ChangeMajor, Summary: "Require a test at every limit."},
			note:    "changes/2026-09-29-b.yaml", want: "summary: Require a test at every limit.\nrules:\n  practices/testing/a: major\n  practices/testing/b: major\n",
			rows: []string{"practices/testing/a major 1.0.0 2.0.0", "practices/testing/b major 1.0.0 2.0.0"}},
		{name: "new rule", files: map[string]string{"practices/testing/verify-backoff.md": ruleText("New.")},
			request: ChangeRequest{IDs: []string{"practices/testing/verify-backoff"}, Summary: "Add a rule about testing retry backoff."},
			note:    "changes/2026-09-29-verify-backoff.yaml", want: "summary: Add a rule about testing retry backoff.\nrules:\n  practices/testing/verify-backoff: new\n",
			rows: []string{"practices/testing/verify-backoff new - 1.0.0"}},
		{name: "retirement with a replacement", files: map[string]string{"practices/testing/b.md": ""},
			request: ChangeRequest{IDs: []string{"practices/testing/b"}, Retire: true, ReplacedBy: "practices/testing/a", Summary: "Covered by a."},
			note:    "changes/2026-09-29-b.yaml", want: "summary: Covered by a.\nrules:\n  practices/testing/b:\n    change: retired\n    replacedBy: practices/testing/a\n",
			rows: []string{"practices/testing/b retired 1.0.0 - replacedBy practices/testing/a"}},
		{name: "retirement without a replacement", files: map[string]string{"practices/testing/a.md": "", "practices/testing/assets/a": ""},
			request: ChangeRequest{IDs: []string{"practices/testing/a"}, Retire: true, Summary: "Agents shouldn't test retries this way."},
			note:    "changes/2026-09-29-a.yaml", want: "summary: Agents shouldn't test retries this way.\nrules:\n  practices/testing/a: retired\n",
			rows: []string{"practices/testing/a retired 1.0.0 -"}},
		{name: "summary that YAML must quote", files: map[string]string{"practices/testing/a.md": ruleText("Changed.")},
			request: ChangeRequest{IDs: []string{"practices/testing/a"}, Bump: rules.ChangePatch, Summary: "  Fix: '#' is \"quoted\" - " + strings.Repeat("long ", 40) + " "},
			note:    "changes/2026-09-29-a.yaml",
			rows:    []string{"practices/testing/a patch 1.0.0 1.0.1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, options := authorClone(t, libraryFiles(), releaseOne)
			edit(t, options.Directory, test.files)
			name, content, err := recordChange(t, options, test.request)
			if err != nil || name != test.note || (test.want != "" && content != test.want) {
				t.Fatalf("wrote %s:\n%s\nerror %v; want %s:\n%s", name, content, err, test.note, test.want)
			}
			note, err := rules.ParseChangeNote([]byte(content), name)
			if lines := strings.Split(content, "\n"); err != nil || note.Summary != strings.TrimSpace(test.request.Summary) || !strings.HasPrefix(lines[0], "summary: ") || lines[1] != "rules:" {
				t.Fatalf("note %+v does not keep the summary on one line: %v\n%s", note, err, content)
			}
			result, err := Check(context.Background(), options)
			if err != nil || !slices.Equal(previewRows(result.PendingRelease), test.rows) {
				t.Fatalf("check after the note: %q %v", previewRows(result.PendingRelease), err)
			}
		})
	}
}

// TestChange_NamesEachNoteUniquely never edits an existing note or reuses a published note's name.
func TestChange_NamesEachNoteUniquely(t *testing.T) {
	ctx := context.Background()
	fixture, options := authorClone(t, libraryFiles(), releaseOne)
	published := "summary: Published.\nrules:\n  practices/testing/a: patch\n"
	if _, err := fixture.Commit(ctx, options.Directory, "Release 2", map[string][]byte{"practices/testing/a.md": []byte(ruleText("Two.")), "changes/2026-09-29-a.yaml": []byte(published)}); err != nil {
		t.Fatal(err)
	}
	two := "Library release 2.\n---\nrelease: 2\nrules:\n  practices/testing/a: 1.0.1\n  practices/testing/b: 1.0.0\nchanges:\n  practices/testing/a:\n    change: patch\n    from: 1.0.0\n    summary: Published.\n"
	if err := fixture.Tag(ctx, options.Directory, "release/2", two); err != nil {
		t.Fatal(err)
	}
	// The published note is gone from the working tree, but its name stays taken.
	edit(t, options.Directory, map[string]string{"changes/2026-09-29-a.yaml": "", "changes/2026-09-29-a-2.yaml": "summary: Pending.\nrules:\n  practices/testing/a: patch\n", "practices/testing/a.md": ruleText("Three.")})
	request := ChangeRequest{IDs: []string{"practices/testing/a"}, Bump: rules.ChangePatch, Summary: "Another fix."}
	for _, want := range []string{"changes/2026-09-29-a-3.yaml", "changes/2026-09-29-a-4.yaml"} {
		name, _, err := recordChange(t, options, request)
		if err != nil || name != want {
			t.Fatalf("wrote %s, want %s: %v", name, want, err)
		}
	}
	pending, err := os.ReadFile(filepath.Join(options.Directory, "changes/2026-09-29-a-2.yaml"))
	if err != nil || string(pending) != "summary: Pending.\nrules:\n  practices/testing/a: patch\n" {
		t.Fatalf("changed an existing note: %q %v", pending, err)
	}
}

// TestChange_RejectsNotesThatDontMatchTheLibrary fails before writing anything.
func TestChange_RejectsNotesThatDontMatchTheLibrary(t *testing.T) {
	a, b, c := "practices/testing/a", "practices/testing/b", "practices/testing/c"
	for _, test := range []struct {
		name    string
		files   map[string]string
		request ChangeRequest
		code    string
		message string
	}{
		{"unknown rule", nil, ChangeRequest{IDs: []string{c}, Summary: "Add c."}, "unknown-rule", c + " isn't a rule in the library."},
		{"deleted rule without --retire", map[string]string{"practices/testing/b.md": ""}, ChangeRequest{IDs: []string{b}, Bump: rules.ChangePatch, Summary: "Fix b."}, "unknown-rule", "To record its retirement, add --retire."},
		{"ID with .md", nil, ChangeRequest{IDs: []string{a + ".md"}, Bump: rules.ChangePatch, Summary: "Fix a."}, "unknown-rule", "Rule IDs omit the .md extension."},
		{"bump for a new rule", map[string]string{"practices/testing/c.md": ruleText("New.")}, ChangeRequest{IDs: []string{c}, Bump: rules.ChangeMinor, Summary: "Add c."}, "invalid-change", "--bump isn't accepted for new rules"},
		{"new and versioned rules", map[string]string{"practices/testing/c.md": ruleText("New.")}, ChangeRequest{IDs: []string{a, c}, Summary: "Add c."}, "invalid-change", "Record them in separate notes."},
		{"retirement of a rule that exists", nil, ChangeRequest{IDs: []string{a}, Retire: true, Summary: "Retire a."}, "invalid-change", a + " still exists."},
		{"retirement of an unpublished rule", nil, ChangeRequest{IDs: []string{c}, Retire: true, Summary: "Retire c."}, "invalid-change", c + " was never published, so it can't be retired."},
		{"bump for a retirement", map[string]string{"practices/testing/b.md": ""}, ChangeRequest{IDs: []string{b}, Retire: true, Bump: rules.ChangeMajor, Summary: "Retire b."}, "invalid-change", "--bump isn't accepted for retired rules."},
		{"replacement without --retire", nil, ChangeRequest{IDs: []string{a}, Bump: rules.ChangeMajor, ReplacedBy: b, Summary: "Replace a."}, "invalid-change", "--replaced-by requires --retire and a single rule"},
		{"replacement for several rules", map[string]string{"practices/testing/a.md": "", "practices/testing/b.md": ""}, ChangeRequest{IDs: []string{a, b}, Retire: true, ReplacedBy: c, Summary: "Replace both."}, "invalid-change", "--replaced-by requires --retire and a single rule"},
		{"duplicate rule", nil, ChangeRequest{IDs: []string{a, a}, Bump: rules.ChangePatch, Summary: "Fix a."}, "invalid-change", a + " is named more than once"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, options := authorClone(t, libraryFiles(), releaseOne)
			edit(t, options.Directory, test.files)
			_, err := PlanChange(context.Background(), test.request, options)
			if errorCode(err) != test.code || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("got %v, want %s: %s", err, test.code, test.message)
			}
			if _, err := os.Stat(filepath.Join(options.Directory, "changes")); !os.IsNotExist(err) {
				t.Fatal("wrote changes/", err)
			}
		})
	}
}

// TestChange_RequiresABumpAndSummaryToCommit lets prompts supply them, then requires them when writing.
func TestChange_RequiresABumpAndSummaryToCommit(t *testing.T) {
	ctx := context.Background()
	_, options := authorClone(t, libraryFiles(), releaseOne)
	plan, err := PlanChange(ctx, ChangeRequest{IDs: []string{"practices/testing/a"}}, options)
	if err != nil || !plan.Versioned {
		t.Fatal(plan, err)
	}
	if _, err := plan.Commit(ctx, "", "Fix a.", noteDay); errorCode(err) != "invalid-change" || !strings.Contains(err.Error(), "--bump is required") {
		t.Fatal(err)
	}
	if _, err := plan.Commit(ctx, rules.ChangePatch, " ", noteDay); errorCode(err) != "invalid-change" || !strings.Contains(err.Error(), "--summary is required") {
		t.Fatal(err)
	}
	// The note parser rejects a summary of more than one line, so the command never writes one.
	var validation *rules.ValidationError
	if _, err := plan.Commit(ctx, rules.ChangePatch, "One.\nTwo.", noteDay); !errors.As(err, &validation) || !strings.HasSuffix(validation.Location, ".summary") {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(options.Directory, "changes")); !os.IsNotExist(err) {
		t.Fatal("wrote a note with a multi-line summary", err)
	}
	if _, err := plan.Commit(ctx, rules.ChangePatch, "Fix a.", noteDay); err != nil {
		t.Fatal(err)
	}
}

// TestChange_RetiredRules can't retire a rule twice or reuse a retired rule's ID.
func TestChange_RetiredRules(t *testing.T) {
	ctx := context.Background()
	fixture, options := authorClone(t, libraryFiles(), releaseOne)
	if _, err := fixture.Commit(ctx, options.Directory, "Retire b", map[string][]byte{"practices/testing/b.md": nil}); err != nil {
		t.Fatal(err)
	}
	two := "Library release 2.\n---\nrelease: 2\nrules:\n  practices/testing/a: 1.0.0\nretired:\n  practices/testing/b:\n    lastVersion: 1.0.0\n    summary: Retire b.\n"
	if err := fixture.Tag(ctx, options.Directory, "release/2", two); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanChange(ctx, ChangeRequest{IDs: []string{"practices/testing/b"}, Retire: true}, options); errorCode(err) != "invalid-change" || !strings.Contains(err.Error(), "practices/testing/b was already retired by release/2.") {
		t.Fatal(err)
	}
	edit(t, options.Directory, map[string]string{"practices/testing/b.md": ruleText("Back.")})
	if _, err := PlanChange(ctx, ChangeRequest{IDs: []string{"practices/testing/b"}}, options); errorCode(err) != "invalid-change" || !strings.Contains(err.Error(), "reuses the ID of a rule that release/2 retired") {
		t.Fatal(err)
	}
}

// TestChange_BeforeTheFirstLibraryRelease refuses, because rules need no notes yet.
func TestChange_BeforeTheFirstLibraryRelease(t *testing.T) {
	_, inGit := authorClone(t, libraryFiles())
	outside := Options{Directory: t.TempDir()}
	for name, content := range libraryFiles() {
		edit(t, outside.Directory, map[string]string{name: string(content)})
	}
	for name, options := range map[string]Options{"Git": inGit, "outside Git": outside} {
		t.Run(name, func(t *testing.T) {
			_, err := PlanChange(context.Background(), ChangeRequest{IDs: []string{"practices/testing/a"}, Summary: "Add a."}, options)
			if errorCode(err) != "no-library-release" || !strings.Contains(err.Error(), "rules need no change notes") {
				t.Fatal(err)
			}
		})
	}
}
