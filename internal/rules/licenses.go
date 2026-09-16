// Validate library manifest declarations against an in-memory inventory, preserving term files.

package rules

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// LicenseDeclaration records authored terms and paths, without interpreting legal meaning.
// A nil SPDXExpression means the library declared a file but no SPDX identifier.
type LicenseDeclaration struct {
	SPDXExpression   *string  `json:"spdxExpression"`
	Files            []string `json:"files"`
	AttributionFiles []string `json:"attributionFiles"`
}

// ReadLibraryLicense validates rule-library.json and declared file presence.
// It never decodes, copies, or mutates license/notice bytes; empty and binary files
// count as present. Missing licensing returns nil, not an inferred license.
// Unknown manifest fields retain baseline compatibility; license fields are strict.
func ReadLibraryLicense(files map[string][]byte, source string) (*LicenseDeclaration, error) {
	location := source + "/rule-library.json"
	text, ok := files["rule-library.json"]
	if !ok {
		return nil, invalid(location, "missing required file")
	}
	if !utf8.Valid(text) {
		return nil, invalid(location, "expected UTF-8 text")
	}
	manifest, err := jsonObject(text, location)
	if err != nil {
		return nil, err
	}
	var format float64
	if json.Unmarshal(manifest["formatVersion"], &format) != nil || format != 1 {
		return nil, invalid(location+": formatVersion", "only library formatVersion 1 is supported")
	}
	raw, ok := manifest["license"]
	if !ok {
		return nil, nil
	}
	fields, err := jsonObject(raw, location+": license")
	if err != nil {
		return nil, err
	}
	location += ": license"
	if err := knownJSONFields(fields, []string{"file", "notices", "spdxExpression", "expression"}, location); err != nil {
		return nil, err
	}
	file, err := jsonPath(fields["file"], location+".file")
	if err != nil {
		return nil, err
	}
	var rawNotices []json.RawMessage
	if json.Unmarshal(fields["notices"], &rawNotices) != nil || rawNotices == nil {
		return nil, invalid(location+".notices", "expected an array of notice paths")
	}
	notices := make([]string, len(rawNotices))
	for i, raw := range rawNotices {
		path, err := jsonPath(raw, fmt.Sprintf("%s.notices[%d]", location, i))
		if err != nil {
			return nil, err
		}
		notices[i] = path
	}
	if _, ok := files[file]; !ok {
		return nil, invalid(location+".file", "missing declared file "+quote(file))
	}
	for i, path := range notices {
		if _, ok := files[path]; !ok {
			return nil, invalid(fmt.Sprintf("%s.notices[%d]", location, i), "missing declared file "+quote(path))
		}
	}
	if _, ok := fields["expression"]; ok {
		return nil, invalid(location+".expression", "renamed to license.spdxExpression; move the declaration to that field")
	}
	var expression *string
	if raw, ok := fields["spdxExpression"]; ok {
		text, err := jsonText(raw, location+".spdxExpression")
		if err != nil {
			return nil, err
		}
		if strings.ContainsFunc(text, controlCharacter) {
			return nil, invalid(location+".spdxExpression", "expected a single-line license expression")
		}
		if !validSPDXExpression(text) {
			return nil, invalid(location+".spdxExpression", "expected an SPDX expression using recognized identifiers or LicenseRef- custom terms")
		}
		expression = &text
	}
	result := LicenseDeclaration{SPDXExpression: expression, Files: []string{file}, AttributionFiles: []string{}}
	seen := map[string]bool{file: true}
	for _, path := range notices {
		if !seen[path] {
			result.AttributionFiles = append(result.AttributionFiles, path)
			seen[path] = true
		}
	}
	return &result, nil
}

// controlCharacter identifies bytes that cannot occur in a single-line SPDX expression.
func controlCharacter(r rune) bool { return r < 32 || r == 127 }

// LicensePaths returns unique retained paths in UTF-16 code-unit order without modifying the declaration.
func LicensePaths(license *LicenseDeclaration) []string {
	paths := []string{}
	seen := make(map[string]bool)
	if license != nil {
		for _, list := range [][]string{license.Files, license.AttributionFiles} {
			for _, path := range list {
				if !seen[path] {
					paths = append(paths, path)
					seen[path] = true
				}
			}
		}
	}
	slices.SortFunc(paths, compareText)
	return paths
}

// compareText matches the pinned reference's deterministic UTF-16 ordering for Unicode paths.
func compareText(a, b string) int {
	return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(b)))
}
