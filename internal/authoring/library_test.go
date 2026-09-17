// Verify complete library authoring, original terms, read-only checks, and unused content validation.

package authoring

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestLibraryLifecycle authors a licensed library, validates counts without writes, and rejects marked drafts.
func TestLibraryLifecycle(t *testing.T) {
	ctx := context.Background()
	options := LibraryOptions{Directory: t.TempDir()}
	notice := "Notice\r\n"
	terms := &LibraryTerms{SPDXExpression: "MIT", License: "Original terms\r\n", Notice: &notice}
	result, err := InitializeLibrary(ctx, options, terms)
	if err != nil || len(result.Files) != 4 {
		t.Fatal(result, err)
	}
	data, _ := os.ReadFile(filepath.Join(options.Directory, "LICENSE.md"))
	if string(data) != terms.License {
		t.Fatal("changed terms")
	}
	result, err = InitializeLibrary(ctx, options, nil)
	if err != nil || len(result.Files) != 0 {
		t.Fatal(result, err)
	}
	if _, err = InitializeLibrary(ctx, options, terms); err == nil {
		t.Fatal("overwrote terms")
	}
	metadata := rules.GroupMetadata{Name: "Go", Description: "Go guidance.", WhenToRead: "When editing Go."}
	if _, err = AddLibraryGroup(ctx, "techs/go", metadata, options); err != nil {
		t.Fatal(err)
	}
	body := "# Return failures\n\nReturn failures to the caller.\n"
	rule := RuleMetadata{Title: "Return failures", Impact: "HIGH", ImpactDescription: "Preserve failures.", WhenToRead: "When calling fallible functions."}
	if _, err = AddLibraryRule(ctx, "techs/go/errors", rule, LibraryRuleOptions{LibraryOptions: options, Body: &body}); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(options.Directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	before, err := project.ReadTree(ctx, root, ".")
	if err != nil {
		t.Fatal(err)
	}
	check, err := CheckLibrary(ctx, options)
	if err != nil || check.Groups != 1 || check.Rules != 1 || len(check.Warnings) != 0 {
		t.Fatal(check, err)
	}
	after, _ := project.ReadTree(ctx, root, ".")
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if !bytes.Equal(a, b) {
		t.Fatal("check modified library")
	}
	if _, err = AddLibraryRule(ctx, "techs/go/draft", rule, LibraryRuleOptions{LibraryOptions: options}); err != nil {
		t.Fatal(err)
	}
	if _, err = CheckLibrary(ctx, options); err == nil || !strings.Contains(err.Error(), "draft") {
		t.Fatal("draft accepted", err)
	}
}

// TestLibraryCheckUnusedContent enforces every library-owned file even when no selected rule references it.
func TestLibraryCheckUnusedContent(t *testing.T) {
	for _, scenario := range []string{"empty", "orphan", "missing-link", "unrelated", "cross-rule", "malformed-rule", "symlink", "binary"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			options := LibraryOptions{Directory: t.TempDir()}
			if _, err := InitializeLibrary(ctx, options, nil); err != nil {
				t.Fatal(err)
			}
			write := func(name, content string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(filepath.Join(options.Directory, name)), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(options.Directory, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			switch scenario {
			case "orphan":
				write("techs/go/assets/missing/image.bin", "binary")
			case "missing-link":
				write("assets/unreferenced.md", "[missing](absent.md)")
			case "unrelated":
				write(".git/objects/example", "arbitrary repository bytes")
				write("docs/design.txt", "unrelated documentation")
			case "cross-rule":
				write("assets/shared.md", "[rule](/techs/go/errors.md)")
			case "malformed-rule":
				write("techs/go/_group.json", `{"name":"Go","description":"Go.","whenToRead":"Go."}`)
				write("techs/go/errors.md", "Missing frontmatter")
			case "symlink":
				if err := os.Symlink(t.TempDir(), filepath.Join(options.Directory, "assets")); err != nil {
					t.Fatal(err)
				}
			case "binary":
				write("assets/image.bin", string([]byte{0, 255, 254, 42}))
			}
			result, err := CheckLibrary(ctx, options)
			valid := scenario == "empty" || scenario == "unrelated" || scenario == "binary"
			if valid {
				if err != nil || result.Groups != 0 || result.Rules != 0 || len(result.Warnings) != 1 {
					t.Fatal(result, err)
				}
			} else if err == nil {
				t.Fatal("invalid library accepted", result)
			}
		})
	}
}

// TestLibraryInitializationPreservesConflicts refuses a term-file collision before publishing the manifest.
func TestLibraryInitializationPreservesConflicts(t *testing.T) {
	directory := t.TempDir()
	os.WriteFile(filepath.Join(directory, "LICENSE.md"), []byte("existing terms"), 0600)
	_, err := InitializeLibrary(context.Background(), LibraryOptions{Directory: directory}, &LibraryTerms{SPDXExpression: "MIT", License: "different"})
	if err == nil {
		t.Fatal("accepted terms collision")
	}
	if _, err = os.Stat(filepath.Join(directory, "rule-library.json")); !os.IsNotExist(err) {
		t.Fatal("partial manifest", err)
	}
	data, _ := os.ReadFile(filepath.Join(directory, "LICENSE.md"))
	if string(data) != "existing terms" {
		t.Fatal("overwrote terms")
	}
}

// TestLibraryCheckRejectsConcurrentEdits covers files that the selected catalog does not reread.
func TestLibraryCheckRejectsConcurrentEdits(t *testing.T) {
	for _, name := range []string{"assets/guide.md", "assets/new.md", "rule-library.json", "LICENSE.md"} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			ctx := context.Background()
			_, err := InitializeLibrary(ctx, LibraryOptions{Directory: directory}, &LibraryTerms{SPDXExpression: "MIT", License: "Original terms"})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(directory, "assets"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "assets/guide.md"), []byte("Original guidance"), 0600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			before, _, err := libraryCheckInput(ctx, root)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, name), []byte("[changed](missing.md)"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := requireLibraryUnchanged(ctx, root, before); err == nil || !strings.Contains(err.Error(), "changed during validation") {
				t.Fatal(err)
			}
		})
	}
}

