// Decode case-sensitive JSON fields while preserving authored text and diagnostic paths.

package decode

import (
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Object distinguishes malformed JSON from missing, null, or non-object values.
func Object(input json.RawMessage, location string) (map[string]json.RawMessage, error) {
	if len(input) > 0 && !json.Valid(input) {
		return nil, Invalid(location, "invalid JSON")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(input, &fields) != nil || fields == nil {
		return nil, Invalid(location, "expected an object")
	}
	return fields, nil
}

// KnownFields rejects unknown keys in deterministic order before reading values.
func KnownFields(fields map[string]json.RawMessage, allowed []string, location string) error {
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if !slices.Contains(allowed, key) {
			return Invalid(location, "unknown field "+key)
		}
	}
	return nil
}

// Text preserves nonblank text and rejects malformed Unicode rather than replacing it.
func Text(input json.RawMessage, location string) (string, error) {
	var text string
	if json.Unmarshal(input, &text) != nil || strings.TrimFunc(text, IsSpace) == "" {
		return "", Invalid(location, "expected nonempty text")
	}
	if !ValidUnicode(input) {
		return "", Invalid(location, "expected valid Unicode text: invalid UTF-8 or unpaired surrogate escape")
	}
	return text, nil
}

// Path validates a contained relative file name without cleaning or accessing it.
func Path(input json.RawMessage, location string) (string, error) {
	text, err := Text(input, location)
	if err != nil {
		return "", err
	}
	if !Contained(text) {
		return "", Invalid(location, "expected a contained relative path, got "+Quote(text))
	}
	return text, nil
}

// Line returns a one-line text, such as a change summary: nonblank, valid Unicode text on one line, without
// surrounding whitespace or control characters. Projects show summaries in terminals, where control characters,
// such as ESC, could rewrite what they display.
func Line(input json.RawMessage, location string) (string, error) {
	text, err := Text(input, location)
	if err != nil {
		return "", err
	}
	line := strings.TrimFunc(text, IsSpace)
	if strings.ContainsAny(line, "\n\r") {
		return "", Invalid(location, "expected one line")
	}
	if strings.ContainsFunc(line, func(r rune) bool { return r < 0x20 || (r >= 0x7f && r <= 0x9f) }) {
		return "", Invalid(location, "expected text without control characters, such as tabs or escape sequences")
	}
	return line, nil
}

// ValidUnicode checks a JSON value, such as a string, for invalid UTF-8 and unpaired surrogate escapes, which
// encoding/json otherwise replaces with U+FFFD. The input must already be valid JSON, so escape lengths and hex
// digits are well formed.
func ValidUnicode(input json.RawMessage) bool {
	if !utf8.Valid(input) {
		return false
	}
	for i := 0; i < len(input); i++ {
		if input[i] != '\\' {
			continue
		}
		i++
		if input[i] != 'u' {
			continue
		}
		code, err := strconv.ParseUint(string(input[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if code >= 0xdc00 && code <= 0xdfff {
			return false
		}
		if code >= 0xd800 && code <= 0xdbff {
			if i+6 >= len(input) || input[i+1] != '\\' || input[i+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(input[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}
