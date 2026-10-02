// Validate authored YAML library metadata without filesystem side effects.

package rules_test

import (
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestYAMLMetadata(t *testing.T) {
	manifest := "# Library terms\nformatVersion: 1\nlicense:\n  spdxExpression: MIT\n  file: LICENSE.md\n  notices:\n    - NOTICE.md\n"
	license, err := rules.ParseLibraryLicense([]byte(manifest), "library")
	if err != nil || license == nil || *license.SPDXExpression != "MIT" || len(license.AttributionFiles) != 1 {
		t.Fatal(license, err)
	}
}

func TestYAMLMetadataRejectsAmbiguousInput(t *testing.T) {
	for name, input := range map[string]string{
		"duplicate": "formatVersion: 1\nformatVersion: 1\n", "multiple": "formatVersion: 1\n---\nformatVersion: 1\n",
		"anchor": "formatVersion: &version 1", "alias": "formatVersion: *version", "tag": "formatVersion: !!int 1",
		"wrong-type": "formatVersion: false", "invalid-utf8": "formatVersion: 1\n#\xff", "empty": "", "nonmapping": "[]",
		"nested-duplicate": "formatVersion: 1\nlicense: {file: LICENSE.md, file: NOTICE.md, notices: []}",
	} {
		t.Run("library/"+name, func(t *testing.T) {
			if _, err := rules.ParseLibraryLicense([]byte(input), "library"); err == nil {
				t.Fatal("accepted invalid metadata")
			}
		})
	}
}
