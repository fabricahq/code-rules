// Exercise project initialization, local authoring, and imported group selection.

package project

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestProjectAuthoringLifecycle covers idempotent init, local authoring, offline build, and config-only source addition.
func TestProjectAuthoringLifecycle(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(t.TempDir(), ".code-rules")
	options := Options{Directory: filepath.Dir(directory)}
	result, err := Initialize(ctx, options)
	if err != nil || len(result.Files) != 3 {
		t.Fatal(result, err)
	}
	original, err := os.ReadFile(filepath.Join(directory, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err = Initialize(ctx, options)
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
	ruleMeta := rules.RuleMetadata{Title: "Return errors: always", Impact: "HIGH", ImpactDescription: "Preserve failures.", WhenToRead: "When calling fallible functions."}
	result, err = AddLocalRule(ctx, "techs/go/errors", ruleMeta, RuleOptions{Options: options, Body: &body})
	if err != nil || len(result.Files) != 1 {
		t.Fatal(result, err)
	}
	if _, err = Build(ctx, options); err != nil {
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
	updated, _ := os.ReadFile(filepath.Join(directory, "config.json"))
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
			directory := filepath.Join(t.TempDir(), ".code-rules")
			options := Options{Directory: filepath.Dir(directory)}
			if _, err := Initialize(ctx, options); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "invalid-config":
				os.WriteFile(filepath.Join(directory, "config.json"), []byte(`{`), 0600)
			case "linked-local":
				os.RemoveAll(filepath.Join(directory, "local"))
				if err := os.Symlink(t.TempDir(), filepath.Join(directory, "local")); err != nil {
					t.Fatal(err)
				}
			case "case-alias":
				os.MkdirAll(filepath.Join(directory, "local", "Techs"), 0700)
			case "hardlink":
				if err := os.Link(filepath.Join(directory, "config.json"), filepath.Join(directory, "alias")); err != nil {
					t.Fatal(err)
				}
			case "pending-recovery":
				os.Mkdir(filepath.Join(directory, ".code-rules-authoring-pending"), 0700)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			before, _ := os.ReadFile(filepath.Join(directory, "config.json"))
			result, err := AddLocalGroup(ctx, "techs/go", rules.GroupMetadata{Name: "Go", Description: "Go.", WhenToRead: "When editing Go."}, options)
			if err == nil || result.Files != nil {
				t.Fatal("expected no partial result", result, err)
			}
			after, _ := os.ReadFile(filepath.Join(directory, "config.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("changed config on failure")
			}
			if _, err = os.Lstat(filepath.Join(directory, "local", "techs", "go", "_group.json")); !os.IsNotExist(err) {
				t.Fatal("created group on failure", err)
			}
		})
	}
}

// TestRuleRequiresGroup rejects missing metadata without changing an existing document.
func TestRuleRequiresGroup(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(t.TempDir(), ".code-rules")
	options := Options{Directory: filepath.Dir(directory)}
	if _, err := Initialize(ctx, options); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(directory, "local/techs/go"), 0700)
	target := filepath.Join(directory, "local/techs/go/errors.md")
	os.WriteFile(target, []byte("authored content"), 0600)
	body := "Return failures."
	_, err := AddLocalRule(ctx, "techs/go/errors", rules.RuleMetadata{Title: "Errors", Impact: "HIGH", ImpactDescription: "Failures.", WhenToRead: "When calling."}, RuleOptions{Options: options, Body: &body})
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

// TestRuleUsesImportedGroup accepts imported metadata without creating a local override.
func TestRuleUsesImportedGroup(t *testing.T) {
	ctx := context.Background()
	directory := filepath.Join(t.TempDir(), ".code-rules")
	options := Options{Directory: filepath.Dir(directory)}
	if _, err := Initialize(ctx, options); err != nil {
		t.Fatal(err)
	}
	source := json.RawMessage(`{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":{},"replace":{}}`)
	if _, err := AddSource(ctx, "team", source, options); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanLocalRule(ctx, "techs/go/errors", options); err == nil {
		t.Fatal("planned rule before imported group was fetched")
	}
	data, err := os.ReadFile(filepath.Join(directory, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	config, err := rules.ParseConfiguration(data)
	if err != nil {
		t.Fatal(err)
	}
	vendor, err := encodeSnapshots(config, map[string]snapshot{"team": {Repository: config.Sources[0].Repository, Ref: "v1.0.0", Commit: strings.Repeat("a", 40), Groups: []string{"techs/go"}, Selection: config.Sources[0].Groups, Files: map[string][]byte{"rule-library.json": []byte(`{"formatVersion":1}`), "techs/go/_group.json": []byte(`{"name":"Go","description":"Imported guidance.","whenToRead":"When editing Go."}`)}}})
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
	body := "Return errors."
	result, err := AddLocalRule(ctx, "techs/go/errors", rules.RuleMetadata{Title: "Errors", Impact: "HIGH", ImpactDescription: "Failures.", WhenToRead: "When calling."}, RuleOptions{Options: options, Body: &body})
	if err != nil || len(result.Files) != 1 {
		t.Fatal(result, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "local/techs/go/_group.json")); !os.IsNotExist(err) {
		t.Fatal("rule creation created local override", err)
	}
}
