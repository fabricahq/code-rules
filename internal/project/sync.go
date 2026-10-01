// Fetch libraries and refresh vendor, generated output, and the managed guide under one recoverable writer.

package project

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Sync imports the rule versions each source's snapshot records, choosing versions only where configuration asks
// for something a snapshot doesn't have, then validates and renders everything before replacing managed trees.
// Git settings are trusted caller options. Any failure returns no change report, and, unless a rollback left files
// to recover, an *UnchangedError. Authored files stay untouched,
// except that it adds the metadata of a group whose local rules would otherwise lose it, as install does.
func Sync(ctx context.Context, options Options, git imports.Options) (FileChanges, error) {
	// recovered reports that the writer first recovered an interrupted earlier command, which changed files.
	recovered := false
	if err := ctx.Err(); err != nil {
		return FileChanges{}, unchanged(err, recovered)
	}
	root, err := openProject(ctx, options, false)
	if err != nil {
		return FileChanges{}, unchanged(err, recovered)
	}
	defer root.Close()
	var changes FileChanges
	err = filetxn.WithWriter(ctx, root, func(w *filetxn.Writer) error {
		recovered = w.Recovered()
		before, err := readProject(ctx, root)
		if err != nil {
			return err
		}
		guide, err := planProjectGuide(ctx, root)
		if err != nil {
			return err
		}
		recorded, err := recordedSnapshots(before.config, treeFiles(before.vendor))
		if err != nil {
			return err
		}
		git.GroupMetadata = groupsWithoutLocalMetadata(before)
		imported, err := imports.ImportLibraries(ctx, before.config, recorded, git)
		if err != nil {
			return err
		}
		changes, err = install(ctx, root, w, before, installation{guide: guide, config: before.config, imported: imported, options: options})
		return err
	})
	if err != nil {
		return FileChanges{}, unchanged(err, recovered)
	}
	return changes, nil
}

// installation is what install writes. guide is the managed guide as planned under the writer. edited is nil unless
// config is an edited configuration whose bytes install writes to config.yaml. imported holds every source of config.
type installation struct {
	guide    *filetxn.File
	config   rules.Configuration
	edited   []byte
	imported map[string]imports.Library
	options  Options
	// forks are the forks an update replaces, each with its files read.
	forks []forkUpdate
}

