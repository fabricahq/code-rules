// Locate and validate the fixed Code Rules directory for every project operation.

package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

const configurationFile = "config.json"

// openProject applies the same no-link policy to readers and writers; only initialization creates the root.
func openProject(ctx context.Context, options Options, create bool) (*os.Root, error) {
	directory := filepath.Join(options.Directory, ".code-rules")
	if create {
		return filetxn.Create(ctx, directory)
	}
	root, err := filetxn.Open(ctx, directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, failure("needs-init", fmt.Sprintf(".code-rules: missing Code Rules directory; run code-rules project init first from %s", options.Directory), err)
	}
	return root, err
}

// configuration validates the fixed configuration and retains its exact bytes for change detection.
func configuration(ctx context.Context, root *os.Root) ([]byte, rules.Configuration, error) {
	data, err := filetxn.ReadOptional(ctx, root, configurationFile)
	if err != nil {
		return nil, rules.Configuration{}, err
	}
	if data == nil {
		return nil, rules.Configuration{}, failure("needs-init", fmt.Sprintf(".code-rules/config.json: missing configuration; run code-rules project init first from %s", filepath.Dir(root.Name())), nil)
	}
	config, err := rules.ParseConfiguration(data)
	return data, config, err
}
