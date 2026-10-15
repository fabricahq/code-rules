// Locate and validate the fixed Syllabus directory for every project operation.

package project

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/librarytree"
	"github.com/fabricahq/code-rules/internal/rules"
)

const configurationFile = "config.yaml"

// openProject applies the same no-link policy to readers and writers; only initialization creates the root. It opens
// the directory Directory names. It fails, closing the root, when the directory still has the local/ folder that
// held authored files before rules/ and assets/.
func openProject(ctx context.Context, options Options, create bool) (*os.Root, error) {
	directory := Directory(options.Directory)
	var root *os.Root
	var err error
	if create {
		root, err = filetxn.Create(ctx, directory)
	} else {
		root, err = filetxn.Open(ctx, directory)
	}
	if errors.Is(err, os.ErrNotExist) && !create {
		return nil, failure("needs-init", fmt.Sprintf(".syllabus: missing Syllabus directory; run syllabus project init first from %s", options.Directory), err)
	}
	if err != nil {
		return nil, err
	}
	if err := requireNoLocalFolder(root); err != nil {
		root.Close()
		return nil, err
	}
	return root, nil
}

// Directory returns the path of the Syllabus directory of the project at root: .syllabus/, or .code-rules/ when
// UsesLegacyDirectory reports the project still keeps its files there.
func Directory(root string) string {
	if UsesLegacyDirectory(root) {
		return filepath.Join(root, librarytree.LegacyDirectory)
	}
	return filepath.Join(root, librarytree.Directory)
}

// UsesLegacyDirectory reports whether the project at root has no Syllabus directory but has a directory under the
// name Code Rules used, .code-rules/, which project commands then read and update in its place.
func UsesLegacyDirectory(root string) bool {
	if _, err := os.Lstat(filepath.Join(root, librarytree.Directory)); !errors.Is(err, fs.ErrNotExist) {
		return false
	}
	info, err := os.Lstat(filepath.Join(root, librarytree.LegacyDirectory))
	return err == nil && info.IsDir()
}

// localFolderProblem tells a project that still keeps authored files in local/ where each kind now belongs.
const localFolderProblem = `this version of Syllabus reads authored rules and groups from .syllabus/rules/, and shared assets from .syllabus/assets/. It no longer reads .syllabus/local/, and doesn't move its files. Move them yourself, keeping each path below local/, then delete local/:

  local/techs/go/_group.yaml         ->  rules/techs/go/_group.yaml
  local/techs/go/errors.md           ->  rules/techs/go/errors.md
  local/techs/go/assets/errors/      ->  rules/techs/go/assets/errors/
  local/practices/testing/           ->  rules/practices/testing/
  local/README.md                    ->  rules/README.md
  local/assets/retry-lifecycle.svg   ->  assets/retry-lifecycle.svg

Rule IDs don't change, so config.yaml needs no edits. A moved rule sits one folder deeper, so its links to shared assets need one more ../, such as ../../../assets/retry-lifecycle.svg. Then run syllabus project build.`

// requireNoLocalFolder fails, writing nothing, when the Syllabus directory has an entry named local, whatever its
// kind, explaining how to move its files to rules/ and assets/.
func requireNoLocalFolder(root *os.Root) error {
	_, err := root.Lstat("local")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check for .syllabus/local: %v", err)
	}
	return &ValidationError{Location: ".syllabus/local/", Problem: localFolderProblem}
}

// configuration validates the fixed configuration and retains its exact bytes for change detection.
func configuration(ctx context.Context, root *os.Root) ([]byte, rules.Configuration, error) {
	data, err := filetxn.ReadOptional(ctx, root, configurationFile)
	if err != nil {
		return nil, rules.Configuration{}, err
	}
	if data == nil {
		return nil, rules.Configuration{}, failure("needs-init", fmt.Sprintf(".syllabus/config.yaml: missing configuration; run syllabus project init first from %s", filepath.Dir(root.Name())), nil)
	}
	if err := rules.RequireResolvedMerge(data, ".syllabus/config.yaml", "run the command again"); err != nil {
		return nil, rules.Configuration{}, err
	}
	config, err := rules.ParseConfigurationYAML(data)
	return data, config, err
}
