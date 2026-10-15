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

// localFolderProblem is the problem every project command reports for a project with a local/ folder.
const localFolderProblem = `this version of Syllabus reads authored rules and groups from .syllabus/rules/, and shared assets from .syllabus/assets/. It no longer reads .syllabus/local/, and doesn't move its files. Move them yourself, keeping each path below local/, then delete local/:

  local/techs/go/_group.yaml         ->  rules/techs/go/_group.yaml
  local/techs/go/errors.md           ->  rules/techs/go/errors.md
  local/techs/go/assets/errors/      ->  rules/techs/go/assets/errors/
  local/practices/testing/           ->  rules/practices/testing/
  local/README.md                    ->  rules/README.md
  local/assets/retry-lifecycle.svg   ->  assets/retry-lifecycle.svg

Rule IDs don't change, so config.yaml needs no edits. A moved rule sits one folder deeper, so its links to shared assets need one more ../, such as ../../../assets/retry-lifecycle.svg. Then run syllabus project build.`

// TestProjectCommands_RejectALocalFolder refuses, writing nothing, every project command in a project whose Code
// Rules directory still has local/, and says where its files belong now, in human and JSON output.
func TestProjectCommands_RejectALocalFolder(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	if out, diagnostic, code := runCLI(t, binary, directory, "project", "init"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	for name, content := range map[string]string{
		"local/techs/go/_group.yaml": `{"name":"Go","description":"Go guidance.","whenToRead":"When writing Go."}`,
		"local/techs/go/errors.md":   "---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Keep failures visible.\nwhenToRead: When calling functions.\n---\nSee [the diagram](../../assets/retry.svg).\n",
		"local/assets/retry.svg":     "<svg/>",
	} {
		path := filepath.Join(directory, ".syllabus", name)
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
			want := "Error: " + localFolderProblem + test.outcome + "\n\nLocation: .syllabus/local/\n"
			if out, diagnostic, code := runCLI(t, binary, directory, test.args...); code != 1 || out != "" || diagnostic != want {
				t.Errorf("exit %d\n%s%s", code, out, diagnostic)
			}
			out, diagnostic, code := runCLI(t, binary, directory, append(test.args, "--json")...)
			var result response
			if err := json.Unmarshal([]byte(out), &result); err != nil || code != 1 || diagnostic != "" || result.OK || result.Error == nil {
				t.Fatalf("exit %d\n%s%s", code, out, diagnostic)
			}
			if got := *result.Error; got != (responseError{Kind: "validation", Message: ".syllabus/local/: " + localFolderProblem + test.outcome, Location: ".syllabus/local/"}) {
				t.Errorf("error %+v", got)
			}
		})
	}
	if !reflect.DeepEqual(before, projectFileContents(t, directory)) {
		t.Fatal("a command changed the project")
	}
}
