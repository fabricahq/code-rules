// Exercise real directory reads, selection, byte preservation, and unsafe-file failures.

package library_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

const metadata = `{"name":" Go ","description":"Go rules","whenToRead":"When writing Go."}`
const document = "---\ntitle: Handle errors\nimpact: HIGH\nimpactDescription: Avoid losing failures.\nwhenToRead: When calling fallible functions.\n---\n# Handle errors\n\nReturn the error.\n"

// fixture creates an isolated on-disk library and registers both directory and handle cleanup.
func fixture(t *testing.T, files map[string]string) (string, *os.Root) {
	t.Helper()
	directory := t.TempDir()
	for path, text := range files {
		target := filepath.Join(directory, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	// Close the caller-owned root before the fixture directory is removed.
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	return directory, root
}

// validFiles provides original authored fixtures rather than production source material.
func validFiles() map[string]string {
	return map[string]string{"rule-library.json": `{"formatVersion":1}`, "techs/go/_group.json": metadata, "techs/go/errors.md": document, "practices/testing/_group.json": metadata}
}

// TestLoadCatalog checks wildcard expansion, empty groups, exact bytes, and explicit empty selection.
func TestLoadCatalog(t *testing.T) {
	files := validFiles()
	files["techs/go/assets/errors/image.bin"] = "\xff\x00"
	_, root := fixture(t, files)
	got, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Groups) != 2 || got.Groups[0].ID != "practices/testing" || len(got.Groups[0].Rules) != 0 || got.Groups[1].Rules[0].ID != "team:techs/go/errors" || got.Groups[1].Metadata.Name != "Go" {
		t.Fatalf("unexpected catalog %+v", got)
	}
	if got.Groups[1].Rules[0].Document != document {
		t.Fatal("changed original bytes")
	}
	if _, ok := got.SupportingFiles["techs/go/errors.md"]; ok {
		t.Fatal("rule document duplicated in supporting files")
	}
	if !reflect.DeepEqual(got.Paths(), []string{"practices/testing/_group.json", "rule-library.json", "techs/go/_group.json", "techs/go/assets/errors/image.bin", "techs/go/errors.md"}) {
		t.Fatalf("incomplete read inventory: %v", got.Paths())
	}
	if string(got.SupportingFiles["techs/go/assets/errors/image.bin"]) != "\xff\x00" {
		t.Fatal("catalog lost binary asset bytes")
	}
	techs, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "techs/*"})
	if err != nil || len(techs.Groups) != 1 {
		t.Fatalf("techs: %+v, %v", techs, err)
	}
	empty, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Groups: []string{}})
	if err != nil || len(empty.Groups) != 0 || !reflect.DeepEqual(empty.Paths(), []string{"rule-library.json"}) {
		t.Fatalf("empty: %+v, %v", empty, err)
	}
}

// TestLoadRejectsInvalidLibraries checks partial-result prevention for selected content failures.
func TestLoadRejectsInvalidLibraries(t *testing.T) {
	for _, test := range []struct {
		name, path, text string
		remove           bool
	}{
		{"missing manifest", "rule-library.json", "", true},
		{"missing metadata", "techs/go/_group.json", "", true},
		{"invalid rule", "techs/go/errors.md", "no frontmatter", false},
		{"blank body", "techs/go/errors.md", strings.Split(document, "# Handle")[0], false},
		{"invalid utf8", "techs/go/errors.md", "\xff", false},
		{"unsupported file", "techs/go/data.json", "{}", false},
		{"lfs pointer", "techs/go/errors.md", "version https://git-lfs.github.com/spec/v1\noid sha256:example", false},
	} {
		// Every malformed library is exercised through the filesystem API.
		t.Run(test.name, func(t *testing.T) {
			files := validFiles()
			if test.remove {
				delete(files, test.path)
			} else {
				files[test.path] = test.text
			}
			_, root := fixture(t, files)
			got, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "*"})
			var validation *rules.ValidationError
			if !errors.As(err, &validation) || !reflect.DeepEqual(got, library.Catalog{}) {
				t.Fatalf("got %+v, %v", got, err)
			}
		})
	}
}

