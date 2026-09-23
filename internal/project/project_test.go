// Verify complete offline project behavior against real disposable project trees.

package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

const projectRule = "---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Preserve failures.\nwhenToRead: When calling functions.\n---\n# Return errors\n\nReturn errors to the caller.\n"
const projectMetadata = `{"name":"Go","description":"Go guidance.","whenToRead":"When editing Go."}`

// localProject supplies a project with local-only authored Go guidance.
func localProject(t *testing.T) (*os.Root, Options) {
	t.Helper()
	root := openTestProject(t)
	if _, err := Initialize(context.Background(), Options{Directory: filepath.Dir(root.Name())}); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "config.yaml", `{"schemaVersion":1,"sources":{}}`)
	writeFixture(t, root, "local/techs/go/_group.yaml", projectMetadata)
	writeFixture(t, root, "local/techs/go/errors.md", projectRule)
	return root, Options{Directory: filepath.Dir(root.Name()), ToolVersion: "1.2.3"}
}

// importedProject seeds byte-verified native snapshots without invoking Git or a network process.
func importedProject(t *testing.T) (*os.Root, Options) {
	t.Helper()
	root, options := localProject(t)
	if err := root.RemoveAll("local"); err != nil {
		t.Fatal(err)
	}
	configJSON := []byte(`{"schemaVersion":1,"sources":{"team":{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":{},"replace":{}}}}`)
	writeFixture(t, root, "config.yaml", string(configJSON))
	config, err := rules.ParseConfiguration(configJSON)
	if err != nil {
		t.Fatal(err)
	}
	imported := snapshot{Repository: config.Sources[0].Repository, Ref: "v1.0.0", Commit: strings.Repeat("a", 40), Groups: []string{"techs/go"}, Selection: config.Sources[0].Groups, Files: map[string][]byte{"rule-library.yaml": []byte(`{"formatVersion":1}`), "techs/go/_group.yaml": []byte(projectMetadata), "techs/go/errors.md": []byte(projectRule)}}
	vendor, err := encodeSnapshots(config, map[string]snapshot{"team": imported})
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range vendor {
		writeFixture(t, root, filepath.Join("vendor", name), string(data))
	}
	return root, options
}

// TestOfflineBuildCheckAndRepeat covers generated changes and no writes from both stale and clean checks.
func TestOfflineBuildCheckAndRepeat(t *testing.T) {
	for _, kind := range []string{"local", "imported"} {
		t.Run(kind, func(t *testing.T) {
			var root *os.Root
			var options Options
			if kind == "local" {
				root, options = localProject(t)
			} else {
				root, options = importedProject(t)
			}
			// An empty PATH proves no runtime dependency on Git, Node, Bun, or shell utilities.
			t.Setenv("PATH", t.TempDir())
			before, err := filetxn.ReadTree(context.Background(), root, ".")
			if err != nil {
				t.Fatal(err)
			}
			stale, err := Check(context.Background(), options)
			if err != nil || stale.Current() {
				t.Fatalf("stale check: %+v %v", stale, err)
			}
			after, err := filetxn.ReadTree(context.Background(), root, ".")
			if err != nil || before.Digest() != after.Digest() {
				t.Fatal("check wrote files")
			}
			changes, err := Build(context.Background(), options)
			if err != nil {
				t.Fatal(err)
			}
			if len(changes.Added) != len(stale.Problems) {
				t.Fatalf("build/check differ: %+v %+v", changes, stale)
			}
			for i, problem := range stale.Problems {
				if problem.Kind != MissingFile || problem.Path != "generated/"+changes.Added[i] || problem.Repair != Rebuild {
					t.Fatalf("unexpected initial problem: %+v", problem)
				}
			}
			tree, err := filetxn.ReadTree(context.Background(), root, "generated")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(tree.Files["groups/techs/go.md"]), "Return errors to the caller.") {
				t.Fatal("short group not delivered in full")
			}
			again, err := Build(context.Background(), options)
			if err != nil || len(again.Added)+len(again.Changed)+len(again.Removed) != 0 {
				t.Fatalf("not idempotent: %+v %v", again, err)
			}
			clean, err := Check(context.Background(), options)
			if err != nil || !clean.Current() {
				t.Fatalf("clean check: %+v %v", clean, err)
			}
			writeFixture(t, root, "generated/RULES.md", "stale")
			writeFixture(t, root, "generated/extra.md", "extra")
			if err := root.Remove("generated/groups/techs/go.md"); err != nil {
				t.Fatal(err)
			}
			before, err = filetxn.ReadTree(context.Background(), root, ".")
			if err != nil {
				t.Fatal(err)
			}
			differences, err := Check(context.Background(), options)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(differences.Problems, []Problem{{MissingFile, "generated/groups/techs/go.md", Rebuild}, {StaleContents, "generated/RULES.md", Rebuild}, {UnexpectedFile, "generated/extra.md", Rebuild}}) {
				t.Fatalf("diffs: %+v", differences)
			}
			after, err = filetxn.ReadTree(context.Background(), root, ".")
			if err != nil || before.Digest() != after.Digest() {
				t.Fatal("stale check changed tree")
			}
		})
	}
}

