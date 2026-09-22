// Verify page boundaries, directory completeness, and actionable discovery links.

package build

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"

	"github.com/yuin/goldmark/v2/parser"
	htmlrenderer "github.com/yuin/goldmark/v2/renderer/html"
)

// TestIndexPagesMeasuresLines covers source lines at an exact fit and preserves complete ordered entries.
func TestIndexPagesMeasuresLines(t *testing.T) {
	entries := []string{"第一の項目"}
	whole, err := indexPages("RULES.md", "# Index", entries, "End", 1000)
	if err != nil {
		t.Fatal(err)
	}
	size := strings.Count(whole["RULES.md"], "\n")
	exact, err := indexPages("RULES.md", "# Index", entries, "End", size)
	if err != nil || exact["RULES.md"] != whole["RULES.md"] {
		t.Fatalf("exact fit: %v", err)
	}
	if output, err := indexPages("RULES.md", "# Index", entries, "End", size-1); err == nil || output != nil {
		t.Fatal("oversize entry accepted")
	}
	entries = nil
	for i := range 7 {
		entries = append(entries, fmt.Sprintf("Entry %d: %s", i, strings.Repeat("界\r\n", 240)))
	}
	pages, err := indexPages("groups/techs/go.md", "# Go", entries, "Footer", 400)
	if err != nil {
		t.Fatal(err)
	}
	combined := ""
	for file, text := range pages {
		if strings.Count(text, "\n") > 400 {
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
	entries := make([]string, 200)
	for i := range entries {
		entries[i] = strings.Repeat("x\n", 180)
	}
	if pages, err := indexPages("RULES.md", "# Index", entries, "", 250); err == nil || pages != nil {
		t.Fatal("accepted incomplete part directory")
	}
	if _, err := indexPages("../RULES.md", "", nil, "", 100); err == nil {
		t.Fatal("accepted escaping output")
	}
}

// TestRenderIndexesLinksToEffectiveDefinitions includes replacements and excludes full bodies.
func TestRenderIndexesLinksToEffectiveDefinitions(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	resolved, err := resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := renderIndexes(resolved, defaultIndexMaxLines, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pages["RULES.md"], "groups/techs/go.md") || !strings.Contains(pages["groups/techs/go.md"], "../../rules/team/techs/go/errors.md") {
		t.Fatal(pages)
	}
	if strings.Contains(pages["RULES.md"], "No active rules are selected") {
		t.Fatal("active rules labeled as empty")
	}
	if strings.Contains(pages["groups/techs/go.md"], "Return errors to the caller.") {
		t.Fatal("body leaked into summary")
	}
}

// TestIndexPagesRejectsMalformedUTF8 rejects invalid content before returning any pages.
func TestIndexPagesRejectsMalformedUTF8(t *testing.T) {
	bad := string([]byte{0xff})
	for _, content := range [][]string{{bad, "entry", "footer"}, {"header", bad, "footer"}, {"header", "entry", bad}} {
		if output, err := indexPages("RULES.md", content[0], []string{content[1]}, content[2], 1000); err == nil || output != nil {
			t.Fatal("accepted invalid UTF-8")
		}
	}
}

// TestRenderEmptyGroup explains why an adopted group has no rule summaries.
func TestRenderEmptyGroup(t *testing.T) {
	config, libraries := fixture(t, `{"techs/go/errors":"Not applicable"}`, `{}`)
	resolved, err := resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := renderIndexes(resolved, defaultIndexMaxLines, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pages["groups/techs/go.md"], "No active rules in this group.") {
		t.Fatal(pages)
	}
	if !strings.Contains(pages["RULES.md"], "No active rules are selected for this project.") || !strings.Contains(pages["RULES.md"], "groups/techs/go.md") {
		t.Fatal("empty selection must be explicit while preserving selected group links", pages)
	}
}

// TestGroupPagesRepeatResolvedReadingGuidance keeps group selection cues available on every standalone page.
func TestGroupPagesRepeatResolvedReadingGuidance(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(fmt.Sprint("local=", local), func(t *testing.T) {
			config, libraries := fixture(t, `{}`, `{}`)
			var files map[string][]byte
			want, unwanted := "When editing Go.", ""
			description, unwantedDescription := "Go guidance.", ""
			if local {
				files = map[string][]byte{"techs/go/_group.json": []byte(`{"name":"Project Go","description":"Project guidance.","whenToRead":"When editing this project."}`)}
				want, unwanted = "When editing this project.", "When editing Go."
				description, unwantedDescription = "Project guidance.", "Go guidance."
			}
			resolved, err := resolve(config, libraries, files)
			if err != nil {
				t.Fatal(err)
			}
			original := resolved.Groups[0].Rules[0]
			for i := range 70 {
				active := original
				active.Rule.ID = fmt.Sprintf("team:techs/go/rule-%d", i)
				resolved.Groups[0].Rules = append(resolved.Groups[0].Rules, active)
			}
			pages, err := renderIndexes(resolved, defaultIndexMaxLines, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := pages["groups/techs/go.part-1.md"]; !ok {
				t.Fatal("fixture did not exercise pagination")
			}
			for file, page := range pages {
				if !strings.Contains(page, "**When to read this group:** "+want) || (unwanted != "" && strings.Contains(page, unwanted)) {
					t.Fatalf("%s lost resolved guidance: %s", file, page)
				}
				if !strings.Contains(page, description+"\n\n**When to read this group:**") || (unwantedDescription != "" && strings.Contains(page, unwantedDescription)) {
					t.Fatalf("%s lost resolved description or included overridden description", file)
				}
				instructions := []string{"These files are generated. Edit source rules or configuration and rebuild to change them."}
				if strings.HasPrefix(file, "groups/") {
					instructions = append(instructions, "## How to use this group", "1. **Read every rule in full.** Open every “Read full rule” link", "2. **Determine which rules apply.**", "3. **Follow every applicable rule.**", "4. **Support each reported violation with evidence.**", "5. **Recheck after changes.**", "do not copy the rule’s impact level", "## Rules")
				} else {
					instructions = append(instructions, "## How to use this file", "1. **Assess every group.**", "use **When to read this group** to decide whether to open it", "2. **Open every relevant or plausibly relevant group.**", "3. **Read every rule in each opened group completely.**", "4. **Apply the rules that govern your task.**", "5. **Reassess when context changes.**", "testing guidance can apply even when no test files have changed", "Exclusions and replacements are already applied", "## Technology and practice group indexes")
				}
				for _, instruction := range instructions {
					if !strings.Contains(page, instruction) {
						t.Fatalf("%s omitted %q", file, instruction)
					}
				}
				if !strings.Contains(strings.ToLower(page), "after compaction") || strings.Count(page, "\n") > defaultIndexMaxLines {
					t.Fatalf("%s lost reading instructions or exceeded its budget", file)
				}
			}
		})
	}
}

// TestGroupPagesKeepMultipleSourceCues labels each imported definition when no local override applies.
func TestGroupPagesKeepMultipleSourceCues(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	resolved, err := resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolved.Groups[0].EffectiveGuidance = append(resolved.Groups[0].EffectiveGuidance, groupGuidance{Source: "second", Metadata: rules.GroupMetadata{Name: "Go Services", Description: "Other guidance.", WhenToRead: "When reviewing services."}})
	pages, err := renderIndexes(resolved, defaultIndexMaxLines, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"RULES.md", "groups/techs/go.md"} {
		for _, cue := range []string{"**team: Go:**\n\n**Description:** Go guidance.\n\n**When to read this group:** When editing Go.", "**second: Go Services:**\n\n**Description:** Other guidance.\n\n**When to read this group:** When reviewing services."} {
			if !strings.Contains(pages[file], cue) {
				t.Fatalf("%s omitted %q", file, cue)
			}
		}
	}
}

// TestIndexPagesRejectsNonportablePaths exercises direct callers before either pagination path returns output.
func TestIndexPagesRejectsNonportablePaths(t *testing.T) {
	for _, file := range []string{"groups/bad:name.md", "groups/bad\nname.md", "groups/bad\x7fname.md"} {
		if pages, err := indexPages(file, "# Index", []string{"Entry"}, "", 8000); err == nil || pages != nil {
			t.Fatalf("accepted nonportable path %q", file)
		}
	}
}

// BenchmarkIndexPagesLarge exercises large multi-page inputs rather than the complete-page fast path.
func BenchmarkIndexPagesLarge(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			entries := make([]string, count)
			for i := range entries {
				entries[i] = strings.Repeat("x", 100)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := indexPages("RULES.md", "# Rules", entries, "Footer", defaultIndexMaxLines); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestIndexPagesDefaultBoundary keeps exactly 750 lines together and splits only above it.
func TestIndexPagesDefaultBoundary(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n"} {
		entries := []string{strings.Repeat("界"+ending, 372), strings.Repeat("b"+ending, 366)}
		exact, err := indexPages("RULES.md", "# Rules", entries, "Footer", defaultIndexMaxLines)
		if err != nil || len(exact) != 1 || strings.Count(exact["RULES.md"], "\n") != 750 {
			t.Fatalf("750 lines: %v, %v", exact, err)
		}
		entries[1] += ending
		split, err := indexPages("RULES.md", "# Rules", entries, "Footer", defaultIndexMaxLines)
		if err != nil || len(split) != 3 {
			t.Fatalf("751 lines: %v, %v", split, err)
		}
		for file, page := range split {
			if strings.Count(page, "\n") > 750 {
				t.Fatalf("%s exceeds 750 lines", file)
			}
		}
	}
	// A very long source line must not trigger byte-based pagination.
	pages, err := indexPages("RULES.md", "# Rules", []string{strings.Repeat("界", 10000)}, "", defaultIndexMaxLines)
	if err != nil || len(pages) != 1 {
		t.Fatalf("long line: %v", err)
	}
}

// TestOrdinarySummaryStaysTogether prevents premature part directories for small groups.
func TestOrdinarySummaryStaysTogether(t *testing.T) {
	config, libraries := fixture(t, "{}", "{}")
	resolved, err := resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 11 {
		resolved.Groups[0].Rules = append(resolved.Groups[0].Rules, resolved.Groups[0].Rules[0])
	}
	pages, err := renderIndexes(resolved, defaultIndexMaxLines, 0)
	if err != nil || len(pages) != 2 {
		t.Fatalf("twelve summaries should stay together: %v", err)
	}
}

// TestGroupDescriptionsEscapeMarkdown keeps authored descriptions from adding links or headings.
func TestGroupDescriptionsEscapeMarkdown(t *testing.T) {
	config, libraries := fixture(t, "{}", "{}")
	resolved, err := resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolved.Groups[0].EffectiveGuidance[0].Metadata.Description = "[guide](https://example.com)\n# Heading"
	pages, err := renderIndexes(resolved, defaultIndexMaxLines, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"RULES.md", "groups/techs/go.md"} {
		if !strings.Contains(pages[file], `\[guide\]\(https://example.com\) \# Heading`) {
			t.Fatalf("%s description was not escaped: %s", file, pages[file])
		}
	}
}

// TestGroupDescriptionsRemainLiteral prevents descriptions from becoming Markdown block syntax.
func TestGroupDescriptionsRemainLiteral(t *testing.T) {
	config, libraries := fixture(t, "{}", "{}")
	resolved, err := resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, description := range []string{"---", "- Text", "1. Text"} {
		resolved.Groups[0].EffectiveGuidance[0].Metadata.Description = description
		pages, err := renderIndexes(resolved, defaultIndexMaxLines, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range []string{"RULES.md", "groups/techs/go.md"} {
			var html bytes.Buffer
			if err := htmlrenderer.New().Render(&html, []byte(pages[file]), parser.New().Parse([]byte(pages[file]))); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(html.String(), "<p><strong>Description:</strong> "+description+"</p>") {
				t.Fatalf("%s lost literal description %q", file, description)
			}
		}
	}
}

// TestIndexPagesNavigation labels total pages and links adjacent pages without exceeding the line limit.
func TestIndexPagesNavigation(t *testing.T) {
	for _, total := range []int{2, 3, 12} {
		entries := make([]string, total)
		for i := range entries {
			entries[i] = fmt.Sprintf("Entry %d\n%s", i, strings.Repeat("Content\n", 50))
		}
		pages, err := indexPages("groups/go tips.md", "# Go", entries, "Footer", 70)
		if err != nil {
			t.Fatal(err)
		}
		if len(pages) != total+1 {
			t.Fatalf("got %d files, want %d", len(pages), total+1)
		}
		for page := 1; page <= total; page++ {
			text := pages[fmt.Sprintf("groups/go tips.part-%d.md", page)]
			label := fmt.Sprintf("**Page %d of %d**", page, total)
			if !strings.Contains(text, "\n\n---\n\nFooter\n\n"+label) {
				t.Fatal("footer is not separated from the last entry")
			}
			lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
			if !strings.HasPrefix(lines[len(lines)-1], label) {
				t.Fatal("navigation must end the page")
			}
			if strings.Count(text, label) != 2 {
				t.Fatalf("missing top/bottom label %q", label)
			}
			if strings.Count(text, "\n") > 70 {
				t.Fatal("navigation exceeded the page limit")
			}
			if !strings.Contains(text, "[All pages](go%20tips.md)") {
				t.Fatal("missing directory link")
			}
			if page < total {
				link := fmt.Sprintf("[Next page](go%%20tips.part-%d.md)", page+1)
				if strings.Count(text, link) != 2 {
					t.Fatalf("missing %s", link)
				}
			} else if strings.Contains(text, "[Next page]") {
				t.Fatal("last page points beyond the end")
			}
			if page > 1 {
				link := fmt.Sprintf("[Previous page](go%%20tips.part-%d.md)", page-1)
				if strings.Count(text, link) != 2 {
					t.Fatalf("missing %s", link)
				}
			} else if strings.Contains(text, "[Previous page]") {
				t.Fatal("first page points before the start")
			}
		}
	}
	pages, err := indexPages("RULES.md", "# Rules", []string{"Short summary"}, "", 750)
	if err != nil || len(pages) != 1 || strings.Contains(pages["RULES.md"], "**Page ") {
		t.Fatal("unpaginated output changed")
	}
}

// TestIndexPagesWithoutFooter keeps navigation last when no notice is supplied.
func TestIndexPagesWithoutFooter(t *testing.T) {
	pages, err := indexPages("RULES.md", "# Rules", []string{strings.Repeat("one\n", 40), strings.Repeat("two\n", 40)}, "", 60)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		page := pages[fmt.Sprintf("RULES.part-%d.md", i)]
		if !strings.Contains(page, fmt.Sprintf("\n\n---\n\n**Page %d of 2**", i)) {
			t.Fatal("empty footer added spacing before navigation")
		}
	}
}
