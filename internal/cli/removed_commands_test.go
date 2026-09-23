// Reject retired command paths in human and JSON modes without side effects.

package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRemovedCommandsAreRejected(t *testing.T) {
	binary := buildCLI(t)
	for _, path := range []string{
		"init", "add", "add source team", "local", "local add group techs/go",
		"local add rule techs/go/errors", "build", "sync", "check", "help", "completion",
		"project add source team", "project local", "library sync", "library add source",
		"__complete", "__completeNoDesc",
	} {
		for _, suffix := range [][]string{nil, {"-h"}, {"--help"}, {"--json"}, {"--json", "--help"}} {
			args := append(strings.Fields(path), suffix...)
			t.Run(strings.Join(args, " "), func(t *testing.T) {
				directory := t.TempDir()
				out, diagnostic, code := runCLI(t, binary, directory, args...)
				if code != 2 {
					t.Fatalf("retired command must fail with usage status: code=%d stdout=%s stderr=%s", code, out, diagnostic)
				}
				if strings.Contains(strings.Join(suffix, " "), "--json") {
					var result response
					if diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil || result.Error == nil || result.Error.Kind != "usage" {
						t.Fatal("expected one JSON usage error", out, diagnostic)
					}
				} else if out != "" || !strings.Contains(diagnostic, "Error:") || !strings.Contains(diagnostic, "unknown command") {
					t.Fatal("expected an unknown-command diagnostic", out, diagnostic)
				}
				entries, err := os.ReadDir(directory)
				if err != nil || len(entries) != 0 {
					t.Fatal("retired command changed files", entries, err)
				}
			})
		}
	}
}
