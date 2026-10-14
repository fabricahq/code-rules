// Locate and validate the fixed Code Rules directory for every project operation.

package project

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

const configurationFile = "config.yaml"

// openProject applies the same no-link policy to readers and writers; only initialization creates the root. It
// fails, closing the root, when the directory still has the local/ folder that held authored files before rules/
// and assets/.
func openProject(ctx context.Context, options Options, create bool) (*os.Root, error) {
	directory := filepath.Join(options.Directory, ".code-rules")
	var root *os.Root
	var err error
	if create {
		root, err = filetxn.Create(ctx, directory)
	} else {
		root, err = filetxn.Open(ctx, directory)
	}
	if errors.Is(err, os.ErrNotExist) && !create {
		return nil, failure("needs-init", fmt.Sprintf(".code-rules: missing Code Rules directory; run code-rules project init first from %s", options.Directory), err)
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

// localFolderProblem tells a project that still keeps authored files in local/ where each kind now belongs.
const localFolderProblem = `this version of Code Rules reads authored rules and groups from .code-rules/rules/, and shared assets from .code-rules/assets/. It no longer reads .code-rules/local/, and doesn't move its files. Move them yourself, keeping each path below local/, then delete local/:

  local/techs/go/_group.yaml         ->  rules/techs/go/_group.yaml
  local/techs/go/errors.md           ->  rules/techs/go/errors.md
  local/techs/go/assets/errors/      ->  rules/techs/go/assets/errors/
  local/practices/testing/           ->  rules/practices/testing/
  local/README.md                    ->  rules/README.md
  local/assets/retry-lifecycle.svg   ->  assets/retry-lifecycle.svg

Rule IDs don't change, so config.yaml needs no edits. A moved rule sits one folder deeper, so its links to shared assets need one more ../, such as ../../../assets/retry-lifecycle.svg. Then run code-rules project build.`

// requireNoLocalFolder fails, writing nothing, when the Code Rules directory has an entry named local, whatever its
// kind, explaining how to move its files to rules/ and assets/.
func requireNoLocalFolder(root *os.Root) error {
	_, err := root.Lstat("local")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check for .code-rules/local: %v", err)
	}
	return &ValidationError{Location: ".code-rules/local/", Problem: localFolderProblem}
}

// configuration validates the fixed configuration and retains its exact bytes for change detection.
func configuration(ctx context.Context, root *os.Root) ([]byte, rules.Configuration, error) {
	data, err := filetxn.ReadOptional(ctx, root, configurationFile)
	if err != nil {
		return nil, rules.Configuration{}, err
	}
	if data == nil {
		return nil, rules.Configuration{}, failure("needs-init", fmt.Sprintf(".code-rules/config.yaml: missing configuration; run code-rules project init first from %s", filepath.Dir(root.Name())), nil)
	}
	if err := rules.RequireResolvedMerge(data, ".code-rules/config.yaml", "run the command again"); err != nil {
		return nil, rules.Configuration{}, err
	}
	config, err := rules.ParseConfigurationYAML(data)
	return data, config, err
}
