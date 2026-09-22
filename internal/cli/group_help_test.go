// Verify group help presents real command options consistently in human and JSON modes.

package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestGroupHelpSections(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	for _, scope := range []string{"project", "library", "local"} {
		t.Run(scope, func(t *testing.T) {
			path := []string{scope, "add", "group"}
			for _, args := range [][]string{
				append(append([]string{}, path...), "--help"),
				append(append([]string{}, path...), "-h"),
				append(append([]string{}, path...), "--help", "--json"),
			} {
				out, diagnostic, code := runCLI(t, binary, directory, args...)
				if code != 0 || diagnostic != "" {
					t.Fatal(args, code, out, diagnostic)
				}
				if args[len(args)-1] == "--json" {
					var response struct {
						OK    bool
						Value struct{ Text string }
					}
					if err := json.Unmarshal([]byte(out), &response); err != nil || !response.OK {
						t.Fatal(err, out)
					}
					out = response.Value.Text
				}
				_, options, found := strings.Cut(out, "Group options:\n")
				if !found {
					t.Fatal("missing group options", out)
				}
				group, remainder, found := strings.Cut(options, "Common options:\n")
				if !found {
					t.Fatal("missing common options", out)
				}
				common, example, found := strings.Cut(remainder, "Example:\n")
				if !found || !strings.Contains(example, "code-rules "+scope+" add group practices/testing --name \"Testing\"") {
					t.Fatal("incorrect example", out)
				}
				for _, flag := range []string{"--name", "--description", "--when-to-read"} {
					if !strings.Contains(group, flag) || strings.Contains(common, flag) {
						t.Fatal("metadata in wrong section", flag, out)
					}
				}
				location, absent := "--config", "--directory"
				if scope == "library" {
					location, absent = absent, location
				}
				for _, flag := range []string{location, "--help", "--json", "--non-interactive"} {
					if strings.Count(common, flag) != 1 || strings.Contains(group, flag) {
						t.Fatal("common option in wrong section", flag, out)
					}
				}
				if strings.Contains(out, absent) || strings.Contains(out, "Global Flags:") || !strings.Contains(out, "GROUP_PATH combines a category and group slug") || !strings.Contains(group, "Title shown in rule indexes and group pages") {
					t.Fatal("incorrect scope or unclear group name", out)
				}
			}
		})
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("help changed files", entries, err)
	}
}
