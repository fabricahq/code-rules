// Validate portable filesystem inventories before reading or publishing rule content.

package rules

import (
	"io/fs"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"github.com/fabricahq/code-rules/internal/decode"
)

// ValidatePaths rejects unsafe names, portable spelling collisions, and file/directory conflicts.
// It inspects names only, performs no I/O, and leaves the supplied maps and slices unchanged.
func ValidatePaths(files map[string][]byte, directories []string) error {
	names := maps.Clone(files)
	if names == nil {
		names = map[string][]byte{}
	}
	for _, dir := range directories {
		if _, ok := files[dir]; ok {
			return decode.Invalid(dir, "file also used as directory")
		}
		names[dir] = nil
	}
	spellings := map[string]string{}
	for _, file := range slices.Sorted(maps.Keys(names)) {
		if file == "." || !fs.ValidPath(file) || strings.ContainsAny(file, "\\:\x00") || !utf8.ValidString(file) {
			return decode.Invalid(file, "expected a contained relative file path")
		}
		parts := strings.Split(file, "/")
		for i, part := range parts {
			if strings.ContainsFunc(part, func(r rune) bool { return r < 32 || r == 127 }) {
				return decode.Invalid(file, "control characters are unsupported in paths")
			}
			prefix := strings.Join(parts[:i+1], "/")
			folded := cases.Fold().String(norm.NFC.String(prefix))
			if previous, ok := spellings[folded]; ok && previous != prefix {
				return decode.Invalid(file, "portable path collision with "+previous)
			}
			spellings[folded] = prefix
			if i < len(parts)-1 {
				if _, ok := files[prefix]; ok {
					return decode.Invalid(file, "file also used as directory: "+prefix)
				}
			}
		}
	}
	return nil
}