// TestCapturedLibraryIgnoresLaterEdits keeps catalog counts and inventory validation on exactly the same bytes.
func TestCapturedLibraryIgnoresLaterEdits(t *testing.T) {
	ctx := context.Background()
	options := LibraryOptions{Directory: t.TempDir()}
	if _, err := InitializeLibrary(ctx, options, nil); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(options.Directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	snapshot, _, err := libraryCheckInput(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(options.Directory, "techs/go"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(options.Directory, "techs/go/_group.json"), []byte(`{"name":"Go","description":"Go guidance","whenToRead":"When editing Go"}`), 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := library.LoadSource(ctx, capturedLibrary{snapshot}, "library", rules.GroupSelection{Pattern: "*"})
	if err != nil || len(catalog.Groups) != 0 {
		t.Fatal(catalog, err)
	}
	if err := requireLibraryUnchanged(ctx, root, snapshot); err == nil {
		t.Fatal("missed new group")
	}
	if err := os.Remove(filepath.Join(options.Directory, "techs/go/_group.json")); err != nil {
		t.Fatal(err)
	}
	empty, _, err := libraryCheckInput(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := library.LoadSource(ctx, capturedLibrary{empty}, "library", rules.GroupSelection{Pattern: "*"}); err == nil {
		t.Fatal("empty group without metadata accepted")
	}
}

// TestLibraryCheckBoundsUnusedAssets rejects oversized unreferenced content without modifying it.
func TestLibraryCheckBoundsUnusedAssets(t *testing.T) {
	options := LibraryOptions{Directory: t.TempDir()}
	if _, err := InitializeLibrary(context.Background(), options, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(options.Directory, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(options.Directory, "assets/archive.bin")
	data := make([]byte, 9*1024*1024)
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckLibrary(context.Background(), options); err == nil || !strings.Contains(err.Error(), "limits") {
		t.Fatal("oversized unused asset accepted", err)
	}
	after, err := os.ReadFile(name)
	if err != nil || !bytes.Equal(data, after) {
		t.Fatal("check changed asset", err)
	}
}
