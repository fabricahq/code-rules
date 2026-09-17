// Exercise actual terminal prompts and cancellation against the compiled CLI.

package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/terminalfixture"
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
		{"group", []terminalfixture.Step{{Prompt: "(--name):", Answer: "  Go  "}, {Prompt: "(--description):", Answer: "Go guidance."}, {Prompt: "(--when-to-read):", Answer: "When editing Go."}}, 0, true, nil},
		{"EOF", []terminalfixture.Step{{Prompt: "(--name):", EOF: true}}, 2, false, nil},
		{"interrupt", []terminalfixture.Step{{Prompt: "(--name):", Interrupt: true}}, 1, false, nil},
		{"typed-ctrl-c", []terminalfixture.Step{{Prompt: "(--name):", Answer: "\x03"}}, 1, false, nil},
		{"blank", []terminalfixture.Step{{Prompt: "(--name):", Answer: "   "}}, 2, false, nil},
		{"unattended", nil, 2, false, []string{"--non-interactive"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			if _, stderr, code := runCLI(t, binary, directory, "init"); code != 0 {
				t.Fatal(stderr)
			}
			args := append([]string{"local", "add", "group", "techs/go"}, tc.flags...)
			result, err := terminalfixture.Run(context.Background(), binary, directory, args, tc.steps)
			if err != nil {
				t.Fatal(err, result)
			}
			if result.ExitCode != tc.code {
				t.Fatal(result)
			}
			data, err := os.ReadFile(filepath.Join(directory, ".code-rules/local/techs/go/_group.json"))
			if tc.created {
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

// TestInteractiveSourceAndGroupOffer checks source choice and missing-group consent before any rule is created.
func TestInteractiveSourceAndGroupOffer(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	if _, stderr, code := runCLI(t, binary, directory, "init"); code != 0 {
		t.Fatal(stderr)
	}
	steps := []terminalfixture.Step{{Prompt: "(--repository):", Answer: "https://github.com/acme/rules"}, {Prompt: "Revision kind (ref or version):", Answer: "version"}, {Prompt: "(--version):", Answer: ">= 1.2.3"}, {Prompt: "Groups (comma-separated IDs, *, practices/*, or techs/*):", Answer: "techs/go, techs/rust"}}
	result, err := terminalfixture.Run(context.Background(), binary, directory, []string{"add", "source", "team"}, steps)
	if err != nil || result.ExitCode != 0 {
		t.Fatal(err, result)
	}
	// Use a separate local-only project so missing metadata does not require a remote snapshot.
	directory = t.TempDir()
	if _, stderr, code := runCLI(t, binary, directory, "init"); code != 0 {
		t.Fatal(stderr)
	}
	args := []string{"local", "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions."}
	result, err = terminalfixture.Run(context.Background(), binary, directory, args, []terminalfixture.Step{{Prompt: "Create missing group techs/go? [y/N]", Answer: "n"}})
	if err != nil || result.ExitCode != 2 {
		t.Fatal(err, result)
	}
	if _, err := os.Stat(filepath.Join(directory, ".code-rules/local/techs")); !os.IsNotExist(err) {
		t.Fatal("decline wrote files", err)
	}
	result, err = terminalfixture.Run(context.Background(), binary, directory, args, []terminalfixture.Step{{Prompt: "Create missing group techs/go? [y/N]", Answer: "yes"}, {Prompt: "(--group-name):", Answer: "Go"}, {Prompt: "(--group-description):", Answer: "Go guidance."}, {Prompt: "(--group-when-to-read):", Answer: "When editing Go."}})
	if err != nil || result.ExitCode != 0 {
		t.Fatal(err, result)
	}
	if _, err := os.Stat(filepath.Join(directory, ".code-rules/local/techs/go/errors.md")); err != nil {
		t.Fatal(err)
	}
}

// TestLongTerminalPaste preserves text beyond the operating system's canonical line buffer.
func TestLongTerminalPaste(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	if _, stderr, code := runCLI(t, binary, directory, "init"); code != 0 {
		t.Fatal(stderr)
	}
	description := strings.Repeat("x", 2000)
	result, err := terminalfixture.Run(context.Background(), binary, directory, []string{"local", "add", "group", "techs/go", "--name", "Go", "--when-to-read", "When editing Go."}, []terminalfixture.Step{{Prompt: "(--description):", Answer: description}})
	if err != nil || result.ExitCode != 0 {
		t.Fatal(err, result)
	}
	data, err := os.ReadFile(filepath.Join(directory, ".code-rules/local/techs/go/_group.json"))
	if err != nil || !strings.Contains(string(data), description) {
		t.Fatal("long paste was lost", err)
	}
}
