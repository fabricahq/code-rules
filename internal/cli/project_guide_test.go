// Exercise the agent-facing project guide through the native executable.

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/gitfixture"
)

// TestInitCreatesAgentGuide verifies init exposes the main agent entry point and a read-only freshness check.
func TestInitCreatesAgentGuide(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	out, diagnostic, code := runCLI(t, binary, directory, "init")
	if code != 0 {
		t.Fatal(out, diagnostic)
	}
	guidePath := filepath.Join(directory, ".code-rules", "README.md")
	guide, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Add a group", "Add a rule", "Add a third-party library", "init --check"} {
		if !strings.Contains(string(guide), text) {
			t.Fatalf("guide missing %q", text)
		}
	}
	out, diagnostic, code = runCLI(t, binary, directory, "init", "--check")
	if code != 0 || !strings.Contains(out, "up to date") {
		t.Fatal(code, out, diagnostic)
	}
	if err := os.WriteFile(guidePath, append(guide, []byte("\nMy custom text.\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code = runCLI(t, binary, directory, "init", "--check", "--json")
	if code != 1 || !strings.Contains(out, `"ok": false`) || diagnostic != "" {
		t.Fatal(code, out, diagnostic)
	}
	out, diagnostic, code = runCLI(t, binary, directory, "init")
	if code != 1 || !strings.Contains(diagnostic, "README.md") {
		t.Fatal(code, out, diagnostic)
	}
	after, err := os.ReadFile(guidePath)
	if err != nil || !strings.HasSuffix(string(after), "My custom text.\n") {
		t.Fatal("init overwrote custom text", err)
	}
}

// TestProjectGuideExamples executes every shell example from the generated README against the real CLI and an isolated Git publisher.
func TestProjectGuideExamples(t *testing.T) {
	binary := buildCLI(t)
	fixture, err := gitfixture.New(context.Background(), map[string][]byte{
		"rule-library.json":    []byte(`{"formatVersion":1}`),
		"techs/go/_group.json": []byte(`{"name":"Go","description":"Shared Go guidance.","whenToRead":"When writing Go."}`),
		"techs/go/shared.md":   []byte("---\ntitle: Preserve errors\nimpact: HIGH\nimpactDescription: Keep failures visible.\nwhenToRead: When calling fallible functions.\n---\nReturn errors to the caller.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, configName := range []string{"config.json", "project 'custom'.json"} {
		t.Run(configName, func(t *testing.T) {
			directory := t.TempDir()
			out, diagnostic, code := runCLI(t, binary, directory, "init", "--config", configName)
			if code != 0 {
				t.Fatal(out, diagnostic)
			}
			guide, err := os.ReadFile(filepath.Join(directory, "README.md"))
			if err != nil {
				t.Fatal(err)
			}
			blocks := regexp.MustCompile("(?ms)^```sh\\n(.*?)^```$").FindAllSubmatch(guide, -1)
			if len(blocks) == 0 {
				t.Fatal("no executable README examples")
			}
			for index, block := range blocks {
				script := strings.ReplaceAll(string(block[1]), "code-rules ", gitfixture.Quote(binary)+" ")
				script = strings.ReplaceAll(script, "https://github.com/example/engineering-rules", fixture.Repository)
				command := exec.Command("/bin/sh", "-eu", "-c", script)
				command.Dir = directory
				command.Env = fixture.Environment
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("README example %d failed: %v\n%s", index+1, err, output)
				}
			}
			for _, path := range []string{"local/techs/go/return-errors.md", "vendor/team/techs/go/shared.md", "generated/RULES.md", "generated/groups/techs/go.md"} {
				if _, err := os.Stat(filepath.Join(directory, path)); err != nil {
					t.Fatalf("README did not produce %s: %v", path, err)
				}
			}
		})
	}
}
