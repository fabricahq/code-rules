// Validate the library section of a configuration, the manifest, and the license it declares, returning safe
// library paths for callers to read.

package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/decode"
	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/librarytree"
)

// Library is a configuration's library section, the manifest, which makes the repository a library.
type Library struct {
	// License is nil when the library declares no license.
	License *LicenseDeclaration `json:"license"`
}

// LicenseDeclaration records authored terms and paths, without interpreting legal meaning.
// A nil SPDXExpression means the library declared a file but no SPDX identifier. Files and AttributionFiles hold
// library paths, which are the declared paths except inside .code-rules/: a declared
// .code-rules/assets/NOTICE.md is assets/NOTICE.md.
type LicenseDeclaration struct {
	SPDXExpression   *string  `json:"spdxExpression"`
	Files            []string `json:"files"`
	AttributionFiles []string `json:"attributionFiles"`
}

// skillLevelsProblem rejects library.skills, which a later version of Code Rules reads.
const skillLevelsProblem = "skill levels aren't supported yet in this version of Code Rules; remove library.skills"

// ParseLibraryManifest validates text, a library's whole .code-rules/config.yaml, and returns its library section,
// or nil when the configuration has none, which means the repository isn't a library. Errors prefix their location
// with location, such as team/config.yaml. It checks syntax only; callers read the declared files themselves.
func ParseLibraryManifest(text []byte, location string) (*Library, error) {
	if !utf8.Valid(text) {
		return nil, invalid(location, "expected UTF-8 text")
	}
	_, data, err := decode.YAML(text, location)
	if err != nil {
		return nil, err
	}
	config, err := ParseConfiguration(data)
	var validation errs.ValidationError
	if errors.As(err, &validation) {
		return nil, invalid(location+": "+validation.ValidationLocation(), validation.ValidationProblem())
	}
	if err != nil {
		return nil, err
	}
	return config.Library, nil
}

// PublishedManifest returns what a library publishes of text, its whole .code-rules/config.yaml, when it validates
// as ParseLibraryManifest validates it: a configuration that holds only its library section, as canonical JSON, so
// the sources and other fields that configure the repository as a project, and its comments and formatting, stay in
// the repository. It returns nil for a configuration without a library section.
func PublishedManifest(text []byte, location string) ([]byte, error) {
	declaration, err := ParseLibraryManifest(text, location)
	if err != nil || declaration == nil {
		return nil, err
	}
	_, data, err := decode.YAML(text, location)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	return PublishedManifestOf(fields["library"])
}

// PublishedManifestOf returns the configuration that PublishedManifest publishes for a library section, given as
// JSON: schema version 2, no sources, and the section with its object keys sorted.
func PublishedManifestOf(section json.RawMessage) ([]byte, error) {
	var value any
	if err := json.Unmarshal(section, &value); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(map[string]any{"schemaVersion": 2, "sources": map[string]any{}, "library": value})
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// parseLibrary validates a configuration's library section. Its only field is license; skill levels are rejected
// until Code Rules supports them.
func parseLibrary(input json.RawMessage) (*Library, error) {
	fields, err := decode.Object(input, "library")
	if err != nil {
		return nil, err
	}
	if _, ok := fields["skills"]; ok {
		return nil, invalid("library.skills", skillLevelsProblem)
	}
	if err := decode.KnownFields(fields, []string{"license"}, "library"); err != nil {
		return nil, err
	}
	result := &Library{}
	if raw, ok := fields["license"]; ok {
		if result.License, err = parseLicense(raw, "library.license"); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// parseLicense validates a license declaration and returns the library paths of its declared files, without
// checking file presence. Declared paths are relative to the repository root and must be library-wide files, never
// a rule's Markdown file or a file in an asset directory inside a group. Unknown fields are rejected.
func parseLicense(input json.RawMessage, location string) (*LicenseDeclaration, error) {
	fields, err := decode.Object(input, location)
	if err != nil {
		return nil, err
	}
	if err := decode.KnownFields(fields, []string{"file", "notices", "spdxExpression"}, location); err != nil {
		return nil, err
	}
	file, err := termsPath(fields["file"], location+".file")
	if err != nil {
		return nil, err
	}
	var rawNotices []json.RawMessage
	if json.Unmarshal(fields["notices"], &rawNotices) != nil || rawNotices == nil {
		return nil, invalid(location+".notices", "expected an array of notice paths")
	}
	notices := make([]string, len(rawNotices))
	for i, raw := range rawNotices {
		path, err := termsPath(raw, fmt.Sprintf("%s.notices[%d]", location, i))
		if err != nil {
			return nil, err
		}
		notices[i] = path
	}
	var expression *string
	if raw, ok := fields["spdxExpression"]; ok {
		text, err := decode.Text(raw, location+".spdxExpression")
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

// termsPath reads a declared license or notice path, relative to the repository root, and returns its library path.
// The file must be library-wide: outside .code-rules/, or in its assets/ or a group in its rules/. A rule's Markdown
// file or a file in an asset directory inside a group belongs to that rule's version, so declaring it would let a
// library release replace a rule version's content.
func termsPath(input json.RawMessage, location string) (string, error) {
	declared, err := decode.Path(input, location)
	if err != nil {
		return "", err
	}
	path, ok := librarytree.LibraryPath(declared)
	if !ok || path == librarytree.Manifest || strings.HasPrefix(path, "changes/") {
		return "", invalid(location, decode.Quote(declared)+" can't hold a license or notice file; keep it outside .code-rules/, such as LICENSE.md at the repository root, or in .code-rules/assets/. At the repository root, config.yaml and the assets/, changes/, practices/, and techs/ folders are reserved for the library's own files")
	}
	if librarytree.IsRuleContent(path) {
		return "", invalid(location, decode.Quote(declared)+" belongs to a rule's version; keep license and notice files outside rules and their asset directories, such as LICENSE.md at the repository root")
	}
	return path, nil
}

// controlCharacter identifies bytes that cannot occur in a single-line SPDX expression.
func controlCharacter(r rune) bool { return r < 32 || r == 127 }

// LicensePaths returns unique retained library paths in UTF-16 code-unit order without modifying the declaration.
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
