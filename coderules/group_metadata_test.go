// Check that a group's _group.yaml parses to trimmed metadata and that ambiguous or unknown content fails.

package coderules_test

import (
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/coderules"
)

// TestParseGroupMetadata_ReadsAuthoredYAML folds block scalars, ignores comments, and trims text.
func TestParseGroupMetadata_ReadsAuthoredYAML(t *testing.T) {
	group := "# Guidance\nname: ' Testing '\ndescription: Verify behavior.\nwhenToRead: >\n  When writing\n  or reviewing tests.\n"
	got, err := coderules.ParseGroupMetadata([]byte(group), "practices/testing/_group.yaml")
	want := coderules.GroupMetadata{Name: "Testing", Description: "Verify behavior.", WhenToRead: "When writing or reviewing tests."}
	if err != nil || got != want {
		t.Fatalf("got %+v, %v; want %+v", got, err, want)
	}
}

// TestParseGroupMetadata_RefusesAmbiguousOrUnknownContent returns the zero metadata and an error that starts with
// the file's location.
func TestParseGroupMetadata_RefusesAmbiguousOrUnknownContent(t *testing.T) {
	group := "name: Testing\ndescription: Tests\nwhenToRead: When testing\n"
	for name, input := range map[string]string{
		"duplicate":    "name: First\n" + group,
		"alias":        "name: *name\ndescription: Tests\nwhenToRead: When testing\n",
		"anchor":       strings.Replace(group, "Testing", "&name Testing", 1),
		"tag":          strings.Replace(group, "Testing", "!!str Testing", 1),
		"unknown":      group + "extra: true\n",
		"license":      group + "license: MIT\n",
		"wrong-type":   strings.Replace(group, "Testing", "false", 1),
		"blank":        strings.Replace(group, "Testing", "' '", 1),
		"missing":      "name: Testing\ndescription: Tests\n",
		"multiple":     group + "---\n" + group,
		"invalid-utf8": group + "#\xff",
		"empty":        "",
		"nonmapping":   "[]",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := coderules.ParseGroupMetadata([]byte(input), "practices/testing/_group.yaml")
			if err == nil || !strings.HasPrefix(err.Error(), "practices/testing/_group.yaml") || got != (coderules.GroupMetadata{}) {
				t.Fatalf("got %+v, %v; want an error and no metadata", got, err)
			}
		})
	}
}
