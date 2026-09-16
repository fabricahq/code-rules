// Validate the caller-supplied tool identity retained in generated provenance.

package rules

import "strings"

// ValidateToolVersion rejects blank text using the established input whitespace rules.
// The value is an opaque build label, so it need not be a semantic version.
func ValidateToolVersion(version string) error {
	if strings.TrimFunc(version, jsWhitespace) == "" {
		return invalid("toolVersion", "expected nonempty text")
	}
	return nil
}
