// Require an existing group before asking for rule metadata or creating files.

package cli

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fabricahq/code-rules/internal/test/terminalfixture"
)

// TestRuleRequiresExistingGroup exercises the same refusal in unattended and real-terminal use.
func TestRuleRequiresExistingGroup(t *testing.T) {
	binary := buildCLI(t)
	for _, family := range []string{"local", "library"} {
		t.Run(family, func(t *testing.T) {
			directory := t.TempDir()
			init := []string{"init"}
			if family == "library" {
				init = []string{"library", "init"}
			}
			if out, diagnostic, code := runCLI(t, binary, directory, init...); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
			before := projectFileContents(t, directory)
			args := []string{family, "add", "rule", "techs/go/errors"}
			want := "code-rules " + family + " add group techs/go"
			out, diagnostic, code := runCLI(t, binary, directory, append(args, "--json")...)
			if code == 0 || !strings.Contains(out, want) || diagnostic != "" {
				t.Fatal(code, out, diagnostic)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			result, err := terminalfixture.Run(ctx, binary, directory, args, nil)
			if err != nil || result.ExitCode == 0 || !strings.Contains(result.Transcript, want) || strings.Contains(result.Transcript, "Action-oriented rule title:") || strings.Contains(result.Transcript, "[y/N]") {
				t.Fatal(err, result)
			}
			if !reflect.DeepEqual(before, projectFileContents(t, directory)) {
				t.Fatal("missing-group refusal wrote files")
			}
			groupArgs := []string{family, "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go."}
			if out, diagnostic, code := runCLI(t, binary, directory, groupArgs...); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
			ruleArgs := append(args, "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.")
			if out, diagnostic, code := runCLI(t, binary, directory, ruleArgs...); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
		})
	}
}
