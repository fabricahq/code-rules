// Validate rendered guidance, relocation, source preservation, and unsafe link failures.

package build_test

import (
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/build"
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

// TestRenderPinnedRemoteFallback selects a commit URL for an unretained rule, never an asset.
func TestRenderPinnedRemoteFallback(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	library := libraries["team"]
	library.Catalog.Groups[0].Rules[0].Document = document + "\n[other](../rust/other.md#x)\n"
	libraries["team"] = library
	resolved, err := build.Resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	output, err := build.RenderRules(resolved)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output["rules/team/techs/go/errors.md"], "https://github.com/acme/rules/blob/"+commit+"/techs/rust/other.md#x") {
		t.Fatal(output)
	}
}
