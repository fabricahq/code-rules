// Check that library commands refuse the layout before version 2, share the writer of the repository's Syllabus
// directory, and compare a moved library with a library release made before the move.

package library

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fabricahq/syllabus/internal/gitexec"
	"github.com/fabricahq/syllabus/internal/rules"
	"github.com/fabricahq/syllabus/internal/test/gitfixture"
	"github.com/fabricahq/syllabus/syllabus"
)

// TestLibraryCommands_RefuseTheLayoutBeforeVersion2 refuses rule-library.yaml and rule folders at the repository
// root, whatever their kind, in every library command, before writing anything, and lists where the file belongs.
func TestLibraryCommands_RefuseTheLayoutBeforeVersion2(t *testing.T) {
	ctx := context.Background()
	for name, want := range map[string]struct{ location, moves string }{
		"rule-library.yaml":  {"rule-library.yaml", ":\n\n  rule-library.yaml  ->  the library section of .syllabus/config.yaml\n\nThen delete rule-library.yaml."},
		"techs/go/errors.md": {"techs/", ":\n\n  techs/go/errors.md  ->  .syllabus/rules/techs/go/errors.md\n\nRule IDs and versions don't change."},
		"practices":          {"practices", ":\n\n  practices  ->  .syllabus/rules/practices\n\nRule IDs and versions don't change."},
	} {
		commands := map[string]func(Options) error{
			"init":  func(options Options) error { _, err := Initialize(ctx, options, nil, "1.2.3"); return err },
			"check": func(options Options) error { _, err := Check(ctx, options); return err },
			"group": func(options Options) error {
				_, err := AddGroup(ctx, "techs/rust", syllabus.GroupMetadata{Name: "Rust", Description: "Rust guidance.", WhenToRead: "When editing Rust."}, options)
				return err
			},
			"rule": func(options Options) error {
				body := "Return errors.\n"
				_, err := AddRule(ctx, "techs/go/errors", rules.RuleMetadata{Title: "Errors", Impact: "HIGH", ImpactDescription: "Keep failures.", WhenToRead: "When calling functions."}, RuleOptions{Options: options, Body: &body})
				return err
			},
			"change": func(options Options) error {
				_, err := PlanChange(ctx, ChangeRequest{IDs: []string{"techs/go/errors"}, Summary: "Change it."}, options)
				return err
			},
			"release": func(options Options) error {
				_, err := Release(ctx, ReleaseRequest{Options: options, DryRun: true})
				return err
			},
		}
		for command, run := range commands {
			t.Run(name+"/"+command, func(t *testing.T) {
				options := Options{Directory: t.TempDir()}
				if command != "init" {
					if _, err := Initialize(ctx, options, nil, "1.2.3"); err != nil {
						t.Fatal(err)
					}
				}
				path := filepath.Join(options.Directory, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("formatVersion: 1\n"), 0600); err != nil {
					t.Fatal(err)
				}
				before := tree(t, options.Directory)
				err := run(options)
				var invalid *ValidationError
				if !errors.As(err, &invalid) || invalid.Location != want.location || !strings.Contains(invalid.Problem, want.moves) {
					t.Fatalf("got %v; want the layout refused at %s, listing\n%s", err, want.location, want.moves)
				}
				if after := tree(t, options.Directory); !slices.Equal(after, before) {
					t.Fatalf("wrote files:\n%q\nwant:\n%q", after, before)
				}
			})
		}
	}
}

