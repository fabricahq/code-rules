// Validate real Git libraries, retained bytes, unsupported objects, and all-or-nothing imports.

package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/gitfixture"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// libraryFiles supplies original rules, complete owned assets, shared dependencies, and exact license bytes.
func libraryFiles() map[string][]byte {
	return map[string][]byte{
		"rule-library.json": []byte(`{"formatVersion":1,"license":{"file":"LICENSE","notices":["NOTICE"]}}`),
		"LICENSE":           []byte("Original license\r\n"), "NOTICE": []byte("Original notice\r\n"),
		"techs/go/_group.json":             []byte(`{"name":"Go","description":"Go guidance.","whenToRead":"When editing Go."}`),
		"techs/go/errors.md":               []byte("---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Preserve failures.\nwhenToRead: When calling functions.\n---\n# Return errors\n\n[Shared](/assets/guide.md)\n"),
		"techs/go/assets/errors/image.bin": {0, 255, 1, 128},
		"assets/guide.md":                  []byte("[More](more.md)\n"), "assets/more.md": []byte("[Guide](guide.md)\n"),
		"techs/rust/_group.json": []byte(`{"name":"Rust","description":"Rust guidance.","whenToRead":"When editing Rust."}`),
		"techs/rust/bad.md":      []byte("Invalid unselected rule."),
		"README.md":              []byte("Ignored repository documentation."),
	}
}

// newLibraryFixture commits original library bytes and owns cleanup for the test.
func newLibraryFixture(t *testing.T, files map[string][]byte) *gitfixture.Fixture {
	t.Helper()
	f, err := gitfixture.New(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	return f
}

// libraryConfig parses the same configuration boundary used by the application.
func libraryConfig(t *testing.T, repository string) rules.Configuration {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"schemaVersion": 1, "sources": map[string]any{"team": map[string]any{"repository": repository, "version": ">= 1.0.0", "groups": []string{"techs/go"}, "exclude": map[string]string{}, "replace": map[string]any{}}}})
	config, err := rules.ParseConfiguration(raw)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

// TestImportLibraryOriginalBytes verifies real Git adoption, complete assets, exact terms, and persisted round trips.
func TestImportLibraryOriginalBytes(t *testing.T) {
	files := libraryFiles()
	f := newLibraryFixture(t, files)
	config := libraryConfig(t, f.Repository)
	imported, err := ImportLibraries(context.Background(), config, Options{GitPath: f.GitPath, Environment: f.Environment})
	if err != nil {
		t.Fatal(err)
	}
	item := imported["team"]
	if item.Snapshot.Commit != f.LatestCommit || item.Snapshot.Tag != "v1.2.0" || item.Catalog.License == nil {
		t.Fatal("lost provenance or license")
	}
	for name, data := range item.Snapshot.Files {
		if !bytes.Equal(data, files[name]) {
			t.Fatalf("changed %s", name)
		}
	}
	for _, name := range []string{"LICENSE", "NOTICE", "assets/guide.md", "assets/more.md", "techs/go/assets/errors/image.bin", "techs/go/errors.md"} {
		if _, ok := item.Snapshot.Files[name]; !ok {
			t.Fatalf("lost %s", name)
		}
	}
	if _, ok := item.Snapshot.Files["techs/rust/bad.md"]; ok {
		t.Fatal("adopted unselected content")
	}
	encoded, err := project.EncodeSnapshots(config, map[string]project.Snapshot{"team": item.Snapshot})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := project.DecodeSnapshots(config, encoded)
	if err != nil || !bytes.Equal(decoded["team"].Files["LICENSE"], files["LICENSE"]) {
		t.Fatal("snapshot round trip", err)
	}
	config.Sources[0].Groups = rules.GroupSelection{Pattern: "*"}
	if result, err := ImportLibraries(context.Background(), config, Options{GitPath: f.GitPath, Environment: f.Environment}); err == nil || result != nil {
		t.Fatal("wildcard did not validate invalid second group")
	}
}

