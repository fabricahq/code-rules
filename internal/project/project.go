// Compose the native pipeline into offline build and read-only project checks.

package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Options locates config and its sibling local/vendor/generated trees and controls rendering.
type Options struct {
	// ConfigPath defaults to .code-rules/config.json, relative to the process working directory.
	ConfigPath  string
	ToolVersion string
	// IndexMaxLines defaults to 750 when zero. Negative limits are rejected by the renderer.
	IndexMaxLines int
	// GroupInlineMaxBytes uses the renderer's 8 KiB default when nil; zero forces summaries.
	GroupInlineMaxBytes *int
}

// FileChanges lists sorted changed paths. Build uses generated-relative paths; Sync prefixes managed tree names.
// Empty lists mean matching output.
type FileChanges struct {
	Added   []string `json:"added"`
	Changed []string `json:"changed"`
	Removed []string `json:"removed"`
}

// Build generates rules entirely offline and replaces only generated/ under exclusive ownership.
// Cancellation, validation failure, or detected edits preserve existing output; errors return no changes.
func Build(ctx context.Context, options Options) (FileChanges, error) {
	if err := ctx.Err(); err != nil {
		return FileChanges{}, err
	}
	root, name, err := projectLocation(options.ConfigPath)
	if err != nil {
		return FileChanges{}, err
	}
	defer root.Close()
	var changes FileChanges
	err = filetxn.WithWriter(ctx, root, func(w *filetxn.Writer) error {
		before, err := readProject(ctx, root, name)
		if err != nil {
			return err
		}
		output, err := prepareProject(ctx, root, before, options)
		if err != nil {
			return err
		}
		changes = compareFiles(treeFiles(before.generated), output.Files)
		return w.Apply(map[filetxn.Target]map[string][]byte{filetxn.Generated: output.Files}, func() error { return requireUnchanged(ctx, root, name, before) })
	})
	if err != nil {
		return FileChanges{}, err
	}
	return changes, nil
}

// checkWithFiles checks generated output and additional root-relative files in one optimistic snapshot.
// Missing files are reported as additions; invalid paths, unreadable files, and concurrent edits are errors.
func checkWithFiles(ctx context.Context, options Options, expected map[string][]byte) (FileChanges, FileChanges, error) {
	if err := ctx.Err(); err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	if err := rules.ValidatePaths(expected, nil); err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	root, name, err := projectLocation(options.ConfigPath)
	if err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	defer root.Close()
	if err := filetxn.RequireIdle(root); err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	before, err := readProject(ctx, root, name)
	if err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	files, err := readCheckFiles(ctx, root, expected)
	if err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	output, err := prepareProject(ctx, root, before, options)
	if err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	if err := requireCheckUnchanged(ctx, root, name, before, expected, files); err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	return compareFiles(treeFiles(before.generated), output.Files), compareFiles(files, expected), nil
}

