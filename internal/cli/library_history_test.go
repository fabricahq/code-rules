// Exercise library commands that read a library's release tags, through the compiled CLI and real Git history.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// releaseOne publishes rules a and b at 1.0.0, as the first library release must.
const releaseOne = "Library release 1.\n---\nrelease: 1\nrules:\n  practices/testing/a: 1.0.0\n  practices/testing/b: 1.0.0\nchanges:\n  practices/testing/a:\n    change: new\n    summary: Add a.\n  practices/testing/b:\n    change: new\n    summary: Add b.\n"

// libraryRule returns a complete rule document with one line of guidance.
func libraryRule(guidance string) string {
	return "---\ntitle: Test retries\nimpact: HIGH\nimpactDescription: Catch retry bugs.\nwhenToRead: When changing retries.\n---\n" + guidance + "\n"
}

// releasedLibrary returns a fixture whose release/1 published rules a and b, and an author's full clone of it.
func releasedLibrary(t *testing.T) (*gitfixture.Fixture, string) {
	t.Helper()
	ctx := context.Background()
	fixture, err := gitfixture.New(ctx, map[string][]byte{
		"rule-library.yaml":             []byte("formatVersion: 1\n"),
		"practices/testing/_group.yaml": []byte("name: Testing\ndescription: Testing guidance.\nwhenToRead: When testing.\n"),
		"practices/testing/a.md":        []byte(libraryRule("Test the retry limit.")),
		"practices/testing/b.md":        []byte(libraryRule("Test the backoff.")),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := fixture.Command(ctx, "tag", "--annotate", "--cleanup=verbatim", "--message", releaseOne, "release/1"); err != nil {
		t.Fatal(err)
	}
	dir, err := fixture.Clone(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return fixture, dir
}

// writeFiles writes each file under dir, creating parent directories.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

// TestLibraryCheck_PreviewsThePendingLibraryRelease shows each pending rule's versions in human and JSON output.
func TestLibraryCheck_PreviewsThePendingLibraryRelease(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	writeFiles(t, dir, map[string]string{
		"practices/testing/a.md":       libraryRule("Test the retry limit and one past it."),
		"practices/testing/retries.md": libraryRule("Test every retry."),
		"changes/a.yaml":               "summary: Test one past the limit.\nrules:\n  practices/testing/a: minor\n",
		"changes/retries.yaml":         "summary: Add a rule about testing retries.\nrules:\n  practices/testing/retries: new\n",
	})
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "check")
	want := "Library is valid: 1 group(s), 3 rule(s).\nWarning: License is undeclared. Decide terms before sharing this library.\n\nPending library release 2\n  practices/testing/a        minor  1.0.0 -> 1.1.0\n  practices/testing/retries  new    1.0.0\n"
	if code != 0 || diagnostic != "" || out != want {
		t.Fatalf("exit %d, stderr %q, stdout:\n%s\nwant:\n%s", code, diagnostic, out, want)
	}
	out, diagnostic, code = runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "check", "--json")
	var response struct {
		OK    bool
		Value struct {
			PendingRelease json.RawMessage `json:"pendingRelease"`
		}
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 0 || diagnostic != "" || !response.OK {
		t.Fatal(err, code, out, diagnostic)
	}
	wantJSON := `{"release":2,"rules":[{"id":"practices/testing/a","change":"minor","currentVersion":"1.0.0","nextVersion":"1.1.0"},{"id":"practices/testing/retries","change":"new","nextVersion":"1.0.0"}]}`
	if compact := compactJSON(t, response.Value.PendingRelease); compact != wantJSON {
		t.Fatalf("pendingRelease %s, want %s", compact, wantJSON)
	}
}

// TestLibraryCheck_ReportsMissingNotesAndShallowClones gives the failure's code in JSON and its repair in human output.
func TestLibraryCheck_ReportsMissingNotesAndShallowClones(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	writeFiles(t, dir, map[string]string{"practices/testing/a.md": libraryRule("Changed.")})
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "check")
	if code != 1 || out != "" || !strings.Contains(diagnostic, "practices/testing/a changed since release/1, where its version is 1.0.0, and no pending change note names it. Record it with: code-rules library change practices/testing/a") {
		t.Fatal(code, out, diagnostic)
	}
	assertJSONError(t, binary, dir, fixture.Environment, "change-notes")
	shallow := filepath.Join(t.TempDir(), "shallow")
	if _, err := fixture.CommandIn(context.Background(), filepath.Dir(shallow), "clone", "--quiet", "--depth=1", "--template=", fixture.Repository, shallow); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code = runCLIWithEnvironment(t, binary, shallow, fixture.Environment, "library", "check")
	if code != 1 || out != "" || !strings.Contains(diagnostic, "git fetch --unshallow --tags") {
		t.Fatal(code, out, diagnostic)
	}
	assertJSONError(t, binary, shallow, fixture.Environment, "shallow-clone")
}

// assertJSONError runs library check with --json and requires a failure with code.
func assertJSONError(t *testing.T, binary, dir string, environment []string, code string) {
	t.Helper()
	out, diagnostic, exit := runCLIWithEnvironment(t, binary, dir, environment, "library", "check", "--json")
	var response struct {
		OK    bool
		Error responseError
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || exit != 1 || diagnostic != "" || response.OK || response.Error.Code != code {
		t.Fatal(err, exit, out, diagnostic)
	}
}

// compactJSON removes insignificant whitespace, keeping field order, so JSON compares as text.
func compactJSON(t *testing.T, data json.RawMessage) string {
	t.Helper()
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		t.Fatal(err)
	}
	return compact.String()
}