// TestCheck_ListsTheMovesOfALibrarysOwnFiles refuses a library in the layout before version 2 and lists where each
// of its files at the repository root belongs now, the manifest first, up to legacyLayoutMoves of them, then how many
// more there are, and asks to rewrite the manifest only when there is one.
func TestCheck_ListsTheMovesOfALibrarysOwnFiles(t *testing.T) {
	ctx := context.Background()
	manyRules := map[string]string{"rule-library.yaml": "formatVersion: 1\n"}
	var manyMoves []string
	for index := range legacyLayoutMoves + 5 {
		name := fmt.Sprintf("practices/testing/rule-%02d.md", index+1)
		manyRules[name] = ruleText("Test it.")
		if index < legacyLayoutMoves-1 {
			manyMoves = append(manyMoves, "  "+name+"  ->  .syllabus/rules/"+name)
		}
	}
	for _, test := range []struct {
		name  string
		files map[string]string
		want  []string
		// manifest is whether the problem asks to rewrite rule-library.yaml.
		manifest bool
		// empty makes the repository root hold an empty techs/ folder, whose problem only asks to delete it.
		empty bool
	}{
		{
			name: "a library",
			files: map[string]string{
				"rule-library.yaml":                     "formatVersion: 1\n",
				"LICENSE.md":                            "MIT License\n",
				"README.md":                             "# Library\n",
				"assets/retry.svg":                      "<svg/>\n",
				"changes/2026-10-01-retry.yaml":         "summary: Retry.\nrules:\n  practices/testing/a: patch\n",
				"practices/testing/_group.yaml":         "name: Testing\ndescription: Testing guidance.\nwhenToRead: When testing.\n",
				"practices/testing/a.md":                ruleText("Test the retry limit."),
				"practices/testing/assets/a/example.go": "package example\n",
				"techs/go/errors.md":                    ruleText("Return errors."),
				".syllabus/config.yaml":                 "schemaVersion: 2\nsources: {}\n",
			},
			want: []string{
				"  rule-library.yaml                      ->  the library section of .syllabus/config.yaml",
				"  assets/retry.svg                       ->  .syllabus/assets/retry.svg",
				"  changes/2026-10-01-retry.yaml          ->  .syllabus/changes/2026-10-01-retry.yaml",
				"  practices/testing/_group.yaml          ->  .syllabus/rules/practices/testing/_group.yaml",
				"  practices/testing/a.md                 ->  .syllabus/rules/practices/testing/a.md",
				"  practices/testing/assets/a/example.go  ->  .syllabus/rules/practices/testing/assets/a/example.go",
				"  techs/go/errors.md                     ->  .syllabus/rules/techs/go/errors.md",
			},
			manifest: true,
		},
		{
			name:     "more files than the list shows",
			files:    manyRules,
			want:     append(append([]string{"  rule-library.yaml" + strings.Repeat(" ", 13) + "->  the library section of .syllabus/config.yaml"}, manyMoves...), "  and 6 more files, moved the same way"),
			manifest: true,
		},
		{
			name:  "rule folders without a manifest",
			files: map[string]string{"techs/go/errors.md": ruleText("Return errors.")},
			want:  []string{"  techs/go/errors.md  ->  .syllabus/rules/techs/go/errors.md"},
		},
		{name: "an empty rule folder", empty: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := Options{Directory: t.TempDir()}
			for name, text := range test.files {
				path := filepath.Join(options.Directory, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if test.empty {
				if err := os.Mkdir(filepath.Join(options.Directory, "techs"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			_, err := Check(ctx, options)
			var invalid *ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("got %v; want the layout refused", err)
			}
			if test.empty {
				if want := "The rule folders there hold no files: delete them, then run syllabus library check."; invalid.Location != "techs/" || !strings.HasSuffix(invalid.Problem, want) {
					t.Fatalf("got %v; want a problem ending %q", err, want)
				}
				return
			}
			want := ":\n\n" + strings.Join(test.want, "\n") + "\n\n"
			if !strings.Contains(invalid.Problem, want) {
				t.Fatalf("got %s\nwant a problem listing\n%s", invalid.Problem, want)
			}
			if got := strings.Contains(invalid.Problem, "delete rule-library.yaml") && strings.Contains(invalid.Problem, "License and notice paths"); got != test.manifest {
				t.Fatalf("asks to rewrite the manifest: %t; want %t\n%s", got, test.manifest, invalid.Problem)
			}
		})
	}
}

// TestRequireLibraryLayout_LeavesTheRepositorysOwnFoldersUnread doesn't read a repository root's own assets/ and
// changes/ folders when nothing at the root marks the layout before version 2, so a folder there that can't be read
// doesn't stop library commands. A context that has already ended makes any walk of a folder fail, so it shows
// that neither folder is read.
func TestRequireLibraryLayout_LeavesTheRepositorysOwnFoldersUnread(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"assets/logo.svg", "changes/2026-10-01.md"} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("project file\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if err := requireLibraryLayout(ended, root); err != nil {
		t.Fatalf("got %v; want the repository's own folders left unread", err)
	}
}

// tree lists every file and directory under dir, relative to it.
func tree(t *testing.T, dir string) []string {
	t.Helper()
	var names []string
	err := filepath.WalkDir(dir, func(path string, _ os.DirEntry, err error) error {
		relative, _ := filepath.Rel(dir, path)
		names = append(names, filepath.ToSlash(relative))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return names
}

// TestLibraryCommands_ShareTheWriterOfTheSyllabusDirectory refuses library check while a project command holds
// the writer of the repository's Syllabus directory, and records no change note while it waits.
func TestLibraryCommands_ShareTheWriterOfTheSyllabusDirectory(t *testing.T) {
	ctx := context.Background()
	_, options := authorClone(t, libraryFiles(), releaseOne)
	edit(t, options.Directory, map[string]string{"practices/testing/a.md": ruleText("Changed.")})
	lock := filepath.Join(options.Directory, ".syllabus", ".syllabus-lock")
	if err := os.WriteFile(lock, []byte("held by a project command\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(ctx, options); errorCode(err) != "busy" {
		t.Fatalf("check while a writer holds the directory: %v", err)
	}
	plan, err := PlanChange(ctx, ChangeRequest{IDs: []string{"practices/testing/a"}, Bump: syllabus.ChangePatch, Summary: "Fix a."}, options)
	if err != nil {
		t.Fatal(err)
	}
	waiting, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if _, err := plan.Commit(waiting, syllabus.ChangePatch, "Fix a.", noteDay); err == nil {
		t.Fatal("recorded a change note while a project command held the writer")
	}
	if notes, err := os.ReadDir(filepath.Join(options.Directory, ".syllabus", "changes")); err == nil && len(notes) != 0 {
		t.Fatalf("wrote %d notes", len(notes))
	}
}

// legacyFiles is libraryFiles in the layout before version 2, at the repository root, with a license, a shared
// asset that rule a links to with an absolute link and rule b with a relative one, and a project configuration in
// .syllabus/, which that layout didn't read.
func legacyFiles() map[string][]byte {
	files := map[string][]byte{
		"rule-library.yaml":          []byte("formatVersion: 1\nlicense:\n  spdxExpression: MIT\n  file: LICENSE.md\n  notices: []\n"),
		"LICENSE.md":                 []byte("MIT License\n"),
		"assets/retry.svg":           []byte("<svg/>\n"),
		".syllabus/config.yaml":      []byte("schemaVersion: 2\nsources: {}\n"),
		".syllabus/generated/OUT.md": []byte("generated\n"),
	}
	for name, data := range libraryFiles() {
		if name != "config.yaml" {
			files[name] = data
		}
	}
	files["practices/testing/a.md"] = []byte(ruleText("Test the retry limit. See [the lifecycle](/assets/retry.svg)."))
	files["practices/testing/b.md"] = []byte(ruleText("Test the backoff. See [the lifecycle](../../assets/retry.svg)."))
	return files
}

// moveLibrary moves legacyFiles into the version 2 layout in dir, as the layout message explains, giving rule b's
// relative link one more ../, and commits the move.
func moveLibrary(t *testing.T, fixture *gitfixture.Fixture, dir string) {
	t.Helper()
	moved := map[string][]byte{"rule-library.yaml": nil, ".syllabus/config.yaml": []byte("schemaVersion: 2\nsources: {}\nlibrary:\n  license:\n    spdxExpression: MIT\n    file: LICENSE.md\n    notices: []\n")}
	for name, data := range legacyFiles() {
		if first, _, _ := strings.Cut(name, "/"); first == "techs" || first == "practices" || first == "assets" {
			moved[name] = nil
			if name == "practices/testing/b.md" {
				data = []byte(ruleText("Test the backoff. See [the lifecycle](../../../assets/retry.svg)."))
			}
			maps.Copy(moved, gitfixture.Library(map[string][]byte{name: data}))
		}
	}
	if _, err := fixture.Commit(context.Background(), dir, "Move the library into .syllabus", moved); err != nil {
		t.Fatal(err)
	}
	// Committing deletions leaves the emptied folders, which git mv would have removed.
	for _, folder := range []string{"techs", "practices", "assets"} {
		if err := os.RemoveAll(filepath.Join(dir, folder)); err != nil {
			t.Fatal(err)
		}
	}
}

// TestCheck_ComparesAMovedLibraryWithALibraryReleaseMadeBeforeTheMove reads the latest library release in the
// layout before version 2, so rule a, whose bytes didn't change, needs no note, rule b, whose link changed, does,
// and the manifest, now the library section of config.yaml, declaring the same license, isn't a changed
// library-wide file.
func TestCheck_ComparesAMovedLibraryWithALibraryReleaseMadeBeforeTheMove(t *testing.T) {
	ctx := context.Background()
	for name, recorded := range map[string]bool{"unrecorded link change": false, "recorded link change": true} {
		t.Run(name, func(t *testing.T) {
			fixture, err := gitfixture.New(ctx, legacyFiles())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = fixture.Close() })
			if _, err := fixture.Command(ctx, "tag", "--annotate", "--cleanup=verbatim", "--message", releaseOne, "release/1"); err != nil {
				t.Fatal(err)
			}
			dir, err := fixture.Clone(ctx)
			if err != nil {
				t.Fatal(err)
			}
			options := Options{Directory: dir, Git: gitexec.Options{GitPath: fixture.GitPath, Environment: fixture.Environment}}
			moveLibrary(t, fixture, dir)
			if !recorded {
				if _, err := Check(ctx, options); errorCode(err) != "change-notes" || !strings.Contains(err.Error(), "practices/testing/b changed since release/1") || strings.Contains(err.Error(), "practices/testing/a") {
					t.Fatalf("got %v; want only rule b's changed link to need a note", err)
				}
				return
			}
			edit(t, dir, map[string]string{"changes/move.yaml": "summary: Link to the moved shared asset.\nrules:\n  practices/testing/b: patch\n"})
			result, err := Check(ctx, options)
			if err != nil || !slices.Equal(previewRows(result.PendingRelease), []string{"practices/testing/b patch 1.0.0 1.0.1"}) || len(result.PendingRelease.LibraryFiles) != 0 {
				t.Fatalf("got %+v, %v", result.PendingRelease, err)
			}
		})
	}
}

// TestCheck_ListsTheManifestOnlyWhenItsLibrarySectionChanges ignores changes to the sources a library repository
// uses as a project, and to its configuration's comments, which the library doesn't publish, and lists config.yaml
// as a changed library-wide file once its library section changes.
func TestCheck_ListsTheManifestOnlyWhenItsLibrarySectionChanges(t *testing.T) {
	ctx := context.Background()
	_, options := authorClone(t, libraryFiles(), releaseOne)
	edit(t, options.Directory, map[string]string{"config.yaml": "# The repository's project and library.\nschemaVersion: 2\nsources:\n  upstream:\n    repository: https://github.com/acme/upstream.git\n    rules:\n      include: ['techs/*']\nlibrary: {}\n"})
	result, err := Check(ctx, options)
	if err != nil || len(result.PendingRelease.LibraryFiles) != 0 {
		t.Fatalf("got %+v, %v; want no library-wide change", result.PendingRelease, err)
	}
	edit(t, options.Directory, map[string]string{
		"config.yaml": "schemaVersion: 2\nsources: {}\nlibrary:\n  license:\n    spdxExpression: MIT\n    file: LICENSE.md\n    notices: []\n",
		"LICENSE.md":  "MIT License\n",
	})
	result, err = Check(ctx, options)
	if err != nil || !slices.Equal(result.PendingRelease.LibraryFiles, []string{"LICENSE.md", "config.yaml"}) {
		t.Fatalf("got %+v, %v; want the manifest and license listed", result.PendingRelease, err)
	}
}
