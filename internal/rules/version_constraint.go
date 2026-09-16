// Validate and match version constraints using HashiCorp go-version semantics.

package rules

import (
	"fmt"
	"strings"
	"unicode/utf16"

	version "github.com/hashicorp/go-version"
)

// VersionConstraint holds an authored constraint and its parsed comparisons.
// Construct it with ParseVersionConstraint; the zero value cannot match versions.
type VersionConstraint struct {
	text        string
	comparisons version.Constraints
}

// ParseVersionConstraint validates go-version's comma-separated AND constraints.
// It retains the exact authored text for provenance. Invalid or blank text returns
// a zero VersionConstraint and a ValidationError at the caller's field location.
func ParseVersionConstraint(text, location string) (VersionConstraint, error) {
	if strings.TrimFunc(text, jsWhitespace) == "" {
		return VersionConstraint{}, invalid(location, "expected nonempty text")
	}
	if len(utf16.Encode([]rune(text))) > 1024 {
		return VersionConstraint{}, invalid(location, "version constraint must be at most 1024 characters")
	}
	comparisons, err := version.NewConstraint(text)
	if err != nil {
		return VersionConstraint{}, invalid(location, "expected a version constraint using =, !=, >, >=, <, <=, or ~>; separate multiple constraints with commas, such as >= 1.2.0, < 2.0.0")
	}
	return VersionConstraint{text: text, comparisons: comparisons}, nil
}

// String returns the original constraint, including authored whitespace.
func (c VersionConstraint) String() string { return c.text }

// Matches validates a complete version tag and checks every parsed constraint.
// A valid version outside the constraint returns false, nil. Invalid input or a
// zero constraint returns a ValidationError. Prerelease matching follows go-version.
func (c VersionConstraint) Matches(tag, location string) (bool, error) {
	if len(c.comparisons) == 0 {
		return false, invalid(location, "version constraint must be parsed before matching")
	}
	normalized, err := TagVersion(tag, location)
	if err != nil {
		return false, err
	}
	parsed, err := version.NewVersion(normalized)
	if err != nil {
		// TagVersion already validated syntax and bounds; a dependency failure is unexpected.
		return false, fmt.Errorf("parse validated version at %s: %v", location, err)
	}
	return c.comparisons.Check(parsed), nil
}
