// Translate the single CLI revision input into the existing exact-ref or version config field.

package cli

import (
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// libraryRef returns the config field and validated value without resolving Git refs.
// Bare versions remain literal tags. refs/tags/ disambiguates tags that resemble constraints.
func libraryRef(input string) (field, value string, err error) {
	value = strings.TrimSpace(input)
	if !strings.HasPrefix(value, "refs/tags/") && value != "" &&
		(strings.ContainsAny(value[:1], "=!<>~^") || strings.Contains(value, ",")) {
		constraint, err := rules.ParseVersionConstraint(value, "--ref")
		if err != nil {
			return "", "", err
		}
		return "version", constraint.String(), nil
	}
	if _, err := rules.ParseGitRef(value, "--ref"); err != nil {
		return "", "", err
	}
	return "ref", value, nil
}
