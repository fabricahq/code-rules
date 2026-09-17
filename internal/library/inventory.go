// Apply catalog resource limits to complete captured inventories used by library authoring checks.

package library

import (
	"context"
	"maps"
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
