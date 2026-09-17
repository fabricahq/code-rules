// Require an existing group before asking for rule metadata or creating files.

package cli

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fabricahq/code-rules/internal/terminalfixture"
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
			out, diagnostic, code := runCLI(t, binary, directory, args...)
			if code == 0 || !strings.Contains(diagnostic, want) || out != "" {
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

// projectFileContents captures file bytes to detect writes by refused commands.
func projectFileContents(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
