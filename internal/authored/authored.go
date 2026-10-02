// Package authored decodes the YAML and JSON that people write for Code Rules, such as rule metadata, release
// records, and project configuration, with one strict policy, and reports invalid input as a *ValidationError at
// the location the caller names. It accesses no files; the coderules package and Code Rules' internal parsers
// share it, so every format applies the same rules.
package authored

import (
	"fmt"
	"strings"

	"github.com/fabricahq/code-rules/internal/errs"
)

// ValidationError reports invalid input at a location, such as a configuration key or a flag.
type ValidationError struct {
	// Location is a caller-supplied field path, file, or flag, including the original array index when
	// applicable. It is diagnostic text, never a path to access.
	Location string
	// Problem explains the failure to a human. Callers must not branch on its text.
	Problem string
}

var _ errs.ValidationError = (*ValidationError)(nil)

// Error returns the location and the problem.
func (e *ValidationError) Error() string { return e.Location + ": " + e.Problem }

// ValidationLocation returns Location.
func (e *ValidationError) ValidationLocation() string { return e.Location }

// ValidationProblem returns Problem.
func (e *ValidationError) ValidationProblem() string { return e.Problem }

// Invalid returns a *ValidationError reporting problem at location.
func Invalid(location, problem string) error {
	return &ValidationError{Location: location, Problem: problem}
}

// Quote preserves the reference's JSON string spelling for Unicode scalar text.
// strconv.Quote uses Go-only escapes; encoding/json also escapes HTML and U+2028.
func Quote(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 32 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// IsSpace reports whether authored text trims r from its ends: JavaScript's trim set, which includes BOM but
// excludes NEL, unlike unicode.IsSpace.
func IsSpace(r rune) bool {
	return r == 0x0009 || r == 0x000a || r == 0x000b || r == 0x000c || r == 0x000d ||
		r == 0x0020 || r == 0x00a0 || r == 0x1680 || (r >= 0x2000 && r <= 0x200a) ||
		r == 0x2028 || r == 0x2029 || r == 0x202f || r == 0x205f || r == 0x3000 || r == 0xfeff
}

// Contained reports whether path is a relative path that stays inside its root without cleaning: no drive prefix,
// backslash, control character, or empty, "." or ".." segment.
func Contained(path string) bool {
	if len(path) >= 2 && path[1] == ':' && ((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) {
		return false
	}
	for _, r := range path {
		if r == '\\' || r < 32 || r == 127 {
			return false
		}
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}
