// Open authoring roots without accepting final-directory symlinks.

package filetxn

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Open opens an existing directory, rejecting a final symlink. The caller closes the returned root.
func Open(ctx context.Context, directory string) (*os.Root, error) {
	return open(ctx, directory, false)
}

// Create opens a directory after safely creating missing parents; existing content remains unchanged.
func Create(ctx context.Context, directory string) (*os.Root, error) {
	return open(ctx, directory, true)
}

func open(ctx context.Context, directory string, create bool) (*os.Root, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if create {
		if err := createDirectory(absolute); err != nil {
			return nil, err
		}
	}
	info, err := os.Lstat(absolute)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, failure("missing-root", absolute+": directory does not exist", err)
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, failure("unsafe-path", absolute+": expected a directory without symlinks", nil)
	}
	return os.OpenRoot(absolute)
}

// createDirectory resolves existing parent aliases before enforcing the final directory's own no-link boundary.
func createDirectory(name string) error {
	parent := filepath.Dir(name)
	if parent == name {
		return nil
	}
	if _, err := os.Stat(parent); errors.Is(err, fs.ErrNotExist) {
		if err = createDirectory(parent); err != nil {
			return err
		}
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer root.Close()
	created := []string{}
	return ensureParents(root, filepath.Base(name), &created)
}
