// Parse, compare, and advance rule versions and the change levels that move them.

package coderules

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/fabricahq/code-rules/internal/decode"
)

// RuleVersion is a published rule version: plain major.minor.patch, without prerelease or build suffixes. Versions
// are comparable with ==; use Compare to order them.
type RuleVersion struct {
	Major, Minor, Patch int
}

// Change is how a library release changed a rule, or how a change note says the next one will. A release record's
// changes use new, major, minor, or patch; only change notes use retired, since a record lists retired rules apart.
type Change string

// Changes a change note or release record can give a rule. Major, minor, and patch apply only to a rule that has a
// version.
const (
	ChangeMajor   Change = "major"
	ChangeMinor   Change = "minor"
	ChangePatch   Change = "patch"
	ChangeNew     Change = "new"
	ChangeRetired Change = "retired"
)

// FirstRuleVersion is every new rule's version, including every rule in a library's first library release.
var FirstRuleVersion = RuleVersion{Major: 1}

// maxRuleVersionComponent is the largest major, minor, or patch number a rule version can have.
const maxRuleVersionComponent = 999_999_999

var ruleVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,8})\.(0|[1-9][0-9]{0,8})\.(0|[1-9][0-9]{0,8})$`)

// ParseRuleVersion accepts only canonical major.minor.patch text, such as 1.3.0, with each component at most
// 999,999,999 and no leading zeros. location names the text in errors.
func ParseRuleVersion(text, location string) (RuleVersion, error) {
	parts := ruleVersionPattern.FindStringSubmatch(text)
	if parts == nil {
		return RuleVersion{}, decode.Invalid(location, "invalid rule version "+decode.Quote(text)+": expected major.minor.patch, such as 1.3.0")
	}
	major, _ := strconv.Atoi(parts[1])
	minor, _ := strconv.Atoi(parts[2])
	patch, _ := strconv.Atoi(parts[3])
	return RuleVersion{major, minor, patch}, nil
}

// String returns the canonical major.minor.patch text.
func (v RuleVersion) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
}

// Compare returns -1, 0, or 1 as v is older than, equal to, or newer than other.
func (v RuleVersion) Compare(other RuleVersion) int {
	for _, pair := range [][2]int{{v.Major, other.Major}, {v.Minor, other.Minor}, {v.Patch, other.Patch}} {
		if pair[0] != pair[1] {
			if pair[0] < pair[1] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// Next returns the version after v for a major, minor, or patch change, resetting lower components.
// It returns v unchanged for any other change, and an error when the changed component would exceed
// 999,999,999, so it never returns a version ParseRuleVersion rejects.
func (v RuleVersion) Next(change Change) (RuleVersion, error) {
	next := v
	switch change {
	case ChangeMajor:
		next = RuleVersion{Major: v.Major + 1}
	case ChangeMinor:
		next = RuleVersion{Major: v.Major, Minor: v.Minor + 1}
	case ChangePatch:
		next = RuleVersion{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	}
	if next.Major > maxRuleVersionComponent || next.Minor > maxRuleVersionComponent || next.Patch > maxRuleVersionComponent {
		return v, fmt.Errorf("a %s change from %s exceeds the largest rule version number, %d", change, v, maxRuleVersionComponent)
	}
	return next, nil
}

// MarshalText writes the canonical version text, so JSON and YAML records hold "1.3.0".
func (v RuleVersion) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText accepts only canonical version text.
func (v *RuleVersion) UnmarshalText(text []byte) error {
	parsed, err := ParseRuleVersion(string(text), "version")
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}