// readCheckFiles retains only requested files, preserving absence as a missing map entry.
func readCheckFiles(ctx context.Context, root *os.Root, expected map[string][]byte) (map[string][]byte, error) {
	files := map[string][]byte{}
	for _, name := range slices.Sorted(maps.Keys(expected)) {
		data, err := filetxn.ReadFile(ctx, root, name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		files[name] = data
	}
	return files, nil
}

// requireCheckUnchanged verifies both inventories before returning a single freshness decision.
func requireCheckUnchanged(ctx context.Context, root *os.Root, name string, before projectState, expected, files map[string][]byte) error {
	if err := requireUnchanged(ctx, root, name, before); err != nil {
		return err
	}
	after, err := readCheckFiles(ctx, root, expected)
	if err != nil {
		return err
	}
	if !maps.EqualFunc(files, after, bytes.Equal) {
		return failure("concurrent-change", "checked project files changed during the operation; retry after edits finish", nil)
	}
	return filetxn.RequireIdle(root)
}

// projectState holds original bytes and inventories for optimistic change detection before writing.
type projectState struct {
	config                   rules.Configuration
	configBytes              []byte
	local, vendor, generated *filetxn.Tree
}

// projectLocation opens the config parent while leaving the config file itself subject to no-link reads.
func projectLocation(configPath string) (*os.Root, string, error) {
	if configPath == "" {
		configPath = filepath.Join(".code-rules", "config.json")
	}
	absolute, err := filepath.Abs(configPath)
	if err != nil {
		return nil, "", err
	}
	root, err := os.OpenRoot(filepath.Dir(absolute))
	if err != nil {
		return nil, "", fmt.Errorf("open configuration directory: %w", err)
	}
	return root, filepath.Base(absolute), nil
}

// readProject validates configuration and retains every authored and managed byte used for change detection.
func readProject(ctx context.Context, root *os.Root, name string) (projectState, error) {
	data, err := filetxn.ReadFile(ctx, root, name)
	if err != nil {
		return projectState{}, err
	}
	config, err := rules.ParseConfiguration(data)
	if err != nil {
		return projectState{}, err
	}
	state := projectState{config: config, configBytes: data}
	for _, item := range []struct {
		name  string
		value **filetxn.Tree
	}{{"local", &state.local}, {"vendor", &state.vendor}, {"generated", &state.generated}} {
		tree, err := filetxn.ReadTree(ctx, root, item.name)
		if err != nil {
			return projectState{}, err
		}
		*item.value = tree
	}
	return state, nil
}

// prepareProject verifies persisted identity, reloads native library semantics, resolves, and renders in memory.
func prepareProject(ctx context.Context, root *os.Root, state projectState, options Options) (build.Output, error) {
	snapshots, err := decodeSnapshots(state.config, treeFiles(state.vendor))
	if err != nil {
		return build.Output{}, err
	}
	libraries := map[string]build.Library{}
	for _, source := range state.config.Sources {
		snapshot := snapshots[source.Name]
		sourceRoot, err := root.OpenRoot("vendor/" + source.Name)
		if err != nil {
			return build.Output{}, err
		}
		catalog, loadErr := library.Load(ctx, sourceRoot, source.Name, source.Groups)
		closeErr := sourceRoot.Close()
		if loadErr != nil {
			return build.Output{}, loadErr
		}
		if closeErr != nil {
			return build.Output{}, closeErr
		}
		if err := verifyLoadedSnapshot(source.Name, catalog, snapshot); err != nil {
			return build.Output{}, err
		}
		libraries[source.Name] = build.Library{Catalog: catalog, Commit: snapshot.Commit, Tag: snapshot.Tag}
	}
	return renderProject(ctx, state, libraries, options)
}

// renderProject resolves local definitions and renders the same output for offline builds and sync.
func renderProject(ctx context.Context, state projectState, libraries map[string]build.Library, options Options) (build.Output, error) {
	if err := ctx.Err(); err != nil {
		return build.Output{}, err
	}
	version := options.ToolVersion
	if version == "" {
		version = "0.0.0-development"
	}
	output, err := build.Generate(state.config, libraries, treeFiles(state.local), build.Options{ToolVersion: version, IndexMaxLines: options.IndexMaxLines, GroupInlineMaxBytes: options.GroupInlineMaxBytes})
	if err != nil {
		return build.Output{}, err
	}
	if err := ctx.Err(); err != nil {
		return build.Output{}, err
	}
	return output, nil
}

// requireUnchanged compares exact input/output identities immediately before a write or final check result.
func requireUnchanged(ctx context.Context, root *os.Root, name string, before projectState) error {
	after, err := readProject(ctx, root, name)
	if err != nil {
		return err
	}
	if !bytes.Equal(before.configBytes, after.configBytes) || before.local.Digest() != after.local.Digest() || before.vendor.Digest() != after.vendor.Digest() || before.generated.Digest() != after.generated.Digest() {
		return failure("concurrent-change", "project inputs or managed output changed during the operation; retry after edits finish", nil)
	}
	return nil
}

// treeFiles projects original bytes while treating an absent directory as an empty inventory.
func treeFiles(tree *filetxn.Tree) map[string][]byte {
	if tree == nil {
		return nil
	}
	return tree.Files
}

// compareFiles produces deterministic byte-level changes without treating timestamp changes as output changes.
func compareFiles(before, after map[string][]byte) FileChanges {
	changes := FileChanges{Added: []string{}, Changed: []string{}, Removed: []string{}}
	for _, name := range slices.Sorted(maps.Keys(after)) {
		data, ok := before[name]
		if !ok {
			changes.Added = append(changes.Added, name)
		} else if !bytes.Equal(data, after[name]) {
			changes.Changed = append(changes.Changed, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(before)) {
		if _, ok := after[name]; !ok {
			changes.Removed = append(changes.Removed, name)
		}
	}
	return changes
}

// verifyLoadedSnapshot ties every parsed document and supporting byte to the immutable verified snapshot.
// A later filesystem recheck alone cannot detect a transient edit that was reverted after loading.
func verifyLoadedSnapshot(source string, catalog library.Catalog, snapshot snapshot) error {
	groups := make([]string, 0, len(catalog.Groups))
	files := maps.Clone(catalog.SupportingFiles)
	if files == nil {
		files = map[string][]byte{}
	}
	for _, group := range catalog.Groups {
		groups = append(groups, group.ID)
		for _, rule := range group.Rules {
			files[rule.Path] = []byte(rule.Document)
		}
	}
	if !slices.Equal(groups, snapshot.Groups) || !slices.Equal(slices.Sorted(maps.Keys(files)), slices.Sorted(maps.Keys(snapshot.Files))) {
		return failure("invalid-snapshot", source+": recorded groups or inventory differ from the selected library; run sync", nil)
	}
	for _, file := range slices.Sorted(maps.Keys(files)) {
		if !bytes.Equal(files[file], snapshot.Files[file]) {
			return failure("concurrent-change", source+":"+file+": loaded content differs from verified snapshot bytes; retry after edits finish", nil)
		}
	}
	return nil
}
