// Exercise authoring lifecycle, exclusive publication, and preservation of user-owned files.

package authoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestProjectAuthoringLifecycle covers idempotent init, local authoring, offline build, and config-only source addition.
func TestProjectAuthoringLifecycle(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(t.TempDir(), ".code-rules")
	options := Options{ConfigPath: filepath.Join(directory, "config.json")}
	result, err := InitializeProject(ctx, options)
	if err != nil || len(result.Files) != 3 {
		t.Fatal(result, err)
	}
	original, err := os.ReadFile(options.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	result, err = InitializeProject(ctx, options)
	if err != nil || len(result.Files) != 0 {
		t.Fatal(result, err)
	}
	metadata := rules.GroupMetadata{Name: " Go ", Description: " Local Go guidance. ", WhenToRead: " When editing Go. "}
	result, err = AddLocalGroup(ctx, "techs/go", metadata, options)
	if err != nil || len(result.Files) != 2 {
		t.Fatal(result, err)
	}
	stored, err := os.ReadFile(result.Files[0])
	if err != nil || !bytes.Contains(stored, []byte(`"name": "Go"`)) {
		t.Fatal(string(stored), err)
	}
	body := "# Return errors\n\nReturn failures to the caller.\n"
	ruleMeta := RuleMetadata{Title: "Return errors: always", Impact: "HIGH", ImpactDescription: "Preserve failures.", WhenToRead: "When calling fallible functions."}
	result, err = AddLocalRule(ctx, "techs/go/errors", ruleMeta, RuleOptions{Options: options, Body: &body})
	if err != nil || len(result.Files) != 1 {
		t.Fatal(result, err)
	}
	if _, err = project.Build(ctx, project.Options{ConfigPath: options.ConfigPath}); err != nil {
		t.Fatal(err)
	}
	if _, err = AddLocalRule(ctx, "techs/go/errors", ruleMeta, RuleOptions{Options: options, Body: &body}); err == nil {
		t.Fatal("overwrote existing rule")
	}
	document, err := os.ReadFile(result.Files[0])
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := rules.Parse(string(document), "techs/go/errors.md", "local")
	if err != nil || parsed.Title != ruleMeta.Title {
		t.Fatal(parsed, err)
	}
	result, err = AddLocalRule(ctx, "techs/go/draft", ruleMeta, RuleOptions{Options: options})
	if err != nil || !strings.Contains(result.Next, "Complete the draft") {
		t.Fatal(result, err)
	}
	draft, _ := os.ReadFile(result.Files[0])
	if !bytes.Contains(draft, []byte("### Validation")) || !bytes.Contains(draft, []byte("<State one concrete obligation.>")) {
		t.Fatal("canonical draft missing")
	}
	source := json.RawMessage(`{"repository":"https://github.com/acme/rules","version":">= 1.2.3, < 2.0.0","groups":"techs/*","exclude":{},"replace":{}}`)
	result, err = AddSource(ctx, "team", source, options)
	if err != nil || len(result.Files) != 1 {
		t.Fatal(result, err)
	}
	updated, _ := os.ReadFile(options.ConfigPath)
	if bytes.Equal(updated, original) || bytes.Contains(updated, []byte(`\u003e`)) {
		t.Fatal(string(updated))
	}
	if _, err = rules.ParseConfiguration(updated); err != nil {
		t.Fatal(err)
	}
	if _, err = AddSource(ctx, "team", source, options); err == nil {
		t.Fatal("duplicate alias accepted")
	}
	if _, err = os.Stat(filepath.Join(directory, "vendor")); !os.IsNotExist(err) {
		t.Fatal("source authoring fetched files", err)
	}
}

// TestAuthoringRefusesUnsafeAndIncompleteInput verifies failure paths preserve existing project bytes.
func TestAuthoringRefusesUnsafeAndIncompleteInput(t *testing.T) {
	for _, scenario := range []string{"invalid-config", "linked-local", "case-alias", "hardlink", "pending-recovery", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			directory := t.TempDir()
			options := Options{ConfigPath: filepath.Join(directory, "config.json")}
			if _, err := InitializeProject(ctx, options); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "invalid-config":
				os.WriteFile(options.ConfigPath, []byte(`{`), 0600)
			case "linked-local":
				os.RemoveAll(filepath.Join(directory, "local"))
				if err := os.Symlink(t.TempDir(), filepath.Join(directory, "local")); err != nil {
					t.Fatal(err)
				}
			case "case-alias":
				os.MkdirAll(filepath.Join(directory, "local", "Techs"), 0700)
			case "hardlink":
				if err := os.Link(options.ConfigPath, filepath.Join(directory, "alias")); err != nil {
					t.Fatal(err)
				}
			case "pending-recovery":
				os.Mkdir(filepath.Join(directory, ".code-rules-authoring-pending"), 0700)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			before, _ := os.ReadFile(options.ConfigPath)
			result, err := AddLocalGroup(ctx, "techs/go", rules.GroupMetadata{Name: "Go", Description: "Go.", WhenToRead: "When editing Go."}, options)
			if err == nil || result.Files != nil {
				t.Fatal("expected no partial result", result, err)
			}
			after, _ := os.ReadFile(options.ConfigPath)
			if !bytes.Equal(before, after) {
				t.Fatal("changed config on failure")
			}
			if _, err = os.Lstat(filepath.Join(directory, "local", "techs", "go", "_group.json")); !os.IsNotExist(err) {
				t.Fatal("created group on failure", err)
			}
		})
	}
}