// TestImportRejectsSelectedContent checks real repository corruption and preserves all-or-nothing results.
func TestImportRejectsSelectedContent(t *testing.T) {
	for _, kind := range []string{"lfs", "symlink", "submodule", "oversize", "escape", "rule-link", "collision", "missing-manifest"} {
		t.Run(kind, func(t *testing.T) {
			files := libraryFiles()
			switch kind {
			case "lfs":
				files["techs/go/assets/errors/image.bin"] = []byte("version https://git-lfs.github.com/spec/v1\noid sha256:abc\nsize 1\n")
			case "oversize":
				files["techs/go/assets/errors/image.bin"] = bytes.Repeat([]byte("x"), maxBlobBytes+1)
			case "escape":
				files["assets/guide.md"] = []byte("[Escape](../../outside.md)\n")
			case "rule-link":
				files["assets/guide.md"] = []byte("[Rule](/techs/rust/bad.md)\n")
			case "missing-manifest":
				delete(files, "rule-library.json")
			}
			f := newLibraryFixture(t, files)
			if kind == "symlink" {
				name := filepath.Join(f.Directory, "repository", "techs/go/assets/errors/link")
				if err := os.Symlink("image.bin", name); err != nil {
					t.Fatal(err)
				}
				if _, err := f.Command(context.Background(), "add", "--all"); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "submodule" {
				if _, err := f.Command(context.Background(), "update-index", "--add", "--cacheinfo", "160000,"+f.FirstCommit+",techs/go/assets/errors/submodule"); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "collision" {
				object, err := f.Command(context.Background(), "rev-parse", "HEAD:techs/go/assets/errors/image.bin")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.Command(context.Background(), "update-index", "--add", "--cacheinfo", "100644,"+strings.TrimSpace(object)+",techs/go/assets/errors/Image.bin"); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "symlink" || kind == "submodule" || kind == "collision" {
				if _, err := f.Command(context.Background(), "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Unsupported object"); err != nil {
					t.Fatal(err)
				}
				if _, err := f.Command(context.Background(), "tag", "-f", "v1.2.0"); err != nil {
					t.Fatal(err)
				}
			}
			result, err := ImportLibraries(context.Background(), libraryConfig(t, f.Repository), Options{GitPath: f.GitPath, Environment: f.Environment})
			if err == nil || result != nil {
				t.Fatal("invalid content returned usable import")
			}
			if strings.Contains(err.Error(), "license:") || strings.Contains(err.Error(), "configuration:") {
				t.Fatalf("fixture failed before the selected-content check: %v", err)
			}
		})
	}
}

// TestImportMultipleSources verifies independent source identity and no partial result when the second source fails.
func TestImportMultipleSources(t *testing.T) {
	a := newLibraryFixture(t, libraryFiles())
	bfiles := libraryFiles()
	bfiles["NOTICE"] = []byte("Second source notice\n")
	b := newLibraryFixture(t, bfiles)
	environment, err := a.Route(map[string]*gitfixture.Fixture{"alpha": a, "beta": b})
	if err != nil {
		t.Fatal(err)
	}
	config := libraryConfig(t, "git@fixture.invalid:alpha")
	first := config.Sources[0]
	first.Name = "alpha"
	second := first
	second.Name = "beta"
	second.Repository = "git@fixture.invalid:beta"
	config.Sources = []rules.Source{first, second}
	result, err := ImportLibraries(context.Background(), config, Options{GitPath: a.GitPath, Environment: environment})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || string(result["beta"].Snapshot.Files["NOTICE"]) != "Second source notice\n" {
		t.Fatal("sources mixed")
	}
	config.Sources[1].Groups = rules.GroupSelection{Pattern: "*"}
	result, err = ImportLibraries(context.Background(), config, Options{GitPath: a.GitPath, Environment: environment})
	if err == nil || result != nil || !strings.Contains(err.Error(), "beta") {
		t.Fatal("partial or unattributed failure", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := ImportLibraries(ctx, config, Options{}); result != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("lost cancellation", err)
	}
}

// TestTreeFramingRejectsUnsafeRecords probes framing before any path can become a filesystem request.
func TestTreeFramingRejectsUnsafeRecords(t *testing.T) {
	oid := strings.Repeat("a", 40)
	for _, record := range []string{"100644 blob " + oid + " 1\t../escape\x00", "100644 blob " + oid + " 1\tx", "100644 blob " + oid + " -1\tx\x00", "100644 blob " + oid + " 1\tx\x00" + "100644 blob " + oid + " 1\tx/y\x00"} {
		if _, err := parseTree([]byte(record)); err == nil {
			t.Fatalf("accepted %q", record)
		}
	}
}

// cancelMetadata cancels on the metadata boundary before the actual Git source observes its context.
type cancelMetadata struct {
	*gitFiles
	cancel context.CancelFunc
}

// Lstat injects cancellation at the boundary where the loader wraps source errors.
func (c cancelMetadata) Lstat(name string) (fs.FileInfo, error) {
	c.cancel()
	return c.gitFiles.Lstat(name)
}

// TestCancellationDuringGitMetadata preserves errors.Is through shared loader metadata inspection.
func TestCancellationDuringGitMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := cancelMetadata{gitFiles: &gitFiles{ctx: ctx}, cancel: cancel}
	_, err := library.LoadSource(ctx, input, "team", rules.GroupSelection{Groups: []string{}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cancellation identity: %v", err)
	}
}

// TestGitTreeAndRetainedLimits exercises allocation guards before any oversized blob is requested.
func TestGitTreeAndRetainedLimits(t *testing.T) {
	oid := strings.Repeat("a", 40)
	var records strings.Builder
	for i := 0; i < maxTreeFiles+1; i++ {
		fmt.Fprintf(&records, "100644 blob %s 0\tfile-%d\x00", oid, i)
	}
	_, err := parseTree([]byte(records.String()))
	requireCode(t, err, "limit-exceeded")
	_, err = parseTree(bytes.Repeat([]byte{0}, maxTreeBytes+1))
	requireCode(t, err, "limit-exceeded")
	g := &gitFiles{ctx: context.Background(), total: maxRetainedBytes, entries: map[string]treeEntry{"x": {name: "x", mode: 0644, size: 1}}}
	_, err = g.ReadFile("x")
	requireCode(t, err, "limit-exceeded")
}

// TestGitBlobFraming rejects truncated, wrong-object and trailing data from a faulty Git process.
func TestGitBlobFraming(t *testing.T) {
	oid := strings.Repeat("a", 40)
	for _, output := range []string{oid + " blob 1\nx", strings.Repeat("b", 40) + " blob 1\nx\n", oid + " blob 1\nx\nextra"} {
		dir := t.TempDir()
		script := filepath.Join(dir, "git")
		if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf %s "+gitfixture.Quote(output)+"\n"), 0700); err != nil {
			t.Fatal(err)
		}
		g := &gitFiles{ctx: context.Background(), revision: &Revision{directory: dir, runner: gitRunner{executable: script, environment: gitEnvironment(os.Environ())}}, entries: map[string]treeEntry{"x": {name: "x", mode: 0644, size: 1, object: oid}}}
		_, err := g.ReadFile("x")
		requireCode(t, err, "git-failed")
	}
}

// TestCatalogMutationPreservesSnapshot keeps verified bytes and selection independent from editable catalog data.
func TestCatalogMutationPreservesSnapshot(t *testing.T) {
	files := libraryFiles()
	fixture := newLibraryFixture(t, files)
	config := libraryConfig(t, fixture.Repository)
	result, err := ImportLibraries(context.Background(), config, Options{GitPath: fixture.GitPath, Environment: fixture.Environment})
	if err != nil {
		t.Fatal(err)
	}
	item := result["team"]
	item.Catalog.SupportingFiles["NOTICE"][0] = 'X'
	item.Catalog.Selection.Groups[0] = "techs/rust"
	if !bytes.Equal(item.Snapshot.Files["NOTICE"], files["NOTICE"]) || item.Snapshot.Selection.Groups[0] != "techs/go" {
		t.Fatal("catalog mutation altered verified snapshot")
	}
	encoded, err := project.EncodeSnapshots(config, map[string]project.Snapshot{"team": item.Snapshot})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := project.DecodeSnapshots(config, encoded)
	if err != nil || !bytes.Equal(decoded["team"].Files["NOTICE"], files["NOTICE"]) {
		t.Fatal("snapshot no longer retains original Git bytes", err)
	}
}
