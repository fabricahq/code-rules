// Fetch libraries and refresh vendor, generated output, and the managed guide under one recoverable writer.

package project

import (
	"context"
	"fmt"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
)

// Sync re-resolves configured revisions, validates and renders everything before replacing managed trees.
// Git settings are trusted caller options. Any failure returns no change report; authored files stay untouched.
func Sync(ctx context.Context, options Options, git imports.Options) (FileChanges, error) {
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
		guide, err := planProjectGuide(ctx, root, name)
		if err != nil {
			return err
		}
		imported, err := imports.ImportLibraries(ctx, before.config, git)
		if err != nil {
			return err
		}
		snapshots := map[string]snapshot{}
		libraries := map[string]build.Library{}
		for alias, item := range imported {
			snapshots[alias] = item.Snapshot
			libraries[alias] = build.Library{Catalog: item.Catalog, Commit: item.Snapshot.Commit, Tag: item.Snapshot.Tag}
		}
		vendor, err := encodeSnapshots(before.config, snapshots)
		if err != nil {
			return err
		}
		output, err := renderProject(ctx, before, libraries, options)
		if err != nil {
			return err
		}
		changes = compareFiles(managedFiles(treeFiles(before.vendor), treeFiles(before.generated)), managedFiles(vendor, output.Files))
		targets := map[filetxn.Target]map[string][]byte{filetxn.Vendor: vendor, filetxn.Generated: output.Files}
		includeProjectGuide(targets, guide, &changes)
		return w.Apply(targets, func() error {
			if err := requireUnchanged(ctx, root, name, before); err != nil {
				return err
			}
			return requireGuideUnchanged(ctx, root, guide)
		})
	})
	if err != nil {
		return FileChanges{}, fmt.Errorf("sync project: %w", err)
	}
	return changes, nil
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
