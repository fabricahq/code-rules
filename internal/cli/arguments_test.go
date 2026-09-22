// Exercise actionable argument errors through the executable, preserving usage status and output modes.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestAuthoringArgumentErrors(t *testing.T) {
	binary := buildCLI(t)
	for _, test := range []struct{ command, name, example string }{
		{"project add library", "library alias", "team"},
		{"project add group", "group ID", "practices/testing"},
		{"project add rule", "rule ID", "practices/testing/my-rule"},
		{"library add group", "group ID", "practices/testing"},
		{"library add rule", "rule ID", "practices/testing/my-rule"},
		{"add source", "library alias", "team"},
		{"local add group", "group ID", "practices/testing"},
		{"local add rule", "rule ID", "practices/testing/my-rule"},
	} {
		for _, extra := range []bool{false, true} {
			for _, structured := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/extra=%t/json=%t", test.command, extra, structured), func(t *testing.T) {
					dir := t.TempDir()
					args := strings.Fields(test.command)
					if extra {
						args = append(args, test.example, "extra")
					}
					if structured {
						args = append(args, "--json")
					}
					out, diagnostic, code := runCLI(t, binary, dir, args...)
					if code != 2 {
						t.Fatalf("exit=%d stdout=%s stderr=%s", code, out, diagnostic)
					}
					message := diagnostic
					if structured {
						var result response
						if diagnostic != "" || json.Unmarshal([]byte(out), &result) != nil || result.OK || result.Error == nil || result.Error.Kind != "usage" {
							t.Fatalf("invalid JSON usage response: %s %s", out, diagnostic)
						}
						message = result.Error.Message
					} else if out != "" {
						t.Fatalf("unexpected stdout: %s", out)
					}
					want := "missing " + test.name + "."
					if extra {
						want = "expected one " + test.name + "; received 2 arguments."
					}
					if !strings.Contains(message, want) || !strings.Contains(message, "Usage:\n  code-rules "+test.command+" ") {
						t.Fatalf("unhelpful argument error: %s", message)
					}
					if !extra && !strings.Contains(message, "Example:\n  code-rules "+test.command+" "+test.example) {
						t.Fatalf("missing example: %s", message)
					}
					if !structured && !strings.Contains(diagnostic, "Run code-rules "+test.command+" --help") {
						t.Fatalf("wrong help destination: %s", diagnostic)
					}
					entries, err := os.ReadDir(dir)
					if err != nil || len(entries) != 0 {
						t.Fatalf("usage error wrote files: %v %v", entries, err)
					}
				})
			}
		}
	}
}
