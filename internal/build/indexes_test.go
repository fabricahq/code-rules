// Verify page boundaries, directory completeness, and actionable discovery links.

package build_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/build"
)

// TestIndexPagesMeasuresBytes covers multibyte text at an exact fit and preserves complete ordered entries.
func TestIndexPagesMeasuresBytes(t *testing.T) {
	entries := []string{"第一の項目"}
	whole, err := build.IndexPages("RULES.md", "# Index", entries, "End", 1000)
	if err != nil {
		t.Fatal(err)
	}
	size := len(whole["RULES.md"])
	exact, err := build.IndexPages("RULES.md", "# Index", entries, "End", size)
	if err != nil || exact["RULES.md"] != whole["RULES.md"] {
		t.Fatalf("exact fit: %v", err)
	}
	if output, err := build.IndexPages("RULES.md", "# Index", entries, "End", size-1); err == nil || output != nil {
		t.Fatal("oversize entry accepted")
	}
	entries = nil
	for i := range 7 {
		entries = append(entries, fmt.Sprintf("Entry %d: %s", i, strings.Repeat("界", 45)))
	}
	pages, err := build.IndexPages("groups/techs/go.md", "# Go", entries, "Footer", 400)
	if err != nil {
		t.Fatal(err)
	}
	combined := ""
	for file, text := range pages {
		if len(text) > 400 {
			t.Fatalf("oversized %s: %d", file, len(text))
		}
		if file != "groups/techs/go.md" {
			combined += text
			if !strings.Contains(pages["groups/techs/go.md"], strings.TrimPrefix(file, "groups/techs/")) {
				t.Fatal("part absent from directory")
			}
		}
	}
	for _, entry := range entries {
		if strings.Count(combined, entry) != 1 {
			t.Fatalf("missing or repeated %q", entry)
		}
	}
}

// TestIndexPagesRejectsUnboundedDirectory refuses partial output when the complete parts list cannot fit.
func TestIndexPagesRejectsUnboundedDirectory(t *testing.T) {
	entries := make([]string, 100)
	for i := range entries {
		entries[i] = strings.Repeat("x", 180)
	}
	if pages, err := build.IndexPages("RULES.md", "# Index", entries, "", 250); err == nil || pages != nil {
		t.Fatal("accepted incomplete part directory")
	}
	if _, err := build.IndexPages("../RULES.md", "", nil, "", 100); err == nil {
		t.Fatal("accepted escaping output")
	}
}

// TestRenderIndexesLinksToEffectiveDefinitions includes replacements and excludes full bodies.
func TestRenderIndexesLinksToEffectiveDefinitions(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	resolved, err := build.Resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := build.RenderIndexes(resolved, 8000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pages["RULES.md"], "groups/techs/go.md") || !strings.Contains(pages["groups/techs/go.md"], "../../rules/team/techs/go/errors.md") {
		t.Fatal(pages)
	}
	if strings.Contains(pages["groups/techs/go.md"], "Return errors to the caller.") {
		t.Fatal("body leaked into summary")
	}
}
