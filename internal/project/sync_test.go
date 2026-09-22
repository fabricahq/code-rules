// Exercise complete sync against real Git sources and disposable project trees.

package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// syncProject initializes a project and a tagged library with exact binary and license content.
func syncProject(t *testing.T) (*gitfixture.Fixture, Options, imports.Options) {
	t.Helper()
	f, err := gitfixture.New(context.Background(), map[string][]byte{
		"rule-library.json":               []byte(`{"formatVersion":1,"license":{"file":"LICENSE","notices":[]}}`),
		"LICENSE":                         []byte("Original terms\r\n"),
		"techs/go/_group.json":            []byte(projectMetadata),
		"techs/go/errors.md":              []byte(projectRule),
		"techs/go/assets/errors/data.bin": {0, 255, 128},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	root := openTestProject(t)
	if _, err := Initialize(context.Background(), Options{Directory: filepath.Dir(root.Name())}); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"schemaVersion": 1, "sources": map[string]any{"team": map[string]any{"repository": f.Repository, "ref": "v1.0.0", "groups": []string{"techs/go"}, "exclude": map[string]any{}, "replace": map[string]any{}}}})
	writeFixture(t, root, "config.json", string(raw))
	return f, Options{Directory: filepath.Dir(root.Name()), ToolVersion: "1.2.3"}, imports.Options{GitPath: f.GitPath, Environment: f.Environment}
}

// TestSyncRoundTripAndRetirement preserves original bytes, produces clean offline output, and removes retired sources.
func TestSyncRoundTripAndRetirement(t *testing.T) {
	_, options, git := syncProject(t)
	ctx := context.Background()
	first, err := Sync(ctx, options, git)
	if err != nil || len(first.Added) == 0 {
		t.Fatalf("initial sync: %+v %v", first, err)
	}
	root, name, err := projectLocation(options.Directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	state, err := readProject(ctx, root, name)
	if err != nil {
		t.Fatal(err)
	}
	snapshots, err := decodeSnapshots(state.config, state.vendor.Files)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(snapshots["team"].Files["LICENSE"], []byte("Original terms\r\n")) || !bytes.Equal(snapshots["team"].Files["techs/go/assets/errors/data.bin"], []byte{0, 255, 128}) {
		t.Fatal("original snapshot bytes changed")
	}
	before, _ := filetxn.ReadTree(ctx, root, ".")
	again, err := Sync(ctx, options, git)
	if err != nil || len(again.Added)+len(again.Changed)+len(again.Removed) != 0 {
		t.Fatalf("repeat: %+v %v", again, err)
	}
	// This offline check cannot use Git or any language runtime through PATH.
	t.Setenv("PATH", t.TempDir())
	checked, err := Check(ctx, options)
	if err != nil || !checked.Current() {
		t.Fatalf("offline check: %+v %v", checked, err)
	}
	after, _ := filetxn.ReadTree(ctx, root, ".")
	if before.Digest() != after.Digest() {
		t.Fatal("repeat/check changed bytes")
	}
	writeFixture(t, root, name, `{"schemaVersion":1,"sources":{}}`)
	retired, err := Sync(ctx, options, imports.Options{GitPath: "/missing/git"})
	if err != nil || len(retired.Removed) == 0 {
		t.Fatalf("retire: %+v %v", retired, err)
	}
	vendor, err := filetxn.ReadTree(ctx, root, "vendor")
	if err != nil || len(vendor.Files) != 0 {
		t.Fatal("retired files retained", err)
	}
	index, err := root.ReadFile("generated/RULES.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "No active rules are selected for this project.") || strings.Contains(string(index), "Open the relevant group indexes below") {
		t.Fatalf("retired source left misleading reading instructions: %s", index)
	}
}

// TestSyncUpdatesTag refetches a moved exact tag and updates both vendor bytes and generated content.
func TestSyncUpdatesTag(t *testing.T) {
	f, options, git := syncProject(t)
	ctx := context.Background()
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Command(ctx, "tag", "-f", "v1.0.0", f.LatestCommit); err != nil {
		t.Fatal(err)
	}
	changes, err := Sync(ctx, options, git)
	if err != nil || len(changes.Changed) == 0 {
		t.Fatalf("updated sync: %+v %v", changes, err)
	}
	root, name, err := projectLocation(options.Directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	state, err := readProject(ctx, root, name)
	if err != nil {
		t.Fatal(err)
	}
	snapshots, err := decodeSnapshots(state.config, state.vendor.Files)
	if err != nil || snapshots["team"].Commit != f.LatestCommit {
		t.Fatal("stale identity", err)
	}
}

// TestSyncFailurePreservesManagedTrees covers import failures, invalid local rules, and cancellation after a valid sync.
func TestSyncFailurePreservesManagedTrees(t *testing.T) {
	for _, scenario := range []string{"missing-ref", "invalid-local", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			_, options, git := syncProject(t)
			ctx := context.Background()
			if _, err := Sync(ctx, options, git); err != nil {
				t.Fatal(err)
			}
			root, name, err := projectLocation(options.Directory)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if scenario == "missing-ref" {
				raw, err := root.ReadFile(name)
				if err != nil {
					t.Fatal(err)
				}
				raw = bytes.ReplaceAll(raw, []byte("v1.0.0"), []byte("missing"))
				writeFixture(t, root, name, string(raw))
			} else if scenario == "invalid-local" {
				writeFixture(t, root, "local/techs/go/bad.md", "invalid")
			}
			before, err := filetxn.ReadTree(ctx, root, ".")
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			changes, err := Sync(ctx, options, git)
			if err == nil || !reflect.DeepEqual(changes, FileChanges{}) {
				t.Fatal("failure returned result", err)
			}
			if scenario == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal("lost cancellation", err)
			}
			var validation *rules.ValidationError
			if scenario == "invalid-local" && !errors.As(err, &validation) {
				t.Fatal("lost validation error", err)
			}
			after, err := filetxn.ReadTree(context.Background(), root, ".")
			if err != nil || before.Digest() != after.Digest() {
				t.Fatal("failed sync changed project", err)
			}
		})
	}
}