// TestRuleAndGroupArePublishedTogether rejects collisions before creating either definition.
func TestRuleAndGroupArePublishedTogether(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	options := Options{ConfigPath: filepath.Join(directory, "config.json")}
	if _, err := InitializeProject(ctx, options); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(directory, "local/techs/go"), 0700)
	target := filepath.Join(directory, "local/techs/go/errors.md")
	os.WriteFile(target, []byte("authored content"), 0600)
	metadata := rules.GroupMetadata{Name: "Go", Description: "Go.", WhenToRead: "When editing Go."}
	body := "Return failures."
	_, err := AddLocalRule(ctx, "techs/go/errors", RuleMetadata{Title: "Errors", Impact: "HIGH", ImpactDescription: "Failures.", WhenToRead: "When calling."}, RuleOptions{Options: options, Body: &body, Group: &metadata})
	if err == nil {
		t.Fatal("accepted collision")
	}
	if _, err = os.Stat(filepath.Join(directory, "local/techs/go/_group.json")); !os.IsNotExist(err) {
		t.Fatal("published partial group", err)
	}
	data, _ := os.ReadFile(target)
	if string(data) != "authored content" {
		t.Fatal("lost prior bytes")
	}
}

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

// TestRuleRechecksGroupAfterPrompt prevents stale missing-group consent from overriding newly available imported guidance.
func TestRuleRechecksGroupAfterPrompt(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	options := Options{ConfigPath: filepath.Join(directory, "config.json")}
	if _, err := InitializeProject(ctx, options); err != nil {
		t.Fatal(err)
	}
	source := json.RawMessage(`{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":{},"replace":{}}`)
	if _, err := AddSource(ctx, "team", source, options); err != nil {
		t.Fatal(err)
	}
	if available, err := HasLocalRuleGroup(ctx, "techs/go", options); err != nil || available {
		t.Fatal(available, err)
	}
	data, err := os.ReadFile(options.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	config, err := rules.ParseConfiguration(data)
	if err != nil {
		t.Fatal(err)
	}
	vendor, err := project.EncodeSnapshots(config, map[string]project.Snapshot{"team": {Repository: config.Sources[0].Repository, Ref: "v1.0.0", Commit: strings.Repeat("a", 40), Groups: []string{"techs/go"}, Selection: config.Sources[0].Groups, Files: map[string][]byte{"rule-library.json": []byte(`{"formatVersion":1}`), "techs/go/_group.json": []byte(`{"name":"Go","description":"Imported guidance.","whenToRead":"When editing Go."}`)}}})
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range vendor {
		target := filepath.Join(directory, "vendor", name)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	metadata := rules.GroupMetadata{Name: "Unintended override", Description: "Prompt answer.", WhenToRead: "When editing."}
	body := "Return errors."
	result, err := AddLocalRule(ctx, "techs/go/errors", RuleMetadata{Title: "Errors", Impact: "HIGH", ImpactDescription: "Failures.", WhenToRead: "When calling."}, RuleOptions{Options: options, Body: &body, Group: &metadata})
	if err != nil || len(result.Files) != 1 {
		t.Fatal(result, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "local/techs/go/_group.json")); !os.IsNotExist(err) {
		t.Fatal("stale consent created local override", err)
	}
}
