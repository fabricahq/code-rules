// Exercise status reports and their suggested repairs through the real executable.

package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestCheckReportsCurrentProblems verifies all mismatch kinds and executes the reported repair commands.
func TestCheckReportsCurrentProblems(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	for _, command := range []string{"init", "build"} {
		if out, diagnostic, code := runCLI(t, binary, directory, "project", command); code != 0 {
			t.Fatal(code, out, diagnostic)
		}
	}
	for _, name := range []string{"README.md", "generated/groups/README.md"} {
		if err := os.Remove(filepath.Join(directory, ".code-rules", name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"generated/RULES.md", "generated/unexpected.md"} {
		if err := os.WriteFile(filepath.Join(directory, ".code-rules", name), []byte("Manual content."), 0600); err != nil {
			t.Fatal(err)
		}
	}
	before := projectFileContents(t, directory)
	out, diagnostic, code := runCLI(t, binary, directory, "project", "check", "--json")
	var result struct {
		OK    bool
		Value projectCheckResult
		Error responseError
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err, out)
	}
	if code != 1 || diagnostic != "" || result.OK || result.Error.Kind != "out_of_date" || result.Value.Status != "out_of_date" {
		t.Fatal(code, out, diagnostic)
	}
	kinds := map[string]string{}
	repairs := map[string]bool{}
	for _, problem := range result.Value.Problems {
		kinds[problem.Kind] = problem.Path
		if problem.Message == "" || problem.NextStep == "" {
			t.Fatal(problem)
		}
		repairs[problem.NextStep] = true
	}
	want := map[string]string{
		"missing_file":    "generated/groups/README.md",
		"stale_contents":  "generated/RULES.md",
		"unexpected_file": "generated/unexpected.md",
		"outdated_readme": "README.md",
	}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatal(kinds, want)
	}
	var envelope struct{ Value map[string]json.RawMessage }
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"added", "changed", "removed"} {
		if _, exists := envelope.Value[key]; exists {
			t.Fatalf("check implies completed changes: %s", out)
		}
	}
	human, diagnostic, code := runCLI(t, binary, directory, "project", "check")
	if code != 1 || diagnostic != "" || !strings.Contains(human, "No files were changed.") {
		t.Fatal(code, human, diagnostic)
	}
	for _, problem := range result.Value.Problems {
		if !strings.Contains(human, problem.Message) || !strings.Contains(human, problem.NextStep) {
			t.Fatal("human and JSON reports disagree", human, problem)
		}
	}
	if !reflect.DeepEqual(before, projectFileContents(t, directory)) {
		t.Fatal("check wrote files")
	}
	for repair := range repairs {
		script := strings.Replace(repair, "code-rules", "'"+strings.ReplaceAll(binary, "'", "'\"'\"'")+"'", 1)
		command := exec.Command("/bin/sh", "-eu", "-c", script)
		command.Dir = directory
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatal(repair, err, string(output))
		}
	}
	out, diagnostic, code = runCLI(t, binary, directory, "project", "check", "--json")
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if code != 0 || diagnostic != "" || !result.OK || result.Value.Status != "up_to_date" || len(result.Value.Problems) != 0 {
		t.Fatal(code, out, diagnostic)
	}
}

// TestGuideRepairFromSubdirectory runs the reported repair where the user ran check.
func TestGuideRepairFromSubdirectory(t *testing.T) {
	binary := buildCLI(t)
	for _, state := range []string{"missing", "outdated", "edited"} {
		t.Run(state, func(t *testing.T) {
			root := t.TempDir()
			locationGit(t, root, "init", "--quiet", "--template=")
			child := filepath.Join(root, "src", "nested")
			if err := os.MkdirAll(child, 0700); err != nil {
				t.Fatal(err)
			}
			for _, action := range []string{"init", "build"} {
				if out, diagnostic, code := runCLI(t, binary, root, "project", action); code != 0 {
					t.Fatal(code, out, diagnostic)
				}
			}
			guide := filepath.Join(root, ".code-rules", "README.md")
			body := []byte("# Earlier guide\n")
			data := []byte(fmt.Sprintf("<!-- code-rules:project-guide sha256:%x -->\n%s", sha256.Sum256(body), body))
			if state == "edited" {
				data = append(data, []byte("Manual notes.\n")...)
			}
			if err := os.WriteFile(guide, data, 0600); err != nil {
				t.Fatal(err)
			}
			if state == "missing" {
				if err := os.Remove(guide); err != nil {
					t.Fatal(err)
				}
			}
			out, diagnostic, code := runCLI(t, binary, child, "project", "check", "--json")
			var result struct{ Value projectCheckResult }
			if code != 1 || diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil || len(result.Value.Problems) != 1 {
				t.Fatal(code, out, diagnostic)
			}
			problem := result.Value.Problems[0]
			if problem.Kind != "outdated_readme" {
				t.Fatal(problem)
			}
			human, diagnostic, code := runCLI(t, binary, child, "project", "check")
			if code != 1 || diagnostic != "" || !strings.Contains(human, problem.NextStep) {
				t.Fatal(code, human, diagnostic)
			}
			script := strings.Replace(problem.NextStep, "code-rules", shellDirectory(binary), 1)
			repair := func() ([]byte, error) {
				cmd := exec.Command("/bin/sh", "-eu", "-c", script)
				cmd.Dir = child
				return cmd.CombinedOutput()
			}
			if state == "edited" {
				before := projectFileContents(t, root)
				output, err := repair()
				if err == nil || !strings.Contains(string(output), "manually edited project guide") || !strings.Contains(string(output), "rerun this command") {
					t.Fatal(err, string(output))
				}
				if !reflect.DeepEqual(before, projectFileContents(t, root)) {
					t.Fatal("repair overwrote manual edits")
				}
				if err := os.Rename(guide, guide+".saved"); err != nil {
					t.Fatal(err)
				}
			}
			if output, err := repair(); err != nil {
				t.Fatal(problem.NextStep, err, string(output))
			}
			if out, diagnostic, code := runCLI(t, binary, child, "project", "check"); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
		})
	}
}
