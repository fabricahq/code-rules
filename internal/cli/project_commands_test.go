// Verify scoped help, flag handling, and compatibility through the real executable.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestScopedCommandHelp(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	for _, test := range []struct {
		args         []string
		want, absent []string
	}{
		{nil, []string{"  project ", "  library "}, []string{"\n  help ", "  init ", "  add ", "  local ", "  build ", "  sync ", "  check "}},
		{[]string{"project"}, []string{"Main commands:", "Utility commands:", "  init ", "  add ", "  sync ", "  build ", "  check "}, []string{"  local ", "Available Commands:", "Additional Commands:"}},
		{[]string{"project", "--help"}, []string{"Main commands:", "Utility commands:"}, []string{"Available Commands:"}},
		{[]string{"project", "add"}, []string{"  rule ", "  group ", "  library "}, []string{"  source ", "  local "}},
		{[]string{"project", "add", "library", "--help"}, []string{"without fetching", "code-rules project sync", "--repository", "--groups"}, nil},
		{[]string{"project", "check", "--help"}, []string{"not whether application code follows", "--config"}, nil},
		{[]string{"project", "add", "rule", "-h"}, []string{"code-rules project add rule ID", "--body-file"}, nil},
		{[]string{"library"}, []string{"  init ", "  add ", "  check "}, nil},
		{[]string{"local", "add"}, []string{"  rule ", "  group "}, nil},
		{[]string{"library", "add"}, []string{"  rule ", "  group "}, nil},
		{[]string{"local", "add", "rule", "--help"}, []string{"code-rules local add rule ID", "--body-file"}, nil},
		{[]string{"add", "source", "--help"}, []string{"code-rules add source ALIAS", "--repository"}, nil},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			out, diagnostic, code := runCLI(t, binary, directory, test.args...)
			if code != 0 || diagnostic != "" {
				t.Fatal(code, out, diagnostic)
			}
			lastCommand := -1
			for _, want := range test.want {
				if !strings.Contains(out, want) {
					t.Fatalf("missing %q in %s", want, out)
				}

				if strings.HasPrefix(want, "  ") {
					position := strings.Index(out, "\n"+want)
					if position <= lastCommand {
						t.Fatalf("command %q is out of workflow order in %s", want, out)
					}
					lastCommand = position
				}
			}
			for _, absent := range test.absent {
				if strings.Contains(out, absent) {
					t.Fatalf("unexpected %q in %s", absent, out)
				}
			}
		})
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("help wrote files", entries, err)
	}
}

// TestProjectCommandCompatibility proves both spellings produce the same project and JSON results.
func TestProjectCommandCompatibility(t *testing.T) {
	binary := buildCLI(t)
	canonical, legacy := t.TempDir(), t.TempDir()
	for _, directory := range []string{canonical, legacy} {
		if err := os.WriteFile(filepath.Join(directory, "body.md"), []byte("Return errors to the caller.\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, step := range []struct {
		canonical, legacy []string
		code              int
	}{
		{[]string{"project", "init"}, []string{"init"}, 0},
		{[]string{"project", "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."}, []string{"local", "add", "group"}, 0},
		{[]string{"project", "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md"}, []string{"local", "add", "rule"}, 0},
		{[]string{"project", "check"}, []string{"check"}, 1},
		{[]string{"project", "build"}, []string{"build"}, 0},
		{[]string{"project", "check"}, []string{"check"}, 0},
		{[]string{"project", "sync"}, []string{"sync"}, 0},
		{[]string{"project", "add", "library", "team", "--repository", "https://example.invalid/rules.git", "--ref", "v1.0.0", "--groups", "techs/go"}, []string{"add", "source"}, 0},
	} {
		old := append([]string{}, step.legacy...)
		if step.canonical[1] == "add" {
			old = append(old, step.canonical[3:]...)
		}
		args := append(append([]string{}, step.canonical...), "--json")
		a, diagnostic, code := runCLI(t, binary, canonical, args...)
		if code != step.code || diagnostic != "" || !json.Valid([]byte(a)) {
			t.Fatal(args, code, a, diagnostic)
		}
		b, diagnostic, code := runCLI(t, binary, legacy, append(old, "--json")...)
		if code != step.code || diagnostic != "" || a != strings.ReplaceAll(b, legacy, canonical) {
			t.Fatal(old, code, a, b, diagnostic)
		}
		legacyFiles := map[string]string{}
		for path, content := range projectFileContents(t, legacy) {
			legacyFiles[canonical+strings.TrimPrefix(path, legacy)] = content
		}
		if !reflect.DeepEqual(projectFileContents(t, canonical), legacyFiles) {
			t.Fatal("command spellings produced different files", args)
		}
	}
	if _, err := os.Stat(filepath.Join(canonical, ".code-rules/vendor/team")); !os.IsNotExist(err) {
		t.Fatal("add library fetched a snapshot", err)
	}
}

func TestScopedUsageErrors(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	for _, args := range [][]string{
		{"project", "build", "--config", "--json"},
		{"--json", "project", "build", "--config", "one", "--config", "two"},
		{"project", "--json", "add", "library", "team"},
		{"project", "add", "group", "techs/go", "--json"},
		{"project", "check", "unexpected", "--json"},
		{"project", "build", "--bad", "--json"},
	} {
		out, diagnostic, code := runCLI(t, binary, directory, args...)
		var result response
		if code != 2 || diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil || result.Error == nil || result.Error.Kind != "usage" {
			t.Fatal(args, code, out, diagnostic)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("usage errors wrote files", entries, err)
	}
}
