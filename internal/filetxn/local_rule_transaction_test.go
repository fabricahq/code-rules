// Exercise replacing a local rule and its asset directory together with managed trees: publication, removal,
// process death, edits, and refused targets.

package filetxn

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"testing"
)

// forkFile is the local rule the tests replace, relative to local/.
const forkFile = "techs/go/fork.md"

// writeForkProject writes the fork with two assets, another local rule with its own asset, and the previous vendor
// and generated output.
func writeForkProject(t *testing.T, root *os.Root) {
	t.Helper()
	writeFixture(t, root, "local/techs/go/fork.md", "old fork")
	writeFixture(t, root, "local/techs/go/assets/fork/notes.md", "old notes")
	writeFixture(t, root, "local/techs/go/assets/fork/extra/data.bin", "old data")
	writeFixture(t, root, "local/techs/go/other.md", "other rule")
	writeFixture(t, root, "local/techs/go/assets/other/notes.md", "other notes")
	writeFixture(t, root, "vendor/team/rule.md", "old rule")
	writeFixture(t, root, "generated/RULES.md", "old output")
}

// forkUpdate replaces the fork and its assets with assets, which nil removes, and the managed trees with new output.
func forkUpdate(assets map[string][]byte) map[Target]map[string][]byte {
	return map[Target]map[string][]byte{
		LocalRule(forkFile):       {"local/" + forkFile: []byte("new fork")},
		LocalRuleAssets(forkFile): assets,
		Vendor:                    {"team/rule.md": []byte("new rule")},
		Generated:                 {"RULES.md": []byte("new output")},
	}
}

// localFiles returns every file under local/ with its text.
func localFiles(t *testing.T, root *os.Root) map[string]string {
	t.Helper()
	tree, err := ReadTree(context.Background(), root, "local")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for name, data := range tree.Files {
		files[name] = string(data)
	}
	return files
}

// requireForkProject checks that local/ holds exactly want and the managed trees the matching output.
func requireForkProject(t *testing.T, root *os.Root, state string, want map[string]string) {
	t.Helper()
	for name, suffix := range map[string]string{"vendor/team/rule.md": " rule", "generated/RULES.md": " output"} {
		data, err := root.ReadFile(name)
		if err != nil || string(data) != state+suffix {
			t.Fatalf("%s: got %q, want %q: %v", name, data, state+suffix, err)
		}
	}
	if got := localFiles(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("local files:\n%v\nwant\n%v", got, want)
	}
	if err := RequireIdle(root); err != nil {
		t.Fatal(err)
	}
}

// oldLocal is local/ before the update, and newLocal after it replaces the fork's assets with new.md.
var (
	oldLocal = map[string]string{"techs/go/fork.md": "old fork", "techs/go/assets/fork/notes.md": "old notes", "techs/go/assets/fork/extra/data.bin": "old data", "techs/go/other.md": "other rule", "techs/go/assets/other/notes.md": "other notes"}
	newLocal = map[string]string{"techs/go/fork.md": "new fork", "techs/go/assets/fork/new.md": "new notes", "techs/go/other.md": "other rule", "techs/go/assets/other/notes.md": "other notes"}
)

// TestApply_ReplacesALocalRuleAndItsAssetDirectory installs the new fork and exactly its new assets, removing the
// old ones, and leaves the other local rule and its assets alone.
func TestApply_ReplacesALocalRuleAndItsAssetDirectory(t *testing.T) {
	root := openProject(t)
	writeForkProject(t, root)
	if err := WithWriter(context.Background(), root, func(w *Writer) error {
		return w.Apply(forkUpdate(map[string][]byte{"new.md": []byte("new notes")}), nil)
	}); err != nil {
		t.Fatal(err)
	}
	requireForkProject(t, root, "new", newLocal)
}

