// Package rules validates group and rule identities without accessing the filesystem.
package rules

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	groupPattern    = regexp.MustCompile(`^(techs|practices)/[a-z][a-z0-9-]*$`)
	rulePartPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)
)

// ValidationError identifies invalid user input. Location names the input field,
// including its original array index when applicable.
type ValidationError struct {
	Location string
	Problem  string
}

func (e *ValidationError) Error() string { return e.Location + ": " + e.Problem }

// ValidateGroupID accepts a technology or practice group ID, except the reserved
// assets name. It checks syntax only; it does not establish that the group exists.
func ValidateGroupID(value, location string) error {
	if !groupPattern.MatchString(value) {
		return invalid(location, "invalid group ID "+quote(value)+": expected format "+groupPattern.String())
	}
	if strings.HasSuffix(value, "/assets") {
		return invalid(location, "invalid group ID "+quote(value)+`: "assets" is a reserved group name`)
	}
	return nil
}

// GroupFromPath returns the owning group of a contained Markdown rule path.
// Paths use forward slashes on every OS and are never cleaned or normalized.
// Containment errors take precedence over rule syntax, then group syntax errors.
func GroupFromPath(path, location string) (string, error) {
	parts := strings.Split(path, "/")
	if !contained(path, parts) {
		return "", invalid(location, "expected a contained relative path, got "+quote(path)+`: use forward slashes; no leading slash, drive prefix, empty segments, "." or ".." segments, or control characters`)
	}
	if len(parts) < 3 || !strings.HasSuffix(path, ".md") {
		return "", invalid(location, "invalid rule path "+quote(path)+": expected a .md file beneath a group directory")
	}
	for i, part := range parts {
		if part == "assets" {
			return "", invalid(location, "invalid rule path "+quote(path)+`: "assets" is reserved for supporting files, not rules`)
		}
		if i >= 2 && !rulePartPattern.MatchString(part) {
			return "", invalid(location, "invalid rule path "+quote(path)+": rule file and subdirectory names must match "+rulePartPattern.String())
		}
	}
	group := strings.Join(parts[:2], "/")
	if err := ValidateGroupID(group, location); err != nil {
		return "", err
	}
	return group, nil
}

func contained(path string, parts []string) bool {
	if len(path) >= 2 && path[1] == ':' && ((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) {
		return false
	}
	for _, r := range path {
		if r == '\\' || r < 32 || r == 127 {
			return false
		}
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func invalid(location, problem string) error {
	return &ValidationError{Location: location, Problem: problem}
}

// quote preserves the reference's JSON string spelling for Unicode scalar text.
// strconv.Quote uses Go-only escapes; encoding/json also escapes HTML and U+2028.
func quote(value string) string {
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
