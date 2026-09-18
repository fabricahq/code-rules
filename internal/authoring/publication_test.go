// Exercise exclusive authoring publication, rollback, and preservation of concurrent user edits.

package authoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/fabricahq/code-rules/internal/project"
)

// TestPublishAuthoredChecksPriorBytes prevents stale source edits from replacing newer user changes.
func TestPublishAuthoredChecksPriorBytes(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	root.WriteFile("config.json", []byte("newer"), 0600)
	err = project.WithWriter(context.Background(), root, func(_ *project.Writer) error {
		return publishAuthored(context.Background(), root, []authoredFile{{name: "config.json", data: []byte("replacement"), before: []byte("old")}})
	})
	var failure *project.Error
	if !errors.As(err, &failure) || failure.Code != "concurrent-change" {
		t.Fatal(err)
	}
	data, _ := root.ReadFile("config.json")
	if string(data) != "newer" {
		t.Fatal("overwrote newer bytes")
	}
}

// TestPublicationRollbackPreservesEdits injects a real installation failure after the first exclusive create.
func TestPublicationRollbackPreservesEdits(t *testing.T) {
	for _, edit := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchanged", true: "edited"}[edit], func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			root.Mkdir(".code-rules-authoring-test", 0700)
			injected := errors.New("second publication failed")
			operation := &publication{ctx: context.Background(), root: root, stage: ".code-rules-authoring-test"}
			operation.link = func(from, to string) error {
				if to == "second" {
					if edit {
						if err := root.WriteFile("first", []byte("editor content"), 0600); err != nil {
							t.Fatal(err)
						}
					}
					return injected
				}
				return root.Link(from, to)
			}
			err = operation.run([]authoredFile{{name: "first", data: []byte("created")}, {name: "second", data: []byte("second")}})
			if !errors.Is(err, injected) {
				t.Fatal(err)
			}
			data, err := root.ReadFile("first")
			if edit {
				if err != nil || string(data) != "editor content" {
					t.Fatal(string(data), err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("failed to roll back", err)
			}
			if _, err = root.Stat(".code-rules-authoring-test"); !os.IsNotExist(err) {
				t.Fatal("stage leaked", err)
			}
		})
	}
}

// TestReplacementCollisionRetainsRecovery preserves both a newly recreated target and the claimed original.
func TestReplacementCollisionRetainsRecovery(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	root.WriteFile("config.json", []byte("original"), 0600)
	root.Mkdir(".code-rules-authoring-test", 0700)
	operation := &publication{ctx: context.Background(), root: root, stage: ".code-rules-authoring-test"}
	operation.link = func(from, to string) error {
		if err := root.WriteFile(to, []byte("editor replacement"), 0600); err != nil {
			t.Fatal(err)
		}
		return root.Link(from, to)
	}
	err = operation.run([]authoredFile{{name: "config.json", data: []byte("proposed"), before: []byte("original")}})
	var failure *project.Error
	if !errors.As(err, &failure) || failure.Code != "recovery-required" {
		t.Fatal(err)
	}
	data, _ := root.ReadFile("config.json")
	old, _ := root.ReadFile(".code-rules-authoring-test/previous-0")
	if string(data) != "editor replacement" || string(old) != "original" {
		t.Fatal(string(data), string(old))
	}
}

// TestCommittedCleanupWarning keeps committed file paths visible when post-publication cleanup fails.
func TestCommittedCleanupWarning(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(map[bool]string{false: "creation", true: "replacement"}[replace], func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			stage := ".code-rules-authoring-cleanup"
			if err = root.Mkdir(stage, 0700); err != nil {
				t.Fatal(err)
			}
			file := authoredFile{name: "config.json", data: []byte("new")}
			if replace {
				file.before = []byte("old")
				if err = root.WriteFile(file.name, file.before, 0600); err != nil {
					t.Fatal(err)
				}
			}
			cleanupErr := errors.New("injected stage cleanup failure")
			operation := &publication{ctx: context.Background(), root: root, stage: stage, link: root.Link, removeStage: func(string) error { return cleanupErr }}
			err = operation.run([]authoredFile{file})
			if !errors.Is(err, cleanupErr) || !publicationComplete(err) {
				t.Fatal("lost committed status", err)
			}
			result, err := finishAuthoring(root, []authoredFile{file}, "Review committed files.", publicationComplete(err), err)
			if err != nil || len(result.Files) != 1 || len(result.Warnings) != 1 {
				t.Fatal(result, err)
			}
			data, _ := root.ReadFile(file.name)
			if string(data) != "new" {
				t.Fatal("lost committed bytes")
			}
			encoded, _ := json.Marshal(result)
			if !bytes.Contains(encoded, []byte(`"warnings"`)) {
				t.Fatal("cleanup hidden from JSON", string(encoded))
			}
			if err = validatePublication(root, []authoredFile{file}); err == nil {
				t.Fatal("ignored pending cleanup stage")
			}
		})
	}
}
