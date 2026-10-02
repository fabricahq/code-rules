// Recognize Git's merge conflict markers, so a conflicted file is named as such instead of failing to parse.

package rules

import (
	"bytes"
	"strings"

	"github.com/fabricahq/code-rules/internal/decode"
)

// conflictMarkers start the lines Git writes around the two sides of a merge conflict.
var conflictMarkers = []string{"<<<<<<<", "|||||||", "=======", ">>>>>>>"}

// RequireResolvedMerge fails, at location, when data has a line that starts with a Git merge conflict marker, such
// as <<<<<<< HEAD, saying the file has unresolved merge conflicts and what to run once they're resolved.
func RequireResolvedMerge(data []byte, location, next string) error {
	for line := range bytes.Lines(data) {
		text := strings.TrimRight(string(line), "\r\n")
		for _, marker := range conflictMarkers {
			if text == marker || strings.HasPrefix(text, marker+" ") {
				return decode.Invalid(location, "has unresolved merge conflicts, marked by lines such as "+marker+"; resolve them, then "+next)
			}
		}
	}
	return nil
}
