// Check that every project command refuses a project that still keeps authored files in local/.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// localFolderProblem is the problem every project command reports for the project below, which still keeps a
// group, a rule, and a shared asset in local/.
const localFolderProblem = `this version of Code Rules reads authored rules and groups from .code-rules/rules/, and shared assets from .code-rules/assets/. It no longer reads .code-rules/local/, and doesn't move its files. Move them yourself, keeping each path below local/, then delete local/:

  local/assets/retry.svg              ->  assets/retry.svg
  local/practices/legacy/_group.yaml  ->  rules/practices/legacy/_group.yaml
  local/practices/legacy/old-rule.md  ->  rules/practices/legacy/old-rule.md

Rule IDs don't change, so config.yaml needs no edits. A moved rule sits one folder deeper, so its links to shared assets need one more ../, such as ../../../assets/retry-lifecycle.svg. Then run code-rules project build.`

// TestProjectCommands_RejectALocalFolder refuses, writing nothing, every project command in a project whose Code
// Rules directory still has local/, and lists where each of its files belongs now, in human and JSON output.
func TestProjectCommands_RejectALocalFolder(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	if out, diagnostic, code := runCLI(t, binary, directory, "project", "init"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	for name, content := range map[string]string{
		"local/practices/legacy/_group.yaml": `{"name":"Legacy","description":"Legacy guidance.","whenToRead":"When writing code."}`,
		"local/practices/legacy/old-rule.md": "---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Keep failures visible.\nwhenToRead: When calling functions.\n---\nSee [the diagram](../../assets/retry.svg).\n",
		"local/assets/retry.svg":             "<svg/>",
	} {
		path := filepath.Join(directory, ".code-rules", name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	before := projectFileContents(t, directory)
	// Sync and update also say that they wrote nothing, as they do for every failure.
	unchanged := " No files were written."
	for _, test := range []struct {
		args    []string
		outcome string
	}{
		{[]string{"project", "init"}, ""},
		{[]string{"project", "build"}, ""},
		{[]string{"project", "check"}, ""},
		{[]string{"project", "sync"}, unchanged},
		{[]string{"project", "update", "--yes"}, unchanged},
		{[]string{"project", "add", "group", "practices/testing", "--name", "Testing", "--description", "Testing guidance.", "--when-to-read", "When writing tests."}, ""},
		{[]string{"project", "add", "rule", "techs/go/wrap-errors", "--title", "Wrap errors", "--impact", "HIGH", "--impact-description", "Keep context.", "--when-to-read", "When returning errors."}, ""},
		{[]string{"project", "add", "rule", "techs/go/shared", "--from", "team@1.0.0", "--non-interactive"}, ""},
		{[]string{"project", "add", "source", "team", "--repository", "https://example.invalid/team.git", "--rules", "techs/go/*", "--non-interactive"}, ""},
	} {
		t.Run(strings.Join(test.args[1:min(len(test.args), 4)], " "), func(t *testing.T) {
			want := "Error: " + localFolderProblem + test.outcome + "\n\nLocation: .code-rules/local/\n"
			if out, diagnostic, code := runCLI(t, binary, directory, test.args...); code != 1 || out != "" || diagnostic != want {
				t.Errorf("exit %d\n%s%s", code, out, diagnostic)
			}
			out, diagnostic, code := runCLI(t, binary, directory, append(test.args, "--json")...)
			var result response
			if err := json.Unmarshal([]byte(out), &result); err != nil || code != 1 || diagnostic != "" || result.OK || result.Error == nil {
				t.Fatalf("exit %d\n%s%s", code, out, diagnostic)
			}
			if got := *result.Error; got != (responseError{Kind: "validation", Message: ".code-rules/local/: " + localFolderProblem + test.outcome, Location: ".code-rules/local/"}) {
				t.Errorf("error %+v", got)
			}
		})
	}
	if !reflect.DeepEqual(before, projectFileContents(t, directory)) {
		t.Fatal("a command changed the project")
	}
}
