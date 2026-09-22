// Follow interactive rule authoring through editing the draft and executing the displayed completion commands.

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/terminalfixture"
)

func TestRuleAuthoringGuidance(t *testing.T) {
	binary := buildCLI(t)
	for _, scope := range []string{"project", "library"} {
		for _, custom := range []bool{false, true} {
			name := scope
			if custom {
				name += "/custom"
			}
			t.Run(name, func(t *testing.T) {
				directory := t.TempDir()
				var location []string
				if custom {
					if scope == "project" {
						location = []string{"--config", "settings/team 'rules'.json"}
					} else {
						location = []string{"--directory", "shared 'rules'"}
					}
				}
				run := func(args ...string) string {
					t.Helper()
					out, diagnostic, code := runCLI(t, binary, directory, append(args, location...)...)
					if code != 0 || diagnostic != "" {
						t.Fatal(code, out, diagnostic)
					}
					return out
				}
				run(scope, "init")
				run(scope, "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When writing Go.")
				steps := []terminalfixture.Step{
					{Prompt: "Rule title:", Answer: "Return errors to the caller"},
					{Prompt: "When to read:", Answer: "When calling fallible operations."},
					{Prompt: "Impact (CRITICAL, HIGH, MEDIUM-HIGH, MEDIUM, LOW-MEDIUM, LOW):", Answer: "HIGH"},
					{Prompt: "Why it matters:", Answer: "Keep failures visible."},
				}
				args := append([]string{scope, "add", "rule", "techs/go/return-errors"}, location...)
				result, err := terminalfixture.Run(context.Background(), binary, directory, args, steps)
				if err != nil || result.ExitCode != 0 {
					t.Fatal(err, result)
				}
				for _, text := range []string{"Adding a rule at: techs/go/return-errors", "Example rule:", "Title: Test boundary conditions", "Rule text:", "Then edit the created Markdown file"} {
					if !strings.Contains(result.Transcript, text) {
						t.Fatalf("missing orientation %q: %s", text, result.Transcript)
					}
				}
				if strings.Count(result.Transcript, "Example rule:") != 1 || strings.Index(result.Transcript, "Example rule:") > strings.Index(result.Transcript, "Rule title:") {
					t.Fatal("example must precede prompts once", result.Transcript)
				}
				root := filepath.Join(directory, ".code-rules", "local")
				if scope == "library" {
					root = directory
				}
				if custom {
					if scope == "project" {
						root = filepath.Join(directory, "settings", "local")
					} else {
						root = filepath.Join(directory, "shared 'rules'")
					}
				}
				file := filepath.Join(root, "techs/go/return-errors.md")
				data, err := os.ReadFile(file)
				if err != nil || !strings.Contains(string(data), "title: Return errors to the caller") || strings.Contains(string(data), "Test boundary conditions") {
					t.Fatal("example replaced author input", err, string(data))
				}
				for _, text := range []string{"Rule draft created:", file, "Open the Markdown file above in your editor", "correct and incorrect examples", "Replace <...> placeholders", "When the rule is ready, run:"} {
					if !strings.Contains(result.Stdout, text) {
						t.Fatalf("missing completion step %q: %s", text, result.Stdout)
					}
				}
				if scope == "library" && !strings.Contains(result.Stdout, "Remove the <!-- code-rules:draft --> marker") {
					t.Fatal(result.Stdout)
				}
				header, _, found := strings.Cut(string(data), "\n---\n")
				if !found {
					t.Fatal("missing metadata", string(data))
				}
				if err := os.WriteFile(file, []byte(header+"\n---\n\n## Return errors to the caller\n\nReturn a descriptive error when an operation fails.\n"), 0600); err != nil {
					t.Fatal(err)
				}
				commands := 0
				for _, line := range strings.Split(result.Stdout, "\n") {
					command := strings.TrimSpace(line)
					if !strings.HasPrefix(command, "code-rules ") {
						continue
					}
					command = strings.Replace(command, "code-rules", "'"+strings.ReplaceAll(binary, "'", "'\"'\"'")+"'", 1)
					child := exec.Command("/bin/sh", "-eu", "-c", command)
					child.Dir = directory
					if output, err := child.CombinedOutput(); err != nil {
						t.Fatal(command, err, string(output))
					}
					commands++
				}
				expected := 2
				if scope == "library" {
					expected = 1
				}
				if commands != expected {
					t.Fatal("missing completion commands", result.Stdout)
				}
				// Supplying existing text creates a rule without draft-only completion instructions.
				bodyFile := filepath.Join(directory, "body.md")
				if err := os.WriteFile(bodyFile, []byte("Preserve error context.\n"), 0600); err != nil {
					t.Fatal(err)
				}
				out := run(scope, "add", "rule", "techs/go/context", "--title", "Preserve context", "--when-to-read", "When returning errors.", "--impact", "HIGH", "--impact-description", "Keep failures actionable.", "--body-file", bodyFile)
				if !strings.Contains(out, "Rule created from --body-file:") || strings.Contains(out, "Replace <...>") || strings.Contains(out, "Example rule:") {
					t.Fatal("wrong supplied-body guidance", out)
				}
			})
		}
	}
}