// TestLoadSymlinksAndCancellation rejects linked selected entries and preserves context cancellation identity.
func TestLoadSymlinksAndCancellation(t *testing.T) {
	directory, root := fixture(t, validFiles())
	if err := os.Symlink(filepath.Join(t.TempDir(), "private.md"), filepath.Join(directory, "techs/go/link.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "*"}); err == nil {
		t.Fatal("accepted symlink")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := library.Load(ctx, root, "team", rules.GroupSelection{Pattern: "*"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Groups: []string{"techs/missing"}}); err == nil {
		t.Fatal("accepted missing explicit group")
	}
}

// TestLoadTermsAndLimits retains binary licenses and rejects files above the exact byte bound.
func TestLoadTermsAndLimits(t *testing.T) {
	files := validFiles()
	files["rule-library.json"] = `{"formatVersion":1,"license":{"file":"LICENSE","notices":["NOTICE"],"spdxExpression":"MIT"}}`
	files["LICENSE"] = "\xff\x00\r\n"
	files["NOTICE"] = "Notice\r\n"
	_, root := fixture(t, files)
	got, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "*"})
	if err != nil || string(got.SupportingFiles["LICENSE"]) != files["LICENSE"] || string(got.SupportingFiles["NOTICE"]) != files["NOTICE"] {
		t.Fatalf("terms changed: %v", err)
	}
	for _, size := range []int{8 * 1024 * 1024, 8*1024*1024 + 1} {
		bounded := map[string]string{"rule-library.json": files["rule-library.json"], "LICENSE": strings.Repeat("x", size), "NOTICE": ""}
		_, root := fixture(t, bounded)
		_, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Groups: []string{}})
		if (err != nil) != (size > 8*1024*1024) {
			t.Fatalf("size %d: %v", size, err)
		}
	}
}

// TestLoadLeavesFilesUnchanged verifies that catalog loading does not alter the fixture tree.
func TestLoadLeavesFilesUnchanged(t *testing.T) {
	files := validFiles()
	directory, root := fixture(t, files)
	if _, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "*"}); err != nil {
		t.Fatal(err)
	}
	count := 0
	// Compare every file after loading, including unselected content, with its authored bytes.
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		count++
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		expected, ok := files[filepath.ToSlash(relative)]
		if !ok || string(data) != expected {
			t.Errorf("unexpected or changed file: %s", relative)
		}
		return nil
	})
	if err != nil || count != len(files) {
		t.Fatalf("changed inventory: count %d, %v", count, err)
	}
}

// TestLoadRejectsDirectoryLinksAndInvalidSelections covers entry points before file interpretation.
func TestLoadRejectsDirectoryLinksAndInvalidSelections(t *testing.T) {
	directory, root := fixture(t, validFiles())
	if err := os.Symlink(filepath.Join(directory, "techs/go"), filepath.Join(directory, "techs/rust")); err != nil {
		t.Fatal(err)
	}
	if _, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "techs/*"}); err == nil {
		t.Fatal("accepted linked group")
	}
	for _, selection := range []rules.GroupSelection{{}, {Pattern: "other/*"}, {Pattern: "*", Groups: []string{}}, {Groups: []string{"techs/go", "techs/go"}}} {
		if _, err := library.Load(context.Background(), root, "team", selection); err == nil {
			t.Fatalf("accepted selection %+v", selection)
		}
	}
}

// TestLoadOwnsOriginalDocuments preserves authored text after source files change.
func TestLoadOwnsOriginalDocuments(t *testing.T) {
	files := validFiles()
	files["techs/go/errors.md"] = strings.ReplaceAll(strings.Replace(document, "title: Handle errors", "# Author comment\ntitle: 'Handle errors'\ntags: [errors, reliability]", 1), "\n", "\r\n")
	files["techs/go/other.md"] = document + "Keep  spacing.  \n"
	directory, root := fixture(t, files)
	got, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "techs/*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Groups) != 1 || len(got.Groups[0].Rules) != 2 {
		t.Fatalf("unexpected groups or rules: %+v", got.Groups)
	}
	for _, rule := range got.Groups[0].Rules {
		if err := os.WriteFile(filepath.Join(directory, filepath.FromSlash(rule.Path)), []byte("changed after load"), 0600); err != nil {
			t.Fatal(err)
		}
		if rule.Document != files[rule.Path] {
			t.Fatalf("original document lost: %s", rule.Path)
		}
		if _, duplicated := got.SupportingFiles[rule.Path]; duplicated {
			t.Fatalf("rule also stored in supporting files: %s", rule.Path)
		}
		sections, err := rules.SplitDocument(rule.Document, rule.ID)
		if err != nil || !strings.Contains(sections.Frontmatter, "title:") || !strings.Contains(sections.Body, "Return the error.") {
			t.Fatalf("original sections unavailable: %+v, %v", sections, err)
		}
	}
	if string(got.SupportingFiles["techs/go/_group.json"]) != metadata {
		t.Fatal("supporting metadata bytes changed")
	}
}

// TestLoadRejectsReservedSource keeps imported identities separate from project rules.
func TestLoadRejectsReservedSource(t *testing.T) {
	_, root := fixture(t, validFiles())
	got, err := library.Load(context.Background(), root, "local", rules.GroupSelection{Pattern: "*"})
	var validation *rules.ValidationError
	if !errors.As(err, &validation) || validation.Location != "source" || !strings.Contains(err.Error(), "reserved") || got.Groups != nil {
		t.Fatalf("reserved source returned %+v, %v", got, err)
	}
}
