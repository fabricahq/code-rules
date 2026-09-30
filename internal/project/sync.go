// Fetch libraries and refresh vendor, generated output, and the managed guide under one recoverable writer.

package project

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
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
		changes, err = install(ctx, root, w, before, installation{guide: guide, config: before.config, recorded: recorded, git: git, options: options})
		return err
	})
	if err != nil {
		return FileChanges{}, fmt.Errorf("sync project: %w", err)
	}
	return changes, nil
}

// installation is what install imports and writes. guide is the managed guide as planned under the writer.
// edited is nil unless config is an edited configuration whose bytes install writes to config.yaml.
type installation struct {
	guide    *filetxn.File
	config   rules.Configuration
	edited   []byte
	recorded map[string]library.Snapshot
	git      imports.Options
	options  Options
}

// install imports every source of the installation's configuration from its recorded snapshots, renders the
// project, and replaces vendor and generated output, an older managed guide, and config.yaml when edited, in one
// transaction of w. before is the project as read under w; any change to it before replacement fails with
// concurrent-change. The report lists vendor and generated paths, and config.yaml when edited.
func install(ctx context.Context, root *os.Root, w *filetxn.Writer, before projectState, in installation) (FileChanges, error) {
	imported, err := imports.ImportLibraries(ctx, in.config, in.recorded, in.git)
	if err != nil {
		return FileChanges{}, err
	}
	snapshots := map[string]snapshot{}
	libraries := map[string]build.Library{}
	warnings := []string{}
	for _, source := range in.config.Sources {
		item := imported[source.Name]
		snapshots[source.Name] = item.Snapshot
		libraries[source.Name] = build.Library{Catalog: item.Catalog, Snapshot: item.Snapshot}
		warnings = append(warnings, item.Warnings...)
	}
	vendor, err := encodeSnapshots(in.config, snapshots)
	if err != nil {
		return FileChanges{}, err
	}
	state := before
	state.config = in.config
	output, err := renderProject(ctx, state, libraries, in.options)
	if err != nil {
		return FileChanges{}, err
	}
	changes := compareFiles(managedFiles(treeFiles(before.vendor), treeFiles(before.generated)), managedFiles(vendor, output.Files))
	if len(warnings) > 0 {
		changes.Warnings = warnings
	}
	targets := map[filetxn.Target]map[string][]byte{filetxn.Vendor: vendor, filetxn.Generated: output.Files}
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
