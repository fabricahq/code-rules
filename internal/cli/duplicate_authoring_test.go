// Reject duplicate identities before collecting metadata through the real terminal and JSON interfaces.

package cli

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fabricahq/code-rules/internal/test/terminalfixture"
)

func TestDuplicateAuthoringFailsBeforePrompts(t *testing.T) {
	binary := buildCLI(t)
	for _, scenario := range []string{"project", "library", "library/custom"} {
		t.Run(scenario, func(t *testing.T) {
			scope := strings.Split(scenario, "/")[0]
			var location []string
			if scenario == "library/custom" {
				location = []string{"--directory", "shared rules"}
			}
			directory := t.TempDir()
			run := func(args ...string) {
				t.Helper()
				if out, diagnostic, code := runCLI(t, binary, directory, append(args, location...)...); code != 0 {
					t.Fatal(code, out, diagnostic)
				}
			}
			run(scope, "init")
			run(scope, "add", "group", "practices/testing", "--name", "Testing", "--description", "Tests.", "--when-to-read", "When testing.")
			run(scope, "add", "rule", "practices/testing/boundaries", "--title", "Test boundaries", "--when-to-read", "When testing.", "--impact", "HIGH", "--impact-description", "Catch bugs.")
			commands := [][]string{{scope, "add", "group", "practices/testing"}, {scope, "add", "rule", "practices/testing/boundaries"}}
			if scope == "project" {
				run("project", "add", "library", "team", "--repository", "https://github.com/example/rules", "--ref", "v1.0.0", "--groups", "*")
				commands = append(commands, []string{"project", "add", "library", "team"})
			}
			before := projectFileContents(t, directory)
			for _, command := range commands {
				t.Run(command[2], func(t *testing.T) {
					command := append(append([]string{}, command...), location...)
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					result, err := terminalfixture.Run(ctx, binary, directory, command, nil)
					if err != nil || result.ExitCode != 1 || !strings.Contains(result.Transcript, "already exists") || strings.Contains(result.Transcript, "Adding a ") || result.Stdout != "" {
						t.Errorf("duplicate must fail before introduction or prompts: %v %+v", err, result)
					}
					out, diagnostic, code := runCLI(t, binary, directory, append(append([]string{}, command...), "--json")...)
					var response struct {
						OK    bool
						Error responseError
					}
					if err := json.Unmarshal([]byte(out), &response); err != nil || code != 1 || response.OK || diagnostic != "" || !strings.Contains(response.Error.Message, "already exists") {
						t.Errorf("duplicate JSON failure: %v %d %s %s", err, code, out, diagnostic)
					}
					if !reflect.DeepEqual(before, projectFileContents(t, directory)) {
						t.Fatal("duplicate changed files")
					}
				})
			}
		})
	}
}
