// Verify human error visibility through the native executable and real terminals.

package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/terminalfixture"
)

// TestHumanErrorDisplay keeps errors visible in terminals and readable in redirected output.
func TestHumanErrorDisplay(t *testing.T) {
	binary := buildCLI(t)
	for _, tc := range []struct {
		name, term, noColor string
		color               bool
	}{
		{"color", "xterm-256color", "", true},
		{"no-color", "xterm-256color", "1", false},
		{"dumb", "dumb", "", false},
		{"unknown", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERM", tc.term)
			t.Setenv("NO_COLOR", tc.noColor)
			result, err := terminalfixture.Run(context.Background(), binary, t.TempDir(), []string{"project", "add", "rule"}, nil)
			if err != nil || result.ExitCode != 2 {
				t.Fatal(err, result)
			}
			label := "Error:"
			if tc.color {
				label = "\x1b[1;31mError:\x1b[0m"
			}
			if !strings.Contains(result.Transcript, "\n"+label+" missing rule path") {
				t.Fatal(result.Transcript)
			}
			if !tc.color && strings.Contains(result.Transcript, "\x1b[") {
				t.Fatal(result.Transcript)
			}
			out, diagnostic, code := runCLI(t, binary, t.TempDir(), "project", "build")
			if code != 1 || out != "" || !strings.HasPrefix(diagnostic, "\nError: ") || strings.Contains(diagnostic, "\x1b[") {
				t.Fatal(code, out, diagnostic)
			}
		})
	}
}