// TestOfflineFailuresPreserveOutput checks invalid local/vendor content, pending recovery, and cancellation.
func TestOfflineFailuresPreserveOutput(t *testing.T) {
	for _, kind := range []string{"local", "vendor", "recovery", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			root, options := importedProject(t)
			writeFixture(t, root, "generated/keep", "existing output")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "local":
				writeFixture(t, root, "local/techs/go/bad.md", "invalid rule")
			case "vendor":
				writeFixture(t, root, "vendor/team/techs/go/errors.md", "modified imported rule")
			case "recovery":
				writeFixture(t, root, ".code-rules-transaction"+"/journal.json", `{"formatVersion":99}`)
			case "cancel":
				cancel()
			}
			before, err := filetxn.ReadTree(context.Background(), root, ".")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Check(ctx, options); err == nil {
				t.Fatal("check accepted invalid project")
			}
			after, err := filetxn.ReadTree(context.Background(), root, ".")
			if err != nil || before.Digest() != after.Digest() {
				t.Fatal("failed check wrote files")
			}
			if _, err := Build(ctx, options); err == nil {
				t.Fatal("build accepted invalid project")
			} else if kind == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
			if b, _ := root.ReadFile("generated/keep"); string(b) != "existing output" {
				t.Fatal("lost existing output")
			}
		})
	}
}

// TestOfflineRejectsSemanticCorruptionBeyondDigests validates catalog content even with self-consistent hashes.
func TestOfflineRejectsSemanticCorruptionBeyondDigests(t *testing.T) {
	root, options := importedProject(t)
	writeFixture(t, root, "vendor/team/techs/go/errors.md", "invalid but rehashed")
	data, err := root.ReadFile("vendor/team/_source.json")
	if err != nil {
		t.Fatal(err)
	}
	var record sourceRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	record.Files["techs/go/errors.md"] = digest([]byte("invalid but rehashed"))
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "vendor/team/_source.json", string(encoded))
	if _, err := Check(context.Background(), options); err == nil {
		t.Fatal("accepted invalid content with valid digest")
	}
}

// TestProjectInputRecheckRejectsLaterEdits verifies the same guard Build supplies to Apply.
func TestProjectInputRecheckRejectsLaterEdits(t *testing.T) {
	root, _ := localProject(t)
	state, err := readProject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "local/techs/go/errors.md", projectRule+"Later edit.\n")
	projectCode(t, requireUnchanged(context.Background(), root, state), "concurrent-change")
}

// TestOfflineRejectsContentChangedBetweenVerificationAndLoading binds rendered bytes to the verified snapshot.
func TestOfflineRejectsContentChangedBetweenVerificationAndLoading(t *testing.T) {
	for _, file := range []string{"techs/go/errors.md", "techs/go/_group.yaml"} {
		t.Run(file, func(t *testing.T) {
			root, options := importedProject(t)
			state, err := readProject(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			changed := strings.ReplaceAll(projectRule, "Return errors to the caller.", "Unverified body.")
			if strings.HasSuffix(file, ".yaml") {
				changed = strings.ReplaceAll(projectMetadata, "Go guidance.", "Unverified description.")
			}
			writeFixture(t, root, "vendor/team/"+file, changed)
			_, err = prepareProject(context.Background(), root, state, options)
			projectCode(t, err, "concurrent-change")
		})
	}
}
