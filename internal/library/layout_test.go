// Check that library commands refuse the layout before version 2, share the writer of the repository's Syllabus
// directory, and compare a moved library with a library release made before the move.

package library

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// TestLibraryCommands_RefuseTheLayoutBeforeVersion2 refuses rule-library.yaml and rule folders at the repository
// root, whatever their kind, in every library command, before writing anything.
func TestLibraryCommands_RefuseTheLayoutBeforeVersion2(t *testing.T) {
	ctx := context.Background()
	for name, location := range map[string]string{"rule-library.yaml": "rule-library.yaml", "techs/go/errors.md": "techs/", "practices": "practices"} {
		commands := map[string]func(Options) error{
			"init":  func(options Options) error { _, err := Initialize(ctx, options, nil, "1.2.3"); return err },
			"check": func(options Options) error { _, err := Check(ctx, options); return err },
			"group": func(options Options) error {
				_, err := AddGroup(ctx, "techs/rust", coderules.GroupMetadata{Name: "Rust", Description: "Rust guidance.", WhenToRead: "When editing Rust."}, options)
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
				if !errors.As(err, &invalid) || invalid.Location != location || invalid.Problem != legacyLayoutProblem {
					t.Fatalf("got %v; want the layout refused at %s", err, location)
				}
				if after := tree(t, options.Directory); !slices.Equal(after, before) {
					t.Fatalf("wrote files:\n%q\nwant:\n%q", after, before)
				}
			})
		}
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
	plan, err := PlanChange(ctx, ChangeRequest{IDs: []string{"practices/testing/a"}, Bump: coderules.ChangePatch, Summary: "Fix a."}, options)
	if err != nil {
		t.Fatal(err)
	}
	waiting, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if _, err := plan.Commit(waiting, coderules.ChangePatch, "Fix a.", noteDay); err == nil {
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
