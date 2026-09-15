// Decode paired UTF-16 escapes in YAML double-quoted scalars without changing authored text.

package rules

import (
	"strconv"
	"strings"
	"unicode/utf16"
)

// yamlSurrogatePairs repairs valid escape pairs in the scalar identified by the YAML scanner.
// The scanner's index counts runes, not bytes. Other scalar styles and escaped backslashes stay literal.
func yamlSurrogatePairs(text string, start int) (string, bool) {
	chars := []rune(text)
	if start < 0 || start >= len(chars) || chars[start] != '"' {
		return text, false
	}
	var result strings.Builder
	result.WriteString(string(chars[:start+1]))
	changed := false
	for i := start + 1; i < len(chars); {
		if chars[i] == '"' {
			result.WriteString(string(chars[i:]))
			return result.String(), changed
		}
		if chars[i] != '\\' {
			result.WriteRune(chars[i])
			i++
			continue
		}
		high, highSize := yamlUnicodeEscape(chars[i:])
		if high >= 0xD800 && high <= 0xDBFF {
			low, lowSize := yamlUnicodeEscape(chars[i+highSize:])
			if low >= 0xDC00 && low <= 0xDFFF {
				result.WriteRune(utf16.DecodeRune(high, low))
				i += highSize + lowSize
				changed = true
				continue
			}
		}
		// Consume the escape as a unit so \\uD83D remains literal text.
		end := min(i+2, len(chars))
		result.WriteString(string(chars[i:end]))
		i = end
	}
	return result.String(), changed
}

// yamlUnicodeEscape reads a four- or eight-digit YAML Unicode escape, returning zero length otherwise.
func yamlUnicodeEscape(chars []rune) (rune, int) {
	if len(chars) < 2 || chars[0] != '\\' {
		return 0, 0
	}
	size := 0
	switch chars[1] {
	case 'u':
		size = 6
	case 'U':
		size = 10
	default:
		return 0, 0
	}
	if len(chars) < size {
		return 0, 0
	}
	value, err := strconv.ParseUint(string(chars[2:size]), 16, 32)
	if err != nil {
		return 0, 0
	}
	return rune(value), size
}
