// Exercise library commands that read a library's release tags, through the compiled CLI and real Git history.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
	"github.com/fabricahq/code-rules/internal/test/terminalfixture"
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

// changeNotes returns the names of the notes in dir's changes/ directory.
func changeNotes(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, "changes"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// TestLibraryChange_RecordsNotesThatCheckAccepts writes a uniquely named note on every run and reports it.
func TestLibraryChange_RecordsNotesThatCheckAccepts(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	writeFiles(t, dir, map[string]string{"practices/testing/a.md": libraryRule("Test the retry limit in Python too.")})
	args := []string{"library", "change", "practices/testing/a", "--bump", "minor", "--summary", "Add a Python example of the retry-limit test."}
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, args...)
	notes := changeNotes(t, dir)
	generated := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}-a-[0-9a-f]{6}\.yaml$`)
	if code != 0 || diagnostic != "" || len(notes) != 1 || !generated.MatchString(notes[0]) {
		t.Fatal(code, out, diagnostic, notes)
	}
	physical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(physical, "changes", notes[0])
	want := "Change note created:\n  " + note + "\n\nNext: Commit the note with the rule change, then validate the library:\n  code-rules library check\n"
	if out != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out, want)
	}
	if data, err := os.ReadFile(note); err != nil || string(data) != "summary: Add a Python example of the retry-limit test.\nrules:\n  practices/testing/a: minor\n" {
		t.Fatalf("%q %v", data, err)
	}
	if out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "check"); code != 0 || !strings.Contains(out, "practices/testing/a  minor  1.0.0 -> 1.1.0") {
		t.Fatal(code, out, diagnostic)
	}
	out, diagnostic, code = runCLIWithEnvironment(t, binary, dir, fixture.Environment, append(args, "--json")...)
	var response struct {
		OK    bool
		Value authoringValue
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 0 || diagnostic != "" || !response.OK {
		t.Fatal(err, code, out, diagnostic)
	}
	if len(response.Value.Files) != 1 || response.Value.Files[0] == note || !generated.MatchString(filepath.Base(response.Value.Files[0])) || len(response.Value.NextSteps) != 1 || response.Value.NextSteps[0].Commands[0] != "code-rules library check" {
		t.Fatalf("%+v", response.Value)
	}
}

// TestLibraryChange_RejectsInvalidRequests reports usage errors with exit 2 and library mismatches with a code.
func TestLibraryChange_RejectsInvalidRequests(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	writeFiles(t, dir, map[string]string{"practices/testing/a.md": libraryRule("Changed.")})
	for _, args := range [][]string{
		{},
		{"practices/testing/a", "--bump", "huge", "--summary", "Fix a."},
		{"practices/testing/a", "--bump", "minor", "--retire", "--summary", "Retire a."},
		{"practices/testing/a", "--replaced-by", "practices/testing/b", "--bump", "major", "--summary", "Replace a."},
		{"practices/testing/a", "practices/testing/b", "--retire", "--replaced-by", "practices/testing/c", "--summary", "Replace both."},
		{"practices/testing/a", "--bump", "patch", "--summary", "One.\nTwo."},
		{"practices/testing/a", "--bump", "patch", "--non-interactive"},
		{"practices/testing/a", "--summary", "Fix a.", "--json"},
	} {
		out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, append([]string{"library", "change"}, args...)...)
		if code != 2 {
			t.Errorf("%q: exit %d, want 2: %s %s", args, code, out, diagnostic)
		}
	}
	if _, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "change", "practices/testing/a", "--bump", "patch", "--summary", "One.\nTwo."); code != 2 || !strings.Contains(diagnostic, "--summary must be one line") {
		t.Fatal(code, diagnostic)
	}
	out, _, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "change", "practices/testing/c", "--summary", "Add c.", "--json")
	var response struct{ Error responseError }
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 1 || response.Error.Code != "unknown-rule" {
		t.Fatal(err, code, out)
	}
	if notes := changeNotes(t, dir); len(notes) != 0 {
		t.Fatal("wrote notes", notes)
	}
	unreleased := t.TempDir()
	writeFiles(t, unreleased, map[string]string{"rule-library.yaml": "formatVersion: 1\n", "practices/testing/_group.yaml": "name: Testing\ndescription: Testing guidance.\nwhenToRead: When testing.\n", "practices/testing/a.md": libraryRule("Test.")})
	out, _, code = runCLI(t, binary, unreleased, "library", "change", "practices/testing/a", "--summary", "Add a.", "--json")
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 1 || response.Error.Code != "no-library-release" {
		t.Fatal(err, code, out)
	}
}

// TestLibraryChange_PromptsForMissingInputs asks for the change level and summary in a terminal, retrying invalid answers.
func TestLibraryChange_PromptsForMissingInputs(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	writeFiles(t, dir, map[string]string{"practices/testing/a.md": libraryRule("Changed.")})
	args := []string{"library", "change", "practices/testing/a"}
	cancelled, err := terminalfixture.RunWithEnvironment(context.Background(), binary, dir, fixture.Environment, args, []terminalfixture.Step{{Prompt: "Change (major, minor, or patch):", Interrupt: true}})
	if err != nil || cancelled.ExitCode != 1 || len(changeNotes(t, dir)) != 0 {
		t.Fatal(err, cancelled, changeNotes(t, dir))
	}
	steps := []terminalfixture.Step{
		{Prompt: "Change (major, minor, or patch):", Answer: "huge"},
		{Prompt: "Change (major, minor, or patch):", Answer: "minor"},
		{Prompt: "Summary:", Answer: "Add an example for custom hooks."},
	}
	result, err := terminalfixture.RunWithEnvironment(context.Background(), binary, dir, fixture.Environment, args, steps)
	if err != nil || result.ExitCode != 0 {
		t.Fatal(err, result)
	}
	for _, text := range []string{"Recording a change to: practices/testing/a", "major  Work that complied with the previous version could fail this one.", "--bump must be major, minor, or patch"} {
		if !strings.Contains(result.Transcript, text) {
			t.Fatalf("missing %q in transcript:\n%s", text, result.Transcript)
		}
	}
	notes := changeNotes(t, dir)
	if len(notes) != 1 {
		t.Fatal(notes)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "changes", notes[0])); err != nil || string(data) != "summary: Add an example for custom hooks.\nrules:\n  practices/testing/a: minor\n" {
		t.Fatalf("%q %v", data, err)
	}
}

// TestLibraryChange_WarnsAboutAMissingReplacement in its output, after recording the retirement.
func TestLibraryChange_WarnsAboutAMissingReplacement(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	if err := os.Remove(filepath.Join(dir, "practices/testing/b.md")); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "change", "practices/testing/b", "--retire", "--replaced-by", "practices/testing/retries", "--summary", "Covered by the broader rule about testing retries.")
	if code != 0 || diagnostic != "" || !strings.Contains(out, "Warning: practices/testing/retries isn't a rule in the library yet. Add it before the next library release.\n") || len(changeNotes(t, dir)) != 1 {
		t.Fatal(code, out, diagnostic)
	}
}

// TestLibraryAddRule_NextStepsIncludeTheChangeNote after the first library release, and following them passes check.
func TestLibraryAddRule_NextStepsIncludeTheChangeNote(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	writeFiles(t, dir, map[string]string{"body.md": "Test every retry.\n"})
	add := []string{"library", "add", "rule", "practices/testing/retries", "--title", "Test retries", "--when-to-read", "When changing retries.", "--impact", "HIGH", "--impact-description", "Catch retry bugs.", "--body-file", "body.md"}
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, add...)
	want := "\nAfter the first library release, every new rule needs a change note. After writing the rule text, record it with a summary for project maintainers:\n  code-rules library change practices/testing/retries\n\nThen validate the library:\n  code-rules library check\n"
	if code != 0 || diagnostic != "" || !strings.HasSuffix(out, want) {
		t.Fatalf("exit %d, stderr %q, stdout:\n%s", code, diagnostic, out)
	}
	if out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "check"); code != 1 || !strings.Contains(diagnostic, "practices/testing/retries is a new rule") {
		t.Fatal(code, out, diagnostic)
	}
	if out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "change", "practices/testing/retries", "--summary", "Add a rule about testing retries."); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	if out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, fixture.Environment, "library", "check"); code != 0 || !strings.Contains(out, "practices/testing/retries  new  1.0.0") {
		t.Fatal(code, out, diagnostic)
	}
	unreleased, err := fixture.Clone(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.CommandIn(context.Background(), unreleased, "tag", "--delete", "release/1"); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, unreleased, map[string]string{"body.md": "Test every retry.\n"})
	out, diagnostic, code = runCLIWithEnvironment(t, binary, unreleased, fixture.Environment, add...)
	if code != 0 || strings.Contains(out, "library change") || !strings.HasSuffix(out, "After writing the rule text, run:\n  code-rules library check\n") {
		t.Fatalf("exit %d, stderr %q, stdout:\n%s", code, diagnostic, out)
	}
}
