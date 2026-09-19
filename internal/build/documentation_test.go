// Verify copyable website examples through the same resolver and renderer used by the CLI.

package build_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestDocumentationRules rejects documentation drift that would make copied examples fail a real build.
func TestDocumentationRules(t *testing.T) {
	for _, tc := range []struct{ file, pattern, title string }{
		{"../../docs/src/examples/make-errors-actionable.md", "", "Make error messages actionable"},
		{"../../docs/src/content/docs/guides/write-rules.md", "(?s)```md\\n(---.*?)\\n```", "Verify retry limits"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			if tc.pattern != "" {
				match := regexp.MustCompile(tc.pattern).FindSubmatch(data)
				if len(match) != 2 {
					t.Fatal("missing copyable rule example")
				}
				data = match[1]
			}
			config, err := rules.ParseConfiguration([]byte(`{"schemaVersion":1,"sources":{}}`))
			if err != nil {
				t.Fatal(err)
			}
			output, err := build.Generate(config, nil, map[string][]byte{
				"practices/testing/_group.json": []byte(`{"name":"Testing","description":"Verify behavior.","whenToRead":"Changing behavior."}`),
				"practices/testing/example.md":  data,
			}, build.Options{ToolVersion: "documentation-example", IndexMaxLines: 750})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(output.Files["rules/local/practices/testing/example.md"]), tc.title) {
				t.Fatal("missing rendered rule")
			}
		})
	}
}
