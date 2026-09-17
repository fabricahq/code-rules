// Verify CLI walkthrough arguments cannot select an unvalidated project configuration.

package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIConfigConfinement allows the default configuration but rejects alternate files.
func TestCLIConfigConfinement(t *testing.T) {
	for _, args := range [][]string{{"sync"}, {"build", "--help=false"}, {"check", "--config=.code-rules/config.json"}} {
		if err := validateCLIArguments(args); err != nil {
			t.Fatal(args, err)
		}
	}
	for _, args := range [][]string{{"sync", "--config", "config.json"}, {"sync", "--help", "--config", "/tmp/outside.json"}, {"check", "--config=../config.json"}} {
		if err := validateCLIArguments(args); err == nil {
			t.Fatal("confinement bypass", args)
		}
	}
}

// TestCleanCLICheckFixture runs the actual adapter and CLI so fixture setup includes the managed README.
func TestCleanCLICheckFixture(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"rules-lab", "code-rules"} {
		if output, err := exec.Command("go", "build", "-o", filepath.Join(directory, name), "../"+name).CombinedOutput(); err != nil {
			t.Fatal(err, string(output))
		}
	}
	for _, edited := range []bool{false, true} {
		changes := map[string]string{}
		if edited {
			changes[".code-rules/README.md"] = "Manually changed guide."
		}
		input, err := json.Marshal(map[string]any{
			"operation": "cliProject", "location": "project", "input": map[string]any{
				"configuration": map[string]any{"schemaVersion": 1, "sources": map[string]any{}},
				"libraries":     map[string]any{}, "seedSync": true, "changes": changes,
				"arguments": []string{"check", "--json"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command(filepath.Join(directory, "rules-lab"))
		command.Stdin = strings.NewReader(string(input) + "\n")
		output, err := command.Output()
		if err != nil {
			t.Fatal(err)
		}
		var result struct {
			OK    bool
			Value cliObservation
		}
		if err := json.Unmarshal(output, &result); err != nil {
			t.Fatal(err, string(output))
		}
		wantCode := 0
		if edited {
			wantCode = 1
		}
		if !result.OK || result.Value.ExitCode != wantCode || result.Value.Before[".code-rules/README.md"] == "" {
			t.Fatal(string(output))
		}
		if edited && result.Value.Before[".code-rules/README.md"] != changes[".code-rules/README.md"] {
			t.Fatal("fixture replaced intentional README edit")
		}
		if result.Value.Before[".code-rules/README.md"] != result.Value.After[".code-rules/README.md"] {
			t.Fatal("CLI check changed README")
		}
	}
	input := `{"operation":"cliProject","location":"cli","input":{"configuration":{"schemaVersion":1,"sources":{}},"libraries":{},"arguments":["sync"],"changes":{".code-rules/config.json":"{\"schemaVersion\":1,\"sources\":{\"external\":{\"repository\":\"git@fixture.invalid:external\",\"ref\":\"v1.0.0\",\"groups\":\"*\",\"exclude\":{},\"replace\":{}}}}"}}}`
	command := exec.Command(filepath.Join(directory, "rules-lab"))
	command.Stdin = strings.NewReader(input + "\n")
	output, err := command.Output()
	if err != nil || !strings.Contains(string(output), "supplied local Git fixture") {
		t.Fatal(err, string(output))
	}

}