// install renders the project from the installation's imported sources and replaces vendor and generated output,
// an older managed guide, config.yaml when edited, and the local rule and asset directory of each fork it replaces,
// in one transaction of w. When local rules would lose the only imported copy of their group's metadata, the
// transaction also writes that copy to local/<group>/_group.yaml, with a warning. before is the project as read
// under w; any change to it before replacement fails with concurrent-change. The report lists vendor and generated
// paths, config.yaml when edited, local group metadata it adds, and the local files of each replaced fork.
func install(ctx context.Context, root *os.Root, w *filetxn.Writer, before projectState, in installation) (FileChanges, error) {
	snapshots := map[string]snapshot{}
	libraries := map[string]build.Library{}
	warnings := []string{}
	for _, source := range in.config.Sources {
		item := in.imported[source.Name]
		snapshots[source.Name] = item.Snapshot
		libraries[source.Name] = build.Library{Catalog: item.Catalog, Snapshot: item.Snapshot}
		warnings = append(warnings, item.Warnings...)
	}
	vendor, err := encodeSnapshots(in.config, snapshots)
	if err != nil {
		return FileChanges{}, err
	}
	kept, err := keptGroupMetadata(in.config, before, in.imported)
	if err != nil {
		return FileChanges{}, err
	}
	state := before
	state.config = in.config
	if len(kept)+len(in.forks) > 0 {
		local := maps.Clone(treeFiles(before.local))
		if local == nil {
			local = map[string][]byte{}
		}
		for group, data := range kept {
			local[group+"/_group.yaml"] = data
		}
		for _, fork := range in.forks {
			for name := range forkPaths(local, fork.file) {
				delete(local, name)
			}
			maps.Copy(local, fork.files)
		}
		state.local = &filetxn.Tree{Files: local}
	}
	output, err := renderProject(ctx, state, libraries, in.options)
	if err != nil {
		return FileChanges{}, err
	}
	changes := compareFiles(managedFiles(treeFiles(before.vendor), treeFiles(before.generated)), managedFiles(vendor, output.Files))
	targets := map[filetxn.Target]map[string][]byte{filetxn.Vendor: vendor, filetxn.Generated: output.Files}
	for _, group := range slices.Sorted(maps.Keys(kept)) {
		file := path.Join("local", group, "_group.yaml")
		targets[filetxn.LocalGroupMetadata(group)] = map[string][]byte{file: kept[group]}
		changes.Added = append(changes.Added, file)
		warnings = append(warnings, fmt.Sprintf("Wrote %s, the metadata of group %s from the library that last supplied it, because your local rules in the group need it and no imported rule supplies it anymore. It's now yours to edit.", file, group))
	}
	for _, fork := range in.forks {
		forkChanges := compareFiles(localPaths(forkPaths(treeFiles(before.local), fork.file)), localPaths(fork.files))
		changes.Added = append(changes.Added, forkChanges.Added...)
		changes.Changed = append(changes.Changed, forkChanges.Changed...)
		changes.Removed = append(changes.Removed, forkChanges.Removed...)
		assets := rules.RuleAssetDirectory(fork.file)
		var assetFiles map[string][]byte
		for name, data := range fork.files {
			if relative, ok := strings.CutPrefix(name, assets); ok {
				if assetFiles == nil {
					assetFiles = map[string][]byte{}
				}
				assetFiles[relative] = data
			}
		}
		targets[filetxn.LocalRule(fork.file)] = map[string][]byte{path.Join("local", fork.file): fork.files[fork.file]}
		// A nil map removes the old asset directory when the new fork has no assets.
		targets[filetxn.LocalRuleAssets(fork.file)] = assetFiles
	}
	slices.Sort(changes.Added)
	slices.Sort(changes.Changed)
	slices.Sort(changes.Removed)
	changes.Warnings = warnings
	if in.edited != nil {
		targets[filetxn.Config] = map[string][]byte{configurationFile: in.edited}
		changes.Changed = append(changes.Changed, configurationFile)
		slices.Sort(changes.Changed)
	}
	includeProjectGuide(targets, in.guide, &changes)
	err = w.Apply(targets, func() error {
		if err := requireUnchanged(ctx, root, before); err != nil {
			return err
		}
		return requireGuideUnchanged(ctx, root, in.guide)
	})
	if err != nil {
		return FileChanges{}, err
	}
	return changes, nil
}

// forkPaths returns the files of local, by path relative to local/, that belong to the local rule at file: the rule
// and every file in its asset directory.
func forkPaths(local map[string][]byte, file string) map[string][]byte {
	assets := rules.RuleAssetDirectory(file)
	paths := map[string][]byte{}
	for name, data := range local {
		if name == file || strings.HasPrefix(name, assets) {
			paths[name] = data
		}
	}
	return paths
}

// localPaths returns files, which are keyed by path relative to local/, keyed instead by path relative to the Code
// Rules directory.
func localPaths(files map[string][]byte) map[string][]byte {
	result := map[string][]byte{}
	for name, data := range files {
		result[path.Join("local", name)] = data
	}
	return result
}

// groupsWithoutLocalMetadata returns, sorted, the groups of the project's local rules that have no local metadata.
func groupsWithoutLocalMetadata(state projectState) []string {
	local := treeFiles(state.local)
	groups := []string{}
	for file := range local {
		id, versioned := rules.VersionedRule(file)
		if !versioned || file != id+".md" {
			continue
		}
		group := strings.Join(strings.SplitN(id, "/", 3)[:2], "/")
		if _, ok := local[group+"/_group.yaml"]; !ok && !slices.Contains(groups, group) {
			groups = append(groups, group)
		}
	}
	slices.Sort(groups)
	return groups
}