// TestApply_RemovesALocalRuleAssetDirectory removes the fork's asset directory when the new fork has no assets,
// and treats removing an absent one as nothing to do.
func TestApply_RemovesALocalRuleAssetDirectory(t *testing.T) {
	root := openProject(t)
	writeForkProject(t, root)
	for range 2 {
		if err := WithWriter(context.Background(), root, func(w *Writer) error { return w.Apply(forkUpdate(nil), nil) }); err != nil {
			t.Fatal(err)
		}
	}
	// A transaction that only removes the absent directory has nothing to do.
	if err := WithWriter(context.Background(), root, func(w *Writer) error {
		return w.Apply(map[Target]map[string][]byte{LocalRuleAssets(forkFile): nil}, nil)
	}); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"techs/go/fork.md": "new fork", "techs/go/other.md": "other rule", "techs/go/assets/other/notes.md": "other notes"}
	requireForkProject(t, root, "new", want)
	if _, err := root.Lstat("local/techs/go/assets/fork"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("the asset directory is still there", err)
	}
}

// TestApply_CreatesALocalRuleAssetDirectoryAndItsParent gives a fork without assets its first ones, creating
// local/techs/go/assets, and removes that directory again when the transaction fails.
func TestApply_CreatesALocalRuleAssetDirectoryAndItsParent(t *testing.T) {
	setup := func(t *testing.T) *os.Root {
		root := openProject(t)
		writeFixture(t, root, "local/techs/go/fork.md", "old fork")
		writeFixture(t, root, "vendor/team/rule.md", "old rule")
		writeFixture(t, root, "generated/RULES.md", "old output")
		return root
	}
	t.Run("applied", func(t *testing.T) {
		root := setup(t)
		if err := WithWriter(context.Background(), root, func(w *Writer) error {
			return w.Apply(forkUpdate(map[string][]byte{"new.md": []byte("new notes")}), nil)
		}); err != nil {
			t.Fatal(err)
		}
		requireForkProject(t, root, "new", map[string]string{"techs/go/fork.md": "new fork", "techs/go/assets/fork/new.md": "new notes"})
	})
	t.Run("refused", func(t *testing.T) {
		root := setup(t)
		err := WithWriter(context.Background(), root, func(w *Writer) error {
			return w.Apply(forkUpdate(map[string][]byte{"new.md": []byte("new notes")}), func() error { return failure("concurrent-change", "changed", nil) })
		})
		projectCode(t, err, "concurrent-change")
		requireForkProject(t, root, "old", map[string]string{"techs/go/fork.md": "old fork"})
		if _, err := root.Lstat("local/techs/go/assets"); !errors.Is(err, fs.ErrNotExist) {
			t.Fatal("the failed transaction left the directory it created", err)
		}
	})
}

// TestApply_ChangesNoLiveDirectoryBeforeTheInputsAreChecked: when Apply asks the caller to recheck its inputs, the
// live tree is still exactly as before, so a caller that compares directories doesn't mistake Apply's own parent
// directory for a concurrent edit.
func TestApply_ChangesNoLiveDirectoryBeforeTheInputsAreChecked(t *testing.T) {
	root := openProject(t)
	writeFixture(t, root, "local/techs/go/fork.md", "old fork")
	writeFixture(t, root, "vendor/team/rule.md", "old rule")
	writeFixture(t, root, "generated/RULES.md", "old output")
	before, err := ReadTree(context.Background(), root, "local")
	if err != nil {
		t.Fatal(err)
	}
	if err := WithWriter(context.Background(), root, func(w *Writer) error {
		return w.Apply(forkUpdate(map[string][]byte{"new.md": []byte("new notes")}), func() error {
			if current, err := ReadTree(context.Background(), root, "local"); err != nil || treeDigest(current) != treeDigest(before) {
				return failure("concurrent-change", "local/ changed before the inputs were checked", err)
			}
			return nil
		})
	}); err != nil {
		t.Fatal(err)
	}
	requireForkProject(t, root, "new", map[string]string{"techs/go/fork.md": "new fork", "techs/go/assets/fork/new.md": "new notes"})
}

