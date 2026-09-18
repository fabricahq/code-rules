// Validate rendered guidance, relocation, source preservation, and unsafe link failures.

package build_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/rules"
)

// renderFixture resolves a local rule with a caller-supplied body and optional supporting files.
func renderFixture(t *testing.T, body string, support map[string][]byte) (map[string]string, error) {
	t.Helper()
	config, libraries := fixture(t, `{}`, `{}`)
	files := map[string][]byte{"techs/go/local.md": []byte(strings.Split(document, "---\n#")[0] + "---\n" + body)}
	for file, data := range support {
		files[file] = data
	}
	resolved, err := build.Resolve(config, libraries, files)
	if err != nil {
		return nil, err
	}
	return build.RenderRules(resolved)
}

// TestRenderRules retains non-rewritten bytes and relocates parsed links without changing code examples.
func TestRenderRules(t *testing.T) {
	body := "# Return errors\n\nA  sentence.\r\n\n[asset](assets/local/a%20b.png#part)\n\n[self]() [ref][r]\n\n[r]: assets/local/a%20b.png\n\n`[code](missing)`\n\n## More\n\nKeep trailing spaces.  \nNext line.\n"
	output, err := renderFixture(t, body, map[string][]byte{"techs/go/assets/local/a b.png": {0, 255}})
	if err != nil {
		t.Fatal(err)
	}
	got := output["rules/local/techs/go/local.md"]
	for _, want := range []string{"A  sentence.\r\n", "../../../../../local/techs/go/assets/local/a%20b.png#part", "[self](../../../../../local/techs/go/local.md)", "`[code](missing)`", "Keep trailing spaces.  \nNext line.", "### Source metadata", "**Rule source:**"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Count(got, "# Return errors") != 1 {
		t.Fatal("retained redundant heading")
	}
}

// TestRenderSetext preserves the paragraph immediately after an underline heading.
func TestRenderSetext(t *testing.T) {
	output, err := renderFixture(t, "Different title\n===\nParagraph after heading.\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := output["rules/local/techs/go/local.md"]
	if !strings.Contains(got, "### Different title\nParagraph after heading.") {
		t.Fatal(got)
	}
}

// TestRenderRejectsUnsafeReferences refuses missing local links and relative HTML attributes.
func TestRenderRejectsUnsafeReferences(t *testing.T) {
	for _, body := range []string{"[missing](missing.md)", "[escape](../../../outside)", "<img src='assets/local/a.png'>", "<a href=missing.md>x</a>"} {
		output, err := renderFixture(t, body, nil)
		if err == nil || output != nil {
			t.Fatalf("accepted %q: %v", body, output)
		}
	}
}

// TestRenderRejectsRuleLinks refuses retained, excluded, and unretained targets without remote fallback.
func TestRenderRejectsRuleLinks(t *testing.T) {
	for _, state := range []string{"selected", "excluded", "unselected"} {
		t.Run(state, func(t *testing.T) {
			exclude := `{}`
			if state == "excluded" {
				exclude = `{"techs/go/other":"Project policy"}`
			}
			config, libraries := fixture(t, exclude, `{}`)
			supplied := libraries["team"]
			target := "techs/go/other.md"
			if state == "unselected" {
				target = "techs/rust/other.md"
			} else {
				other, err := rules.Parse(document, target, "team")
				if err != nil {
					t.Fatal(err)
				}
				supplied.Catalog.Groups[0].Rules = append(supplied.Catalog.Groups[0].Rules, other)
			}
			supplied.Catalog.Groups[0].Rules[0].Document = document + "\n[other](/" + target + "#details)\n"
			libraries["team"] = supplied
			resolved, err := build.Resolve(config, libraries, nil)
			if err != nil {
				t.Fatal(err)
			}
			output, err := build.RenderRules(resolved)
			if err == nil || !strings.Contains(err.Error(), "links to other rule documents are not allowed") || output != nil {
				t.Fatalf("expected rule-link error without partial output: %+v, %v", output, err)
			}
		})
	}
}

// TestRenderPreservesEntityDestinations prevents a second Markdown decode from changing external URL semantics.
func TestRenderPreservesEntityDestinations(t *testing.T) {
	output, err := renderFixture(t, "[x](https://example.com/?q=&amp;copy;)\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output["rules/local/techs/go/local.md"], "?q=&amp;copy;") {
		t.Fatal(output)
	}
}

// TestRenderNestsEmptyHeading preserves empty heading structure below the generated Guidance section.
func TestRenderNestsEmptyHeading(t *testing.T) {
	output, err := renderFixture(t, "#\nAfter.\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output["rules/local/techs/go/local.md"], "###\nAfter.") {
		t.Fatal(output)
	}
}

// TestRenderNestedHeadings nests container headings without consuming following guidance.
func TestRenderNestedHeadings(t *testing.T) {
	for _, body := range []string{"> # Caveat\n> Keep this.\n", "- # Caveat\n\n  Keep this.\n", "> Multi\n> line\n> ===\n> Keep this.\n"} {
		output, err := renderFixture(t, body, nil)
		if err != nil {
			t.Fatal(err)
		}
		got := output["rules/local/techs/go/local.md"]
		if !strings.Contains(got, "### ") || !strings.Contains(got, "Keep this.") {
			t.Fatal(got)
		}
	}
}

// TestRenderIgnoresInertHTML distinguishes comments and data attributes from actual resource URLs.
func TestRenderIgnoresInertHTML(t *testing.T) {
	for _, body := range []string{`Text <script>const example='<img src="draft.md">';</script>`, `Text <textarea><img src="draft.md"></textarea>`, `<!-- href="draft.md" -->`, `<span data-href="draft.md">Text</span>`, `<script>const example='src="draft.md"';</script>`} {
		if _, err := renderFixture(t, body, nil); err != nil {
			t.Fatalf("%q: %v", body, err)
		}
	}
}

// TestRenderRejectsFileDirectoryConflict refuses an output path that must be both file and directory.
func TestRenderRejectsFileDirectoryConflict(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	resolved, err := build.Resolve(config, libraries, map[string][]byte{"techs/go/a.md": []byte(document), "techs/go/a.md/b.md": []byte(document)})
	if err != nil {
		t.Fatal(err)
	}
	if output, err := build.RenderRules(resolved); err == nil || output != nil {
		t.Fatal("accepted file/directory conflict")
	}
}

// TestRenderReferenceImages keeps shared image definitions usable as both images and file links.
func TestRenderReferenceImages(t *testing.T) {
	for _, body := range []string{"![diagram][asset]\n\n[asset]: /assets/diagram.png", "[download][asset] ![diagram][asset]\n\n[asset]: /assets/diagram.png"} {
		config, libraries := fixture(t, `{}`, `{}`)
		lib := libraries["team"]
		lib.Catalog.Groups[0].Rules[0].Document = document + "\n" + body
		lib.Catalog.SupportingFiles["assets/diagram.png"] = []byte("image bytes")
		libraries["team"] = lib
		resolved, err := build.Resolve(config, libraries, nil)
		if err != nil {
			t.Fatal(err)
		}
		output, err := build.RenderRules(resolved)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output["rules/team/techs/go/errors.md"], "../../../../../vendor/team/assets/diagram.png") {
			t.Fatal(output)
		}
	}
}

// TestRenderRejectsInvalidUTF8 prevents local bytes from being silently replaced during JSON delivery.
func TestRenderRejectsInvalidUTF8(t *testing.T) {
	output, err := renderFixture(t, "Guidance: \xff", nil)
	var validation *rules.ValidationError
	if !errors.As(err, &validation) || validation.Location != "local:techs/go/local.md" || !strings.Contains(err.Error(), "UTF-8") || output != nil {
		t.Fatalf("expected contextual UTF-8 error without output, got %v, %v", output, err)
	}
}

// TestRenderRejectsInvalidDocumentBytes checks callers that construct a resolved value directly.
func TestRenderRejectsInvalidDocumentBytes(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	resolved, err := build.Resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolved.Groups[0].Rules[0].Rule.Document += "\xff"
	output, err := build.RenderRules(resolved)
	var validation *rules.ValidationError
	if !errors.As(err, &validation) || validation.Location != "team:techs/go/errors" || !strings.Contains(err.Error(), "UTF-8") || output != nil {
		t.Fatalf("expected contextual UTF-8 error without output, got %v, %v", output, err)
	}
}

// TestRenderFlattensMetadataLineEndings keeps YAML-escaped CR, LF, and CRLF inside one rendered line.
func TestRenderFlattensMetadataLineEndings(t *testing.T) {
	for _, ending := range []string{`\r`, `\n`, `\r\n`} {
		t.Run(ending, func(t *testing.T) {
			config, libraries := fixture(t, `{}`, `{}`)
			value := `"Safe` + ending + `## Extra"`
			text := "---\ntitle: " + value + "\nimpact: HIGH\nimpactDescription: " + value + "\nwhenToRead: " + value + "\n---\nGuidance."
			resolved, err := build.Resolve(config, libraries, map[string][]byte{"techs/go/local.md": []byte(text)})
			if err != nil {
				t.Fatal(err)
			}
			output, err := build.RenderRules(resolved)
			if err != nil {
				t.Fatal(err)
			}
			got := output["rules/local/techs/go/local.md"]
			for _, prefix := range []string{"# ", "**When to read:** ", "**Why it matters:** "} {
				if !strings.Contains(got, prefix+"Safe \\#\\# Extra\n") {
					t.Errorf("metadata was not flattened for %q: %q", prefix, got)
				}
			}
		})
	}
}
