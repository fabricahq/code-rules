// Verify CLI walkthrough arguments cannot select an unvalidated project configuration.

package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIRejectsDefaultConfig prevents fixture edits from redirecting the child CLI to external repositories.
func TestCLIRejectsDefaultConfig(t *testing.T) {
	input := json.RawMessage(`{"configuration":{"schemaVersion":1,"sources":{}},"libraries":{},"arguments":["sync"],"changes":{".code-rules/config.json":"{\"schemaVersion\":1,\"sources\":{\"external\":{\"repository\":\"git@fixture.invalid:external\",\"ref\":\"v1.0.0\",\"groups\":\"*\"}}}"}}`)
	result, err := cliResponse(input)
	if err != nil || result.OK || result.Error == nil || result.Error.Location != "arguments" {
		t.Fatalf("expected argument rejection before process setup: %+v, %v", result, err)
	}
}

// TestSubcommandHelpNeedsNoConfig preserves help-only invocations without allowing help=false to bypass confinement.
func TestSubcommandHelpNeedsNoConfig(t *testing.T) {
	for _, args := range [][]string{{"sync", "--help"}, {"build", "-h"}, {"check", "--help=true"}} {
		if err := validateCLIArguments(args); err != nil {
			t.Fatal(args, err)
		}
	}
	for _, args := range [][]string{{"sync", "--help=false"}, {"sync", "--help", "--help=false"}, {"sync", "--", "--help"}, {"sync", "--help", "--config", "/tmp/outside.json"}} {
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
			changes["README.md"] = "Manually changed guide."
		}
		input, err := json.Marshal(map[string]any{
			"operation": "cliProject", "location": "project", "input": map[string]any{
				"configuration": map[string]any{"schemaVersion": 1, "sources": map[string]any{}},
				"libraries":     map[string]any{}, "seedSync": true, "changes": changes,
				"arguments": []string{"check", "--config", "config.json", "--json"},
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
		if !result.OK || result.Value.ExitCode != wantCode || result.Value.Before["README.md"] == "" {
			t.Fatal(string(output))
		}
		if edited && result.Value.Before["README.md"] != changes["README.md"] {
			t.Fatal("fixture replaced intentional README edit")
		}
		if result.Value.Before["README.md"] != result.Value.After["README.md"] {
			t.Fatal("CLI check changed README")
		}
	}
}
