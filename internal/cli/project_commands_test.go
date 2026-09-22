// Verify scoped help, flag handling, and usage errors through the real executable.

package cli

import (
	"encoding/json"
	"os"
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
		{[]string{"project", "add", "rule", "-h"}, []string{"code-rules project add rule RULE_PATH", "--body-file"}, nil},
		{[]string{"library"}, []string{"  init ", "  add ", "  check "}, nil},
		{[]string{"library", "add"}, []string{"  rule ", "  group "}, nil},
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
		{"project", "add", "library", "team", "--version", ">= 1.0.0", "--json"},
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
