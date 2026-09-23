// Exercise visible failures and recoverable input through the native executable.

package cli

import (
	"context"
	"encoding/json"
	"go.yaml.in/yaml/v4"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/terminalfixture"
)

// TestInteractiveImpactRetry retains previous answers and validates impact before asking why it matters.
func TestInteractiveImpactRetry(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	binary := buildCLI(t)
	for _, scope := range []string{"project", "library"} {
		t.Run(scope, func(t *testing.T) {
			dir := t.TempDir()
			for _, args := range [][]string{{scope, "init"}, {scope, "add", "group", "practices/testing", "--name", "Testing", "--description", "Testing guidance", "--when-to-read", "When testing"}} {
				if _, stderr, code := runCLI(t, binary, dir, args...); code != 0 {
					t.Fatal(code, stderr)
				}
			}
			impactPrompt := "Impact (CRITICAL, HIGH, MEDIUM-HIGH, MEDIUM, LOW-MEDIUM, LOW):"
			steps := []terminalfixture.Step{
				{Prompt: "Rule title:", Answer: "Keep this title"},
				{Prompt: "When to read:", Answer: "Keep this cue"},
				{Prompt: impactPrompt, Answer: "dfadf"},
				{Prompt: impactPrompt, Answer: "HIGH"},
				{Prompt: "Why it matters:", Answer: "Keep this rationale"},
			}
			result, err := terminalfixture.Run(context.Background(), binary, dir, []string{scope, "add", "rule", "practices/testing/my-rule"}, steps)
			if err != nil || result.ExitCode != 0 {
				t.Fatal(err, result)
			}
			transcript := strings.ReplaceAll(result.Transcript, "\r\n", "\n")
			if !strings.Contains(transcript, "\n\nError: impact must be one of") || strings.Count(transcript, "Rule title:") != 1 || strings.Count(transcript, "Example rule:") != 1 || strings.Index(transcript, "Error:") > strings.LastIndex(transcript, "Why it matters:") {
				t.Fatal(transcript)
			}
			root := dir
			if scope == "project" {
				root = filepath.Join(dir, ".code-rules", "local")
			}
			data, err := os.ReadFile(filepath.Join(root, "practices/testing/my-rule.md"))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Keep this title", "Keep this cue", "Keep this rationale", "HIGH"} {
				if !strings.Contains(string(data), want) {
					t.Fatal(string(data))
				}
			}
			if strings.Contains(string(data), "dfadf") {
				t.Fatal(string(data))
			}
		})
	}
}

// TestSourceAnswerRetry corrects constrained library inputs before the declaration is saved.
func TestSourceAnswerRetry(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	binary := buildCLI(t)
	for _, kind := range []string{"ref", "version"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			if _, stderr, code := runCLI(t, binary, dir, "project", "init"); code != 0 {
				t.Fatal(stderr)
			}
			revisionPrompt, invalid, valid := "Ref (tag, full commit SHA, or version range):", "bad ref", "v1.2.3"
			if kind == "version" {
				invalid, valid = ">= not-a-version", ">= 1.2.3"
			}
			groupPrompt := "Groups (comma-separated paths, *, practices/*, or techs/*):"
			steps := []terminalfixture.Step{
				{Prompt: "Git repository URL:", Answer: "acme/rules"},
				{Prompt: "Git repository URL:", Answer: "https://github.com/acme/rules"},
				{Prompt: revisionPrompt, Answer: invalid},
				{Prompt: revisionPrompt, Answer: valid},
				{Prompt: groupPrompt, Answer: "techs/go, techs/go"},
				{Prompt: groupPrompt, Answer: "techs/*"},
			}
			result, err := terminalfixture.Run(context.Background(), binary, dir, []string{"project", "add", "library", "team"}, steps)
			if err != nil || result.ExitCode != 0 || strings.Count(result.Transcript, "Error:") != 3 {
				t.Fatal(err, result)
			}
			data, err := os.ReadFile(filepath.Join(dir, ".code-rules/config.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			var config struct{ Sources map[string]map[string]any }
			if err := yaml.Unmarshal(data, &config); err != nil {
				t.Fatal(err)
			}
			source := config.Sources["team"]
			if source["repository"] != "https://github.com/acme/rules" || source[kind] != valid || source["groups"] != "techs/*" {
				t.Fatal(string(data))
			}
		})
	}
}

// TestInvalidExplicitImpact never substitutes a prompt for an invalid flag, even with a terminal.
func TestInvalidExplicitImpact(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")
	binary := buildCLI(t)
	dir := t.TempDir()
	for _, args := range [][]string{{"project", "init"}, {"project", "add", "group", "practices/testing", "--name", "Testing", "--description", "Testing guidance", "--when-to-read", "When testing"}} {
		if _, stderr, code := runCLI(t, binary, dir, args...); code != 0 {
			t.Fatal(stderr)
		}
	}
	args := []string{"project", "add", "rule", "practices/testing/my-rule", "--title", "My rule", "--when-to-read", "When testing", "--impact", "dfadf", "--impact-description", "Keep coverage"}
	for _, structured := range []bool{false, true} {
		command := append([]string(nil), args...)
		if structured {
			command = append(command, "--json")
		}
		result, err := terminalfixture.Run(context.Background(), binary, dir, command, nil)
		if err != nil || result.ExitCode != 1 || strings.Contains(result.Transcript, "Please try again.") {
			t.Fatal(err, result)
		}
		if structured {
			var response struct {
				OK    bool
				Error responseError
			}
			if result.Transcript != "" || json.Unmarshal([]byte(result.Stdout), &response) != nil || response.OK || response.Error.Kind != "validation" || response.Error.Location != "local:practices/testing/my-rule.md" || !strings.HasPrefix(response.Error.Message, response.Error.Location+": impact must") {
				t.Fatal(result)
			}
		} else if result.Stdout != "" || !strings.Contains(result.Transcript, "\x1b[1;31mError:\x1b[0m impact must") || !strings.Contains(result.Transcript, "Location: local:practices/testing/my-rule.md") {
			t.Fatal(result)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".code-rules/local/practices/testing/my-rule.md")); !os.IsNotExist(err) {
		t.Fatal("invalid flag wrote a rule", err)
	}
}
