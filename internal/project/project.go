// Compose the native pipeline into offline build and read-only project checks.

package project

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Options identifies the project root and controls rendering.
type Options struct {
	// Directory is the project root; empty means the process working directory.
	Directory   string
	ToolVersion string
	// IndexMaxLines defaults to 750 when zero. Negative limits are rejected by the renderer.
	IndexMaxLines int
	// GroupInlineMaxBytes uses the renderer's 8 KiB default when nil; zero forces summaries.
	GroupInlineMaxBytes *int
}

// FileChanges lists sorted changed paths. Build uses generated-relative paths; sync and update prefix managed tree
// names, list local/<group>/_group.yaml when they add a local group's metadata, and update also lists config.yaml
// when it writes pins or exclusions.
// Empty lists mean matching tree output; Guide reports a separate managed-guide update.
type FileChanges struct {
	Added   []string     `json:"added"`
	Changed []string     `json:"changed"`
	Removed []string     `json:"removed"`
	Guide   *GuideChange `json:"guide,omitempty"`
	// Warnings explain configuration sync and update tolerated, in source order: entries naming retired rules, and
	// sources importing a ref that isn't a library release. Then, in group order, each local group metadata file
	// they wrote. Build reports none.
	Warnings []string `json:"warnings"`
}

// GuideChange reports a managed-guide update separately from generated-relative file paths.
type GuideChange struct {
	Path    string `json:"path"` // Relative to the configuration directory.
	Created bool   `json:"created"`
}

// Build generates rules offline and refreshes the managed project guide in one recoverable transaction.
// Cancellation, validation failure, or detected edits preserve existing output; errors return no changes.
func Build(ctx context.Context, options Options) (FileChanges, error) {
	if err := ctx.Err(); err != nil {
		return FileChanges{}, err
	}
	root, err := openProject(ctx, options, false)
	if err != nil {
		return FileChanges{}, err
	}
	defer root.Close()
	var changes FileChanges
	err = filetxn.WithWriter(ctx, root, func(w *filetxn.Writer) error {
		before, err := readProject(ctx, root)
		if err != nil {
			return err
		}
		guide, err := planProjectGuide(ctx, root)
		if err != nil {
			return err
		}
		output, err := prepareProject(ctx, before, options)
		if err != nil {
			return err
		}
		changes = compareFiles(treeFiles(before.generated), output.Files)
		targets := map[filetxn.Target]map[string][]byte{filetxn.Generated: output.Files}
		includeProjectGuide(targets, guide, &changes)
		return w.Apply(targets, func() error {
			if err := requireUnchanged(ctx, root, before); err != nil {
				return err
			}
			return requireGuideUnchanged(ctx, root, guide)
		})
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
	root, err := openProject(ctx, options, false)
	if err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	defer root.Close()
	if err := filetxn.RequireIdle(root); err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	before, err := readProject(ctx, root)
	if err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	files, err := readCheckFiles(ctx, root, expected)
	if err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	output, err := prepareProject(ctx, before, options)
	if err != nil {
		return FileChanges{}, FileChanges{}, err
	}
	if err := requireCheckUnchanged(ctx, root, before, expected, files); err != nil {
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
func requireCheckUnchanged(ctx context.Context, root *os.Root, before projectState, expected, files map[string][]byte) error {
	if err := requireUnchanged(ctx, root, before); err != nil {
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

// readProject validates configuration and retains every authored and managed byte used for change detection.
func readProject(ctx context.Context, root *os.Root) (projectState, error) {
	data, config, err := configuration(ctx, root)
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

// prepareProject verifies persisted identity, reloads native library semantics from the verified bytes, resolves,
// and renders in memory.
func prepareProject(ctx context.Context, state projectState, options Options) (build.Output, error) {
	snapshots, err := decodeSnapshots(state.config, treeFiles(state.vendor))
	if err != nil {
		return build.Output{}, err
	}
	libraries := map[string]build.Library{}
	for _, source := range state.config.Sources {
		snapshot := snapshots[source.Name]
		catalog, err := loadSnapshot(ctx, source, snapshot)
		if err != nil {
			return build.Output{}, err
		}
		libraries[source.Name] = build.Library{Catalog: catalog, Snapshot: snapshot}
	}
	return renderProject(ctx, state, libraries, options)
}

// loadSnapshot validates the library content of a verified snapshot, assembling each rule from the library
// release that published it, and checks that it holds exactly the recorded rules and files.
func loadSnapshot(ctx context.Context, source rules.Source, snapshot snapshot) (library.Catalog, error) {
	// An individually selected rule the snapshot doesn't import was retired when it was recorded.
	individual := []string{}
	for _, id := range source.Rules {
		if _, ok := snapshot.Rules[id]; ok {
			individual = append(individual, id)
		}
	}
	catalog, err := library.LoadSource(ctx, snapshot.Source(), source.Name, source.Groups, individual)
	if err != nil {
		return library.Catalog{}, err
	}
	if err := verifyLoadedSnapshot(source, catalog, snapshot); err != nil {
		return library.Catalog{}, err
	}
	return catalog, nil
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
func requireUnchanged(ctx context.Context, root *os.Root, before projectState) error {
	after, err := readProject(ctx, root)
	if err != nil {
		return err
	}
	if !sameProject(before, after) {
		return failure("concurrent-change", "project inputs or managed output changed during the operation; retry after edits finish", nil)
	}
	return nil
}

// sameProject reports whether two reads hold the same configuration bytes and local, vendor, and generated trees.
func sameProject(a, b projectState) bool {
	return bytes.Equal(a.configBytes, b.configBytes) && a.local.Digest() == b.local.Digest() && a.vendor.Digest() == b.vendor.Digest() && a.generated.Digest() == b.generated.Digest()
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
	changes := FileChanges{Added: []string{}, Changed: []string{}, Removed: []string{}, Warnings: []string{}}
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

// verifyLoadedSnapshot requires the loaded catalog to hold exactly the snapshot's recorded groups, rules, and
// files, each stored where the snapshot's rule versions place it, with unchanged bytes.
func verifyLoadedSnapshot(source rules.Source, catalog library.Catalog, snapshot snapshot) error {
	groups := []string{}
	loaded := []string{}
	for _, group := range catalog.Groups {
		if source.Groups.Includes(group.ID) {
			groups = append(groups, group.ID)
		}
		for _, rule := range group.Rules {
			loaded = append(loaded, strings.TrimSuffix(rule.Path, ".md"))
		}
	}
	slices.Sort(loaded)
	if !slices.Equal(loaded, slices.Sorted(maps.Keys(snapshot.Rules))) {
		return failure("invalid-snapshot", source.Name+": the snapshot's rule files differ from the rules its record lists; run code-rules project sync", nil)
	}
	files := catalog.Files()
	if !slices.Equal(groups, snapshot.Groups) || !slices.Equal(slices.Sorted(maps.Keys(files)), slices.Sorted(maps.Keys(snapshot.Files))) {
		return failure("invalid-snapshot", source.Name+": recorded groups or inventory differ from the selected library; run code-rules project sync", nil)
	}
	for _, file := range slices.Sorted(maps.Keys(files)) {
		if !bytes.Equal(files[file], snapshot.Files[file]) {
			return failure("concurrent-change", source.Name+":"+file+": loaded content differs from verified snapshot bytes; retry after edits finish", nil)
		}
	}
	return nil
}