// TestKilledWriterRemovesOnlyTheParentsItProvablyCreated stops the writer around creating local/techs/go/assets,
// the new asset directory's missing parent. Recovery restores every target and removes the directory only when the
// writer recorded creating it and it is still an empty directory: a directory another process created while the
// writer was dead, one the writer died before recording, and one another process then put a file in all stay, and
// recovery reports the last.
func TestKilledWriterRemovesOnlyTheParentsItProvablyCreated(t *testing.T) {
	if directory := os.Getenv("CODE_RULES_PARENT_CHILD"); directory != "" {
		root, err := os.OpenRoot(directory)
		if err != nil {
			t.Fatal(err)
		}
		defer root.Close()
		stop := os.Getenv("CODE_RULES_PARENT_STOP")
		err = WithWriter(context.Background(), root, func(w *Writer) error {
			mkdir, rename := w.mkdir, w.rename
			w.mkdir = func(name string) error {
				if stop == "before mkdir" {
					os.Exit(42)
				}
				err := mkdir(name)
				if err == nil && stop == "after mkdir" {
					os.Exit(42)
				}
				return err
			}
			w.rename = func(from, to string) error {
				err := rename(from, to)
				// Vendor's backup is the first rename after the parents are created and recorded.
				if err == nil && stop == "after record" && to == transactionName+"/"+backupPrefix+"vendor" {
					os.Exit(42)
				}
				return err
			}
			return w.Apply(forkUpdate(map[string][]byte{"new.md": []byte("new notes")}), nil)
		})
		t.Fatalf("child did not stop: %v", err)
	}
	for _, test := range []struct {
		name, stop string
		// meanwhile is what another process does while the writer is dead.
		meanwhile func(t *testing.T, root *os.Root)
		kept      bool
		files     map[string]string
		reported  []string
	}{
		{"recorded", "after record", nil, false, nil, nil},
		{"not yet recorded", "after mkdir", nil, true, nil, nil},
		{"created by another process", "before mkdir", func(t *testing.T, root *os.Root) {
			if err := root.Mkdir("local/techs/go/assets", 0700); err != nil {
				t.Fatal(err)
			}
		}, true, nil, nil},
		{"recorded, then given a file", "after record", func(t *testing.T, root *os.Root) {
			writeFixture(t, root, "local/techs/go/assets/theirs.md", "their file")
		}, true, map[string]string{"techs/go/assets/theirs.md": "their file"}, []string{"local/techs/go/assets"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := openProject(t)
			writeFixture(t, root, "local/techs/go/fork.md", "old fork")
			writeFixture(t, root, "vendor/team/rule.md", "old rule")
			writeFixture(t, root, "generated/RULES.md", "old output")
			child := exec.Command(os.Args[0], "-test.run=^TestKilledWriterRemovesOnlyTheParentsItProvablyCreated$")
			child.Env = append(os.Environ(), "CODE_RULES_PARENT_CHILD="+root.Name(), "CODE_RULES_PARENT_STOP="+test.stop)
			output, err := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 42 {
				t.Fatal(err, string(output))
			}
			if test.meanwhile != nil {
				test.meanwhile(t, root)
			}
			var kept []string
			if err := WithWriter(context.Background(), root, func(w *Writer) error {
				kept = w.Kept()
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"techs/go/fork.md": "old fork"}
			maps.Copy(want, test.files)
			requireForkProject(t, root, "old", want)
			if _, err := root.Lstat("local/techs/go/assets"); test.kept != (err == nil) {
				t.Fatalf("directory kept: %v, want %v", err == nil, test.kept)
			}
			if !slices.Equal(kept, test.reported) {
				t.Fatalf("reported %q as kept, want %q", kept, test.reported)
			}
		})
	}
}

// TestApply_RefusesAParentAnotherProcessCreatedMeanwhile: a directory that appears after the journal is written and
// before Apply creates it isn't Apply's, so Apply refuses with concurrent-change, rolls back, and keeps it.
func TestApply_RefusesAParentAnotherProcessCreatedMeanwhile(t *testing.T) {
	root := openProject(t)
	writeFixture(t, root, "local/techs/go/fork.md", "old fork")
	writeFixture(t, root, "vendor/team/rule.md", "old rule")
	writeFixture(t, root, "generated/RULES.md", "old output")
	err := WithWriter(context.Background(), root, func(w *Writer) error {
		mkdir := w.mkdir
		w.mkdir = func(name string) error {
			if err := root.Mkdir(name, 0700); err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, name+"/theirs.md", "their file")
			return mkdir(name)
		}
		return w.Apply(forkUpdate(map[string][]byte{"new.md": []byte("new notes")}), nil)
	})
	projectCode(t, err, "concurrent-change")
	requireForkProject(t, root, "old", map[string]string{"techs/go/fork.md": "old fork", "techs/go/assets/theirs.md": "their file"})
}

