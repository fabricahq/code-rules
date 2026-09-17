// Apply catalog resource limits to complete captured inventories used by library authoring checks.

package library

import (
	"context"
	"github.com/fabricahq/code-rules/internal/rules"
	"maps"
	"os"
	"path"
	"slices"
)

// ValidateInventoryLimits checks all captured files and directories, including unreferenced assets.
// It uses the same byte and entry limits as catalog loading; it does not validate content or paths.
func ValidateInventoryLimits(ctx context.Context, files map[string][]byte, directories []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(files)+len(directories) > maxFiles {
		return bad("library", "library exceeds filesystem entry limit")
	}
	total := 0
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := ctx.Err(); err != nil {
			return err
		}
		size := len(files[name])
		if size > maxFileBytes || size > maxTotalBytes-total {
			return bad(name, "library exceeds file or total read limits")
		}
		total += size
	}
	return nil
}

// Inventory owns a bounded snapshot of every library-owned file, including unused assets and empty directories.
type Inventory struct {
	Files       map[string][]byte
	Directories []string
	License     *rules.LicenseDeclaration
}

// ReadInventory captures complete library content through the catalog's bounded local reader.
// Unrelated repository files are ignored; the caller owns root and no files are written.
func ReadInventory(ctx context.Context, root *os.Root) (Inventory, error) {
	if root == nil {
		return Inventory{}, bad("library", "expected an open filesystem root")
	}
	return readInventory(ctx, rootFiles{ctx: ctx, root: root})
}

// readInventory shares capture logic with bounded sources so tests can verify early read termination.
func readInventory(ctx context.Context, input FileSource) (Inventory, error) {
	r := reader{ctx: ctx, input: input, files: map[string][]byte{}}
	if _, err := r.read("rule-library.json"); err != nil {
		return Inventory{}, err
	}
	license, err := r.license("library")
	if err != nil {
		return Inventory{}, err
	}
	inventory := Inventory{Files: r.files, Directories: []string{}, License: license}
	// walk visits each owned directory once and enforces limits before traversing later siblings.
	var walk func(string, bool, int) error
	walk = func(directory string, optional bool, depth int) error {
		if depth > 64 {
			return bad(directory, "directory depth exceeds limit")
		}
		entries, err := r.entries(directory, optional)
		if err != nil {
			return err
		}
		if _, exists := r.directories[directory]; !exists {
			return nil
		}
		inventory.Directories = append(inventory.Directories, directory)
		if len(r.files)+len(inventory.Directories) > maxFiles {
			return bad(directory, "library exceeds filesystem entry limit")
		}
		for _, entry := range entries {
			name := path.Join(directory, entry.Name())
			if entry.IsDir() {
				if err := walk(name, false, depth+1); err != nil {
					return err
				}
			} else {
				if _, err := r.read(name); err != nil {
					return err
				}
				if len(r.files)+len(inventory.Directories) > maxFiles {
					return bad(name, "library exceeds filesystem entry limit")
				}
			}
		}
		return nil
	}
	for _, directory := range []string{"assets", "practices", "techs"} {
		if err := walk(directory, true, 1); err != nil {
			return Inventory{}, err
		}
	}
	slices.Sort(inventory.Directories)
	if err := ValidateInventoryLimits(ctx, inventory.Files, inventory.Directories); err != nil {
		return Inventory{}, err
	}
	return inventory, nil
}
