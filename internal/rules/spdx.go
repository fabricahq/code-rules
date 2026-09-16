// Adapt SPDX expression validation while retaining authored spelling and baseline case rules.

package rules

import (
	"github.com/github/go-spdx/v2/spdxexp"
	"github.com/github/go-spdx/v2/spdxexp/spdxlicenses"
	"regexp"
	"strings"
)

var spdxWord = regexp.MustCompile(`[A-Za-z0-9.-]+`)

// validSPDXExpression delegates grammar to go-spdx, preserving case-sensitive IDs
// and case-insensitive operators from the reference. It never rewrites the output.
func validSPDXExpression(text string) bool {
	valid := true
	// Normalize only operator tokens for the dependency; identifiers keep their spelling.
	normalized := spdxWord.ReplaceAllStringFunc(text, func(word string) string {
		upper := strings.ToUpper(word)
		if upper == "AND" || upper == "OR" || upper == "WITH" {
			return upper
		}
		if strings.HasPrefix(word, "LicenseRef-") || strings.HasPrefix(word, "DocumentRef-") {
			return word
		}
		active, canonical := spdxlicenses.IsActiveLicense(word)
		if active && canonical == word {
			return word
		}
		deprecated, canonical := spdxlicenses.IsDeprecatedLicense(word)
		if deprecated && canonical == word {
			return word
		}
		exception, canonical := spdxlicenses.IsException(word)
		if exception && canonical == word {
			return word
		}
		valid = false
		return word
	})
	if !valid {
		return false
	}
	// ExtractLicenses parses without the validator's whitespace/case normalization shortcuts.
	_, err := spdxexp.ExtractLicenses(normalized)
	return err == nil
}
