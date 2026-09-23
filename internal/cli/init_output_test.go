// Exercise initialization guidance and repeat setup through the real executable.

package cli

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestProjectInitGuidance(t *testing.T) {
	binary := buildCLI(t)
	t.Run("project-root", func(t *testing.T) {
		directory := t.TempDir()
		args := []string{"project", "init"}
		out, diagnostic, code := runCLI(t, binary, directory, args...)
		if code != 0 || diagnostic != "" || !strings.HasPrefix(out, "Code Rules initialized!\n") {
			t.Fatal(code, out, diagnostic)
		}
		if strings.Contains(out, "Updated files:") || strings.Contains(out, directory) {
			t.Fatal("init led with internal file changes", out)
		}
		if !strings.Contains(out, "Or use a shared library") || !strings.Contains(out, "code-rules project sync") {
			t.Fatal("missing shared-library path", out)
		}
		// Execute the displayed project-only examples, supplying metadata to avoid terminal prompts.
		steps := 0
		for _, line := range strings.Split(out, "\n") {
			command := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(command, "code-rules project add group "):
				command += " --name Testing --description 'Testing practices.' --when-to-read 'When writing tests.'"
			case strings.HasPrefix(command, "code-rules project add rule "):
				command += " --title 'Test boundaries' --impact HIGH --impact-description 'Catch edge cases.' --when-to-read 'When writing tests.'"
			default:
				continue
			}
			command = strings.Replace(command, "code-rules", "'"+strings.ReplaceAll(binary, "'", "'\"'\"'")+"'", 1)
			child := exec.Command("/bin/sh", "-eu", "-c", command)
			child.Dir = directory
			if output, err := child.CombinedOutput(); err != nil {
				t.Fatal(command, err, string(output))
			}
			steps++
		}
		if steps != 2 {
			t.Fatal("missing project-only examples", out)
		}
		out, diagnostic, code = runCLI(t, binary, directory, args...)
		if code != 0 || diagnostic != "" || !strings.HasPrefix(out, "Code Rules is already initialized.\nNo files changed.\n") || strings.Contains(out, "my-rule") {
			t.Fatal("repeat init should report existing setup", code, out, diagnostic)
		}
		location := ".code-rules/config.yaml"
		if !strings.Contains(out, "Project configuration: "+location+"\n") || !strings.Contains(out, "Code Rules directory: "+".code-rules"+"\n") {
			t.Fatal("repeat init should show the configuration path", out)
		}
		out, diagnostic, code = runCLI(t, binary, directory, append(args, "--json")...)
		var result struct {
			OK    bool
			Value map[string]json.RawMessage
		}
		if code != 0 || diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil || !result.OK || len(result.Value) != 3 || result.Value["files"] == nil || result.Value["next"] == nil {
			t.Fatal("JSON contract changed", code, out, diagnostic)
		}
	})
}
