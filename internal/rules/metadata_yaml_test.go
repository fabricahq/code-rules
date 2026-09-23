// Validate authored YAML group and library metadata without filesystem side effects.

package rules_test

import (
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestYAMLMetadata(t *testing.T) {
	group := "# Guidance\nname: Testing\ndescription: Verify behavior.\nwhenToRead: >\n  When writing\n  or reviewing tests.\n"
	metadata, err := rules.ParseGroupMetadataYAML([]byte(group), "practices/testing/_group.yaml")
	if err != nil || metadata.WhenToRead != "When writing or reviewing tests." {
		t.Fatal(metadata, err)
	}
	manifest := "# Library terms\nformatVersion: 1\nlicense:\n  spdxExpression: MIT\n  file: LICENSE.md\n  notices:\n    - NOTICE.md\n"
	license, err := rules.ParseLibraryLicense([]byte(manifest), "library")
	if err != nil || license == nil || *license.SPDXExpression != "MIT" || len(license.AttributionFiles) != 1 {
		t.Fatal(license, err)
	}
}

func TestYAMLMetadataRejectsAmbiguousInput(t *testing.T) {
	group := "name: Testing\ndescription: Tests\nwhenToRead: When testing\n"
	for name, input := range map[string]string{
		"duplicate": "name: First\n" + group,
		"alias":     "name: *name\ndescription: Tests\nwhenToRead: When testing\n",
		"anchor":    strings.Replace(group, "Testing", "&name Testing", 1),
		"tag":       strings.Replace(group, "Testing", "!!str Testing", 1),
		"unknown":   group + "extra: true\n", "wrong-type": strings.Replace(group, "Testing", "false", 1),
		"multiple": group + "---\n" + group, "invalid-utf8": group + "#\xff", "empty": "", "nonmapping": "[]",
	} {
		t.Run("group/"+name, func(t *testing.T) {
			if _, err := rules.ParseGroupMetadataYAML([]byte(input), "group"); err == nil {
				t.Fatal("accepted invalid metadata")
			}
		})
	}
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