// TestRemoveCreatedParent removes only an empty directory: it keeps a directory holding a file and a regular file
// put at its path, reporting both as kept, and does nothing for an absent one.
func TestRemoveCreatedParent(t *testing.T) {
	root := openProject(t)
	if err := root.MkdirAll("local/empty", 0700); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "local/full/theirs.md", "their file")
	writeFixture(t, root, "local/file", "their file")
	for name, want := range map[string]bool{"local/empty": false, "local/full": true, "local/file": true, "local/absent": false} {
		kept, err := removeCreatedParent(root, name)
		if err != nil || kept != want {
			t.Fatalf("%s: kept %v, %v; want %v", name, kept, err, want)
		}
	}
	if _, err := root.Lstat("local/empty"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("kept the empty directory", err)
	}
	for _, name := range []string{"local/full/theirs.md", "local/file"} {
		if data, err := root.ReadFile(name); err != nil || string(data) != "their file" {
			t.Fatalf("%s: %q, %v", name, data, err)
		}
	}
}

// TestApply_KeepsALocalRuleEditedDuringStaging refuses to replace a fork the user edited meanwhile, and keeps the
// previous fork, assets, and managed trees.
func TestApply_KeepsALocalRuleEditedDuringStaging(t *testing.T) {
	for name, edit := range map[string]string{"rule": "local/techs/go/fork.md", "asset": "local/techs/go/assets/fork/notes.md"} {
		t.Run(name, func(t *testing.T) {
			root := openProject(t)
			writeForkProject(t, root)
			err := WithWriter(context.Background(), root, func(w *Writer) error {
				return w.Apply(forkUpdate(map[string][]byte{"new.md": []byte("new notes")}), func() error {
					writeFixture(t, root, edit, "user edit")
					return nil
				})
			})
			projectCode(t, err, "concurrent-change")
			want := maps.Clone(oldLocal)
			want[edit[len("local/"):]] = "user edit"
			requireForkProject(t, root, "old", want)
		})
	}
}

// TestApply_RejectsTargetsOutsideLocalRules refuses paths that aren't a local rule or its asset directory, and
// removing anything but an asset directory.
func TestApply_RejectsTargetsOutsideLocalRules(t *testing.T) {
	root := openProject(t)
	writeForkProject(t, root)
	for _, target := range []Target{"local/techs/Go/fork.md", "local/techs/go/assets/fork/notes.md", "local/techs/go/assets", "local/techs/go/README.md", "local/techs/go/fork", "local/other/go/fork.md", "local/techs/go/assets/fork/nested/assets/x"} {
		err := WithWriter(context.Background(), root, func(w *Writer) error {
			return w.Apply(map[Target]map[string][]byte{target: {string(target): []byte("x")}}, nil)
		})
		projectCode(t, err, "invalid-target")
	}
	for _, target := range []Target{LocalRule(forkFile), Vendor, LocalGroupMetadata("techs/go")} {
		err := WithWriter(context.Background(), root, func(w *Writer) error { return w.Apply(map[Target]map[string][]byte{target: nil}, nil) })
		projectCode(t, err, "invalid-target")
	}
	requireForkProject(t, root, "old", oldLocal)
}

