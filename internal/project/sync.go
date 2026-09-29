// Fetch libraries and refresh vendor, generated output, and the managed guide under one recoverable writer.

package project

import (
	"context"
	"fmt"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
)

// Sync imports the rule versions each source's snapshot records, choosing versions only where configuration asks
// for something a snapshot doesn't have, then validates and renders everything before replacing managed trees.
// Git settings are trusted caller options. Any failure returns no change report; authored files stay untouched.
func Sync(ctx context.Context, options Options, git imports.Options) (FileChanges, error) {
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
		recorded, err := recordedSnapshots(before.config, treeFiles(before.vendor))
		if err != nil {
			return err
		}
		imported, err := imports.ImportLibraries(ctx, before.config, recorded, git)
		if err != nil {
			return err
		}
		snapshots := map[string]snapshot{}
		libraries := map[string]build.Library{}
		warnings := []string{}
		for _, source := range before.config.Sources {
			item := imported[source.Name]
			snapshots[source.Name] = item.Snapshot
			libraries[source.Name] = build.Library{Catalog: item.Catalog, Snapshot: item.Snapshot}
			warnings = append(warnings, item.Warnings...)
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
		if len(warnings) > 0 {
			changes.Warnings = warnings
		}
		targets := map[filetxn.Target]map[string][]byte{filetxn.Vendor: vendor, filetxn.Generated: output.Files}
		includeProjectGuide(targets, guide, &changes)
		return w.Apply(targets, func() error {
			if err := requireUnchanged(ctx, root, before); err != nil {
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
