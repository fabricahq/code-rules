// Exercise actual terminal prompts and cancellation against the compiled CLI.

package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/terminalfixture"
)

// TestInteractiveAuthoring uses a real PTY and verifies prompt results, refusal, EOF, and cancellation without partial writes.
func TestInteractiveAuthoring(t *testing.T) {
	binary := buildCLI(t)
	for _, tc := range []struct {
		name    string
		steps   []terminalfixture.Step
		code    int
		created bool
		flags   []string
	}{
		{"group", []terminalfixture.Step{{Prompt: "Group name:", Answer: "  Go  "}, {Prompt: "Group description:", Answer: "Go guidance."}, {Prompt: "When to read:", Answer: "When editing Go."}}, 0, true, nil},
		{"EOF", []terminalfixture.Step{{Prompt: "Group name:", EOF: true}}, 2, false, nil},
		{"interrupt", []terminalfixture.Step{{Prompt: "Group name:", Interrupt: true}}, 1, false, nil},
		{"typed-ctrl-c", []terminalfixture.Step{{Prompt: "Group name:", Answer: "\x03"}}, 1, false, nil},
		{"blank-retry", []terminalfixture.Step{{Prompt: "Group name:", Answer: "   "}, {Prompt: "Group name:", Answer: "Go"}, {Prompt: "Group description:", Answer: "Go guidance."}, {Prompt: "When to read:", Answer: "When editing Go."}}, 0, true, nil},
		{"blank-EOF", []terminalfixture.Step{{Prompt: "Group name:", Answer: "   "}, {Prompt: "Group name:", EOF: true}}, 2, false, nil},
		{"unattended", nil, 2, false, []string{"--non-interactive"}},
		{"json", nil, 2, false, []string{"--json"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			if _, stderr, code := runCLI(t, binary, directory, "project", "init"); code != 0 {
				t.Fatal(stderr)
			}
			args := append([]string{"project", "add", "group", "techs/go"}, tc.flags...)
			result, err := terminalfixture.Run(context.Background(), binary, directory, args, tc.steps)
			if err != nil {
				t.Fatal(err, result)
			}
			if len(tc.flags) > 0 && (strings.Contains(result.Transcript, "Example group:") || strings.Contains(result.Stdout, "Example group:")) {
				t.Fatal("unattended mode showed interactive introduction", result)
			}
			if tc.name == "group" {
				for _, text := range []string{"Adding a group at: techs/go", "Example group:", "Name: Testing", "Description: Unit and integration testing.", "When to read: When writing or changing tests."} {
					if !strings.Contains(result.Transcript, text) {
						t.Fatalf("missing orientation %q: %s", text, result.Transcript)
					}
				}
				if strings.Index(result.Transcript, "Example group:") > strings.Index(result.Transcript, "Group name:") {
					t.Fatal("example shown after prompts", result.Transcript)
				}
			}
			if result.ExitCode != tc.code {
				t.Fatal(result)
			}
			data, err := os.ReadFile(filepath.Join(directory, ".code-rules/local/techs/go/_group.json"))
			if tc.created {
				if !strings.HasSuffix(strings.ReplaceAll(result.Transcript, "\r\n", "\n"), "\n\n") {
					t.Fatal("missing blank line after prompts", result.Transcript)
				}
				if err != nil || !strings.Contains(string(data), `"name": "Go"`) {
					t.Fatal(string(data), err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("unexpected file", err)
			}
			if _, err := os.Stat(filepath.Join(directory, ".code-rules/.code-rules.lock")); err == nil {
				t.Fatal("prompt left lock")
			}
		})
	}
}

// TestInteractiveSource checks repository, revision, and group selection through a real terminal.
func TestInteractiveSource(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	if _, stderr, code := runCLI(t, binary, directory, "project", "init"); code != 0 {
		t.Fatal(stderr)
	}
	steps := []terminalfixture.Step{{Prompt: "Git repository URL:", Answer: "https://github.com/acme/rules"}, {Prompt: "Ref (tag, full commit SHA, or version range):", Answer: ">= 1.2.3"}, {Prompt: "Groups (comma-separated paths, *, practices/*, or techs/*):", Answer: "techs/go, techs/rust"}}
	result, err := terminalfixture.Run(context.Background(), binary, directory, []string{"project", "add", "library", "team"}, steps)
	if err != nil || result.ExitCode != 0 {
		t.Fatal(err, result)
	}

}

// TestLongTerminalPaste preserves text beyond the operating system's canonical line buffer.
func TestLongTerminalPaste(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	if _, stderr, code := runCLI(t, binary, directory, "project", "init"); code != 0 {
		t.Fatal(stderr)
	}
	description := strings.Repeat("x", 2000)
	result, err := terminalfixture.Run(context.Background(), binary, directory, []string{"project", "add", "group", "techs/go", "--name", "Go", "--when-to-read", "When editing Go."}, []terminalfixture.Step{{Prompt: "Group description:", Answer: description}})
	if err != nil || result.ExitCode != 0 {
		t.Fatal(err, result)
	}
	data, err := os.ReadFile(filepath.Join(directory, ".code-rules/local/techs/go/_group.json"))
	if err != nil || !strings.Contains(string(data), description) {
		t.Fatal("long paste was lost", err)
	}
}

// TestInteractiveRuleRecoversDeadWriter checks advisory group lookup does not block authoritative writer recovery.
func TestInteractiveRuleRecoversDeadWriter(t *testing.T) {
	binary := buildCLI(t)
	for _, kind := range []string{"project", "library"} {
		t.Run(kind, func(t *testing.T) {
			directory := t.TempDir()
			initArgs := []string{"project", "init"}
			root := filepath.Join(directory, ".code-rules")
			if kind == "library" {
				initArgs = []string{"library", "init"}
				root = directory
			}
			if _, stderr, code := runCLI(t, binary, directory, initArgs...); code != 0 {
				t.Fatal(stderr)
			}
			if _, stderr, code := runCLI(t, binary, directory, kind, "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."); code != 0 {
				t.Fatal(stderr)
			}
			process := exec.Command("/usr/bin/true")
			if err := process.Run(); err != nil {
				t.Fatal(err)
			}
			host, err := os.Hostname()
			if err != nil {
				t.Fatal(err)
			}
			lock := filepath.Join(root, ".code-rules-lock")
			if err := os.Mkdir(lock, 0700); err != nil {
				t.Fatal(err)
			}
			owner, _ := json.Marshal(map[string]any{"host": host, "pid": process.Process.Pid, "token": "finished-test-process"})
			if err := os.WriteFile(filepath.Join(lock, "owner.json"), owner, 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{kind, "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions."}
			result, err := terminalfixture.Run(context.Background(), binary, directory, args, nil)
			if err != nil || result.ExitCode != 0 {
				t.Fatal(err, result)
			}
			if _, err := os.Stat(lock); !os.IsNotExist(err) {
				t.Fatal("lock not recovered", err)
			}
			ruleRoot := root
			if kind == "project" {
				ruleRoot = filepath.Join(root, "local")
			}
			if _, err := os.Stat(filepath.Join(ruleRoot, "techs/go/errors.md")); err != nil {
				t.Fatal(err)
			}
		})
	}
}