// TestKilledForkWriterRecovers stops the writer after each rename, replacing the assets or removing them; the next
// writer restores the previous fork, assets, and trees, or, once the commit marker exists, keeps everything new.
func TestKilledForkWriterRecovers(t *testing.T) {
	if directory := os.Getenv("CODE_RULES_FORK_CHILD"); directory != "" {
		root, err := os.OpenRoot(directory)
		if err != nil {
			t.Fatal(err)
		}
		defer root.Close()
		var assets map[string][]byte
		if os.Getenv("CODE_RULES_FORK_ASSETS") != "" {
			assets = map[string][]byte{"new.md": []byte("new notes")}
		}
		err = WithWriter(context.Background(), root, func(w *Writer) error {
			rename := w.rename
			w.rename = func(from, to string) error {
				err := rename(from, to)
				if err == nil && to == os.Getenv("CODE_RULES_FORK_STOP") {
					os.Exit(42)
				}
				return err
			}
			return w.Apply(forkUpdate(assets), nil)
		})
		t.Fatalf("child did not stop at rename: %v", err)
	}
	removed := map[string]string{"techs/go/fork.md": "new fork", "techs/go/other.md": "other rule", "techs/go/assets/other/notes.md": "other notes"}
	assetsBackup := transactionName + "/" + backupPrefix + "local+techs+go+assets+fork"
	for _, test := range []struct {
		name, stop, assets string
		state              string
		want               map[string]string
	}{
		{"probe", renameCheck + "/local+techs+go+fork.md", "yes", "old", oldLocal},
		{"vendor", "vendor", "yes", "old", oldLocal},
		{"generated", "generated", "yes", "old", oldLocal},
		{"rule", "local/techs/go/fork.md", "yes", "old", oldLocal},
		{"assets backed up", assetsBackup, "yes", "old", oldLocal},
		{"assets", "local/techs/go/assets/fork", "yes", "old", oldLocal},
		{"committed", transactionName + "/committed.json", "yes", "new", newLocal},
		{"removed assets backed up", assetsBackup, "", "old", oldLocal},
		{"committed removal", transactionName + "/committed.json", "", "new", removed},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := openProject(t)
			writeForkProject(t, root)
			child := exec.Command(os.Args[0], "-test.run=^TestKilledForkWriterRecovers$")
			child.Env = append(os.Environ(), "CODE_RULES_FORK_CHILD="+root.Name(), "CODE_RULES_FORK_STOP="+test.stop, "CODE_RULES_FORK_ASSETS="+test.assets)
			output, err := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 42 {
				t.Fatal(err, string(output))
			}
			if err := WithWriter(context.Background(), root, func(*Writer) error { return nil }); err != nil {
				t.Fatal(err)
			}
			requireForkProject(t, root, test.state, test.want)
		})
	}
}

// TestRecovery_RefusesALocalRuleEntryInAnOlderJournal preserves a journal whose format can't hold a local rule, one
// that removes a target other than a local rule's asset directory, and one listing parent directories it can't
// have created.
func TestRecovery_RefusesALocalRuleEntryInAnOlderJournal(t *testing.T) {
	assets := &Tree{Files: map[string][]byte{"notes.md": []byte("old notes")}}
	for name, record := range map[string]journalRecord{
		"format 5":              {FormatVersion: 5, Entries: []journalEntry{{LocalRuleAssets(forkFile), treeDigest(assets), treeDigest(assets), true}}},
		"removal":               {FormatVersion: 6, Entries: []journalEntry{{Vendor, treeDigest(assets), treeDigest(nil), true}}},
		"parent outside local/": {FormatVersion: 6, Entries: []journalEntry{{LocalRuleAssets(forkFile), treeDigest(assets), treeDigest(assets), true}}, Parents: []string{"vendor"}},
		"parent of no entry":    {FormatVersion: 6, Entries: []journalEntry{{LocalRuleAssets(forkFile), treeDigest(assets), treeDigest(assets), true}}, Parents: []string{"local/practices"}},
		"parents in format 5":   {FormatVersion: 5, Entries: []journalEntry{{LocalGroupMetadata("techs/go"), treeDigest(nil), treeDigest(assets), false}}, Parents: []string{"local/techs/go"}},
		"reversed ancestors":    {FormatVersion: 6, Entries: []journalEntry{{LocalRuleAssets(forkFile), treeDigest(assets), treeDigest(assets), true}}, Parents: []string{"local/techs/go/assets", "local/techs"}},
		"missing intermediate":  {FormatVersion: 6, Entries: []journalEntry{{LocalRuleAssets(forkFile), treeDigest(assets), treeDigest(assets), true}}, Parents: []string{"local/techs", "local/techs/go/assets"}},
	} {
		t.Run(name, func(t *testing.T) {
			root := openProject(t)
			writeForkProject(t, root)
			if err := root.Mkdir(transactionName, 0700); err != nil {
				t.Fatal(err)
			}
			if err := durableJSON(root, transactionName+"/journal.json", record); err != nil {
				t.Fatal(err)
			}
			projectCode(t, WithWriter(context.Background(), root, func(*Writer) error { return nil }), "recovery-required")
			if got := slices.Sorted(maps.Keys(localFiles(t, root))); !reflect.DeepEqual(got, slices.Sorted(maps.Keys(oldLocal))) {
				t.Fatalf("recovery changed local files: %v", got)
			}
		})
	}
}
