// Exercise library commands through the compiled native CLI with no runtime executables on PATH.

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLibraryCLIProcess checks terms preservation, complete authoring, and actionable unfinished-draft failure.
func TestLibraryCLIProcess(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "terms.txt"), []byte("Terms\r\n"), 0600)
	os.WriteFile(filepath.Join(dir, "body.md"), []byte("Return failures."), 0600)
	commands := [][]string{
		{"library", "init", "--directory", "library", "--spdx", "MIT", "--license-file", "terms.txt"},
		{"library", "add", "group", "techs/go", "--directory", "library", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."},
		{"library", "add", "rule", "techs/go/errors", "--directory", "library", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md"},
		{"library", "check", "--directory", "library"},
	}
	for _, args := range commands {
		out, diagnostic, code := runCLI(t, binary, dir, args...)
		if code != 0 || out == "" || diagnostic != "" {
			t.Fatal(args, code, out, diagnostic)
		}
	}
	data, _ := os.ReadFile(filepath.Join(dir, "library/LICENSE.md"))
	if string(data) != "Terms\r\n" {
		t.Fatal("terms changed")
	}
	draft := append([]string{}, commands[2][:len(commands[2])-2]...)
	draft[3] = "techs/go/draft"
	out, diagnostic, code := runCLI(t, binary, dir, draft...)
	if code != 0 {
		t.Fatal(draft, code, out, diagnostic)
	}
	out, diagnostic, code = runCLI(t, binary, dir, "library", "check", "--directory", "library")
	if code != 1 || out != "" || !strings.Contains(diagnostic, "code-rules:draft") {
		t.Fatal(code, out, diagnostic)
	}
}

// TestLibraryGuideExamples executes the shipped README examples and preserves publisher customization on repeat init.
func TestLibraryGuideExamples(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	if out, diagnostic, code := runCLI(t, binary, directory, "library", "init"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	guidePath := filepath.Join(directory, "README.md")
	guide, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatal(err)
	}
	blocks := regexp.MustCompile("(?ms)^```sh\\n(.*?)^```$").FindAllSubmatch(guide, -1)
	if len(blocks) == 0 {
		t.Fatal("no command examples in library README")
	}
	for index, block := range blocks {
		script := strings.ReplaceAll(string(block[1]), "code-rules ", "'"+strings.ReplaceAll(binary, "'", "'\"'\"'")+"' ")
		command := exec.Command("/bin/sh", "-eu", "-c", script)
		command.Dir = directory
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("README example %d failed: %v\n%s", index+1, err, output)
		}
	}
	for _, path := range []string{"techs/go/_group.json", "techs/go/README.md", "techs/go/return-errors.md"} {
		if _, err := os.Stat(filepath.Join(directory, path)); err != nil {
			t.Fatal(err)
		}
	}
	customized := append(guide, []byte("\nPublisher-specific release instructions.\n")...)
	if err := os.WriteFile(guidePath, customized, 0600); err != nil {
		t.Fatal(err)
	}
	if out, diagnostic, code := runCLI(t, binary, directory, "library", "init"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	after, err := os.ReadFile(guidePath)
	if err != nil || string(after) != string(customized) {
		t.Fatal("repeat init changed publisher README", err)
	}
}
