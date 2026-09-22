// Exercise status reports and their suggested repairs through the real executable.

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestCheckReportsCurrentProblems verifies all mismatch kinds and executes the custom-config repair commands.
func TestCheckReportsCurrentProblems(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	config := "-project 'custom'.json"
	for _, command := range []string{"init", "build"} {
		if out, diagnostic, code := runCLI(t, binary, directory, "project", command, "--config="+config); code != 0 {
			t.Fatal(code, out, diagnostic)
		}
	}
	for _, name := range []string{"CODE_RULES.md", "generated/groups/README.md"} {
		if err := os.Remove(filepath.Join(directory, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"generated/RULES.md", "generated/unexpected.md"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("Manual content."), 0600); err != nil {
			t.Fatal(err)
		}
	}
	before := projectFileContents(t, directory)
	out, diagnostic, code := runCLI(t, binary, directory, "project", "check", "--config="+config, "--json")
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
		"outdated_readme": "CODE_RULES.md",
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
	human, diagnostic, code := runCLI(t, binary, directory, "project", "check", "--config="+config)
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
	out, diagnostic, code = runCLI(t, binary, directory, "project", "check", "--config="+config, "--json")
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if code != 0 || diagnostic != "" || !result.OK || result.Value.Status != "up_to_date" || len(result.Value.Problems) != 0 {
		t.Fatal(code, out, diagnostic)
	}
}
