// Check proposed authoring paths before prompts without reserving or creating files.

package filetxn

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// RequireAbsent rejects existing targets and unsafe parents without reading file contents.
// This is advisory: publication must still enforce exclusive creation under writer ownership.
func RequireAbsent(ctx context.Context, root *os.Root, names ...string) error {
	for _, name := range names {
		if err := rules.ValidatePaths(map[string][]byte{name: nil}, nil); err != nil {
			return err
		}
		parts := strings.Split(name, "/")
		for i := range parts {
			if err := ctx.Err(); err != nil {
				return err
			}
			prefix := strings.Join(parts[:i+1], "/")
			info, err := root.Lstat(prefix)
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			if err != nil {
				return err
			}
			if i == len(parts)-1 {
				return failure("already-exists", name+": already exists; edit the existing file or choose a different path", nil)
			}
			if !info.IsDir() {
				return failure("unsafe-path", prefix+": expected directories without symlinks", nil)
			}
		}
	}
	return nil
}
