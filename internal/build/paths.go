// Validate generated paths before returning files to an eventual writer.

package build

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
	"io/fs"
	"maps"
	"slices"
	"strings"
)

// validateOutputPaths refuses escaping paths, portable spelling collisions, and file/directory conflicts.
func validateOutputPaths[T any](files map[string]T) error {
	spellings := map[string]string{}
	for _, file := range slices.Sorted(maps.Keys(files)) {
		if !fs.ValidPath(file) || file == "." || strings.ContainsAny(file, "\\:") || strings.ContainsFunc(file, func(r rune) bool { return r < 32 || r == 127 }) {
			return invalid(file, "invalid generated path")
		}
		parts := strings.Split(file, "/")
		for i := range parts {
			prefix := strings.Join(parts[:i+1], "/")
			folded := cases.Fold().String(norm.NFC.String(prefix))
			if previous, ok := spellings[folded]; ok && previous != prefix {
				return invalid(file, "portable path collision with "+previous)
			}
			spellings[folded] = prefix
			if i < len(parts)-1 {
				if _, ok := files[prefix]; ok {
					return invalid(file, "generated path conflicts with file "+prefix)
				}
			}
		}
	}
	return nil
}