// keptGroupMetadata returns, by group ID, the metadata to write to local/ for each group whose local rules would
// otherwise have none: the project has no local metadata for it, no source now imports it, and a stored source
// record in vendor/ lists the group's metadata, including the record of a source the configuration no longer has.
// Each copy comes from the first such record: configured sources in configuration order, then removed sources in
// name order. A configured source's copy is the metadata in the library release that now supplies its shared
// files, from imported, when that release still has the group; otherwise, and for a removed source, it is the
// vendored copy. Records it can't read are skipped. It is empty, never nil, when no group needs one. It fails when
// a record was changed outside sync, or the vendored copy it would use differs from its record's checksum, so
// modified metadata never becomes local guidance.
func keptGroupMetadata(config rules.Configuration, before projectState, imported map[string]imports.Library) (map[string][]byte, error) {
	supplied := map[string]bool{}
	for _, source := range config.Sources {
		for _, group := range imported[source.Name].Catalog.Groups {
			supplied[group.ID] = true
		}
	}
	vendored := treeFiles(before.vendor)
	records, err := storedRecords(before.config, vendored)
	if err != nil {
		return nil, err
	}
	kept := map[string][]byte{}
	for _, group := range groupsWithoutLocalMetadata(before) {
		if supplied[group] {
			continue
		}
		for _, record := range records {
			recorded, listed := record.digests[group+"/_group.yaml"]
			if !listed {
				continue
			}
			if current, ok := imported[record.name].GroupMetadata[group]; ok {
				kept[group] = current
				break
			}
			file := record.name + "/" + group + "/_group.yaml"
			data, ok := vendored[file]
			if !ok || digest(data) != recorded {
				return nil, invalidSnapshot("vendor/"+file, fmt.Sprintf("missing or modified since sync imported it, so it can't become local/%s/_group.yaml, the metadata your local rules in the group need. Restore it: run code-rules project sync with the previous configuration, then change the configuration and sync again; or write local/%s/_group.yaml yourself", group, group))
			}
			kept[group] = data
			break
		}
	}
	return kept, nil
}

// storedRecord is a valid source record in vendor/, named by its source.
type storedRecord struct {
	name string
	parsedRecord
}

// storedRecords returns every valid source record in vendor: those of config's sources in configuration order,
// then those of sources config no longer has, in name order. Records that don't parse are left out, but a record
// changed outside sync fails, since nothing it lists, such as group metadata to keep, can be trusted.
func storedRecords(config rules.Configuration, vendored map[string][]byte) ([]storedRecord, error) {
	names := []string{}
	for _, source := range config.Sources {
		names = append(names, source.Name)
	}
	removed := []string{}
	for file := range vendored {
		if name, ok := strings.CutSuffix(file, "/_source.json"); ok && !strings.Contains(name, "/") && !slices.Contains(names, name) {
			removed = append(removed, name)
		}
	}
	slices.Sort(removed)
	records := []storedRecord{}
	for _, name := range append(names, removed...) {
		data, ok := vendored[name+"/_source.json"]
		if !ok {
			continue
		}
		record, err := parseSourceRecord(data, name)
		if isChangedOutsideSync(err, name) {
			return nil, err
		}
		if err == nil {
			records = append(records, storedRecord{name: name, parsedRecord: record})
		}
	}
	return records, nil
}

// managedFiles prefixes source-relative and generated-relative paths for an unambiguous combined change report.
func managedFiles(vendor, generated map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(vendor)+len(generated))
	for name, data := range vendor {
		result["vendor/"+name] = data
	}
	for name, data := range generated {
		result["generated/"+name] = data
	}
	return result
}
