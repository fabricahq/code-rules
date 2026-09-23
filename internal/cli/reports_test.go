// Exercise shared human/JSON guidance and typed error contracts through the public CLI entry point.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func invokeReport(t *testing.T, ctx context.Context, directory string, args ...string) (string, string, int) {
	t.Helper()
	var out, diagnostic bytes.Buffer
	code := Run(ctx, args, Streams{Out: &out, Err: &diagnostic}, Options{Directory: directory})
	return out.String(), diagnostic.String(), code
}

func TestAuthoringReportSharesNextSteps(t *testing.T) {
	for _, scope := range []string{"project", "library"} {
		for _, action := range []string{"init", "group", "rule", "body-rule", "source"} {
			if scope == "library" && action == "source" {
				continue
			}
			t.Run(scope+"/"+action, func(t *testing.T) {
				results := make([]string, 2)
				for i := range results {
					directory := t.TempDir()
					libraryFlags := []string{}
					if scope == "library" {
						libraryFlags = []string{"--directory", "team's rules"}
					}
					run := func(args ...string) string {
						t.Helper()
						args = append(args, libraryFlags...)
						out, diagnostic, code := invokeReport(t, context.Background(), directory, args...)
						if code != 0 || diagnostic != "" {
							t.Fatal(code, out, diagnostic)
						}
						return out
					}
					if action != "init" {
						run(scope, "init")
					}
					if action == "rule" || action == "body-rule" {
						run(scope, "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go.")
					}
					args := []string{scope, "init"}
					switch action {
					case "group":
						args = []string{scope, "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."}
					case "rule", "body-rule":
						args = []string{scope, "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions."}
						if action == "body-rule" {
							if err := os.WriteFile(filepath.Join(directory, "body.md"), []byte("Return failures to callers.\n"), 0600); err != nil {
								t.Fatal(err)
							}
							args = append(args, "--body-file", "body.md")
						}
					case "source":
						args = []string{scope, "add", "library", "team", "--repository", "https://example.invalid/rules.git", "--ref", "v1.0.0", "--groups", "*"}
					}
					if i == 1 {
						args = append(args, "--json")
					}
					results[i] = run(args...)
				}
				var result struct {
					OK    bool
					Value authoringValue
				}
				if err := json.Unmarshal([]byte(results[1]), &result); err != nil {
					t.Fatal(err)
				}
				if !result.OK || len(result.Value.NextSteps) == 0 || result.Value.Next == "" || !strings.Contains(results[0], result.Value.Next) {
					t.Fatal(results)
				}
				for _, step := range result.Value.NextSteps {
					if !strings.Contains(results[0], step.Instruction) {
						t.Fatal("human output lost instruction", step)
					}
					for _, command := range step.Commands {
						if !strings.Contains(results[0], command) {
							t.Fatal("human output lost command", command)
						}
						if scope == "library" && !strings.Contains(command, `--directory='team'"'"'s rules'`) {
							t.Fatal("lost library directory", command)
						}
					}
				}
			})
		}
	}
}

func TestCommandErrorKindsDoNotDependOnExecutionOrder(t *testing.T) {
	directory := t.TempDir()
	for _, args := range [][]string{
		{"project", "init"},
		{"project", "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."},
	} {
		if out, diagnostic, code := invokeReport(t, context.Background(), directory, args...); code != 0 {
			t.Fatal(code, out, diagnostic)
		}
	}
	for _, test := range []struct {
		name          string
		args          []string
		exit          int
		kind, code    string
		empty, cancel bool
	}{
		{name: "unknown flag", args: []string{"project", "build", "--bad"}, exit: 2, kind: "usage"},
		{name: "missing argument", args: []string{"project", "add", "rule"}, exit: 2, kind: "usage"},
		{name: "missing input", args: []string{"project", "add", "group", "techs/python"}, exit: 2, kind: "usage"},
		{name: "missing project", args: []string{"project", "build"}, exit: 1, kind: "operation", code: "needs-init", empty: true},
		{name: "missing group", args: []string{"project", "add", "rule", "techs/python/errors"}, exit: 1, kind: "operation", code: "missing-group"},
		{name: "duplicate", args: []string{"project", "add", "group", "techs/go"}, exit: 1, kind: "operation", code: "already-exists"},
		{name: "invalid extension", args: []string{"project", "add", "rule", "techs/go/errors.md"}, exit: 2, kind: "usage", code: "invalid-rule-path"},
		{name: "missing body", args: []string{"project", "add", "rule", "techs/go/errors", "--title", "Errors", "--when-to-read", "When calling", "--impact", "HIGH", "--impact-description", "Preserve errors", "--body-file", "absent.md"}, exit: 1, kind: "operation"},
		{name: "cancelled", args: []string{"project", "build"}, exit: 1, kind: "cancelled", cancel: true},
		{name: "stale check", args: []string{"project", "check"}, exit: 1, kind: "out_of_date"},
	} {
		t.Run(test.name, func(t *testing.T) {
			location := directory
			if test.empty {
				location = t.TempDir()
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if test.cancel {
				cancel()
			}
			args := append(append([]string{}, test.args...), "--json")
			out, diagnostic, code := invokeReport(t, ctx, location, args...)
			var result response
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatal(err, out)
			}
			if code != test.exit || result.OK || diagnostic != "" || result.Error == nil || result.Error.Kind != test.kind || result.Error.Code != test.code {
				t.Fatal(code, out, diagnostic)
			}
		})
	}
}

// failedPromptSeparator models a terminal that stops accepting output after authoring completes.
type failedPromptSeparator struct{}

func (failedPromptSeparator) Write([]byte) (int, error) {
	return 0, errors.New("terminal output unavailable")
}

func TestCompletedAuthoringReportSurvivesSeparatorFailure(t *testing.T) {
	for _, structured := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", structured), func(t *testing.T) {
			directory := t.TempDir()
			output := &commandOutput{json: structured}
			root := &cobra.Command{Use: "code-rules", SilenceErrors: true, SilenceUsage: true}
			addProjectAuthoringCommands(root, Options{Directory: directory}, output)
			command, _, err := root.Find([]string{"init"})
			if err != nil {
				t.Fatal(err)
			}
			// Force only the post-prompt separator to fail, after the real initialization commits.
			flags := &authoringFlags{prompted: true}
			command.PostRunE = flags.finishPrompts
			root.SetErr(failedPromptSeparator{})
			root.SetArgs([]string{"init"})
			command, err = root.ExecuteContextC(context.Background())
			if err == nil || !strings.Contains(err.Error(), "terminal output unavailable") {
				t.Fatal("expected post-commit output failure", err)
			}
			if _, err := os.Stat(filepath.Join(directory, ".code-rules", "config.json")); err != nil {
				t.Fatal("initialization did not commit", err)
			}
			var out, diagnostic bytes.Buffer
			if code := output.finish(Streams{Out: &out, Err: &diagnostic}, command, err); code != 1 {
				t.Fatal("lost operation failure", code)
			}
			if structured {
				var result response
				if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.OK || result.Value == nil || result.Error == nil {
					t.Fatal("lost completed result or failure", out.String(), err)
				}
			} else if !strings.Contains(out.String(), "config.json") || !strings.Contains(diagnostic.String(), "terminal output unavailable") {
				t.Fatal("lost completed result or failure", out.String(), diagnostic.String())
			}
		})
	}
}
