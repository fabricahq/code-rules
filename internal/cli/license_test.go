// Verify that the standalone executable carries the exact source license without runtime files.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedLicense checks plain and JSON output from a binary without an adjacent license file.
func TestEmbeddedLicense(t *testing.T) {
	binary := buildCLI(t)
	want, err := os.ReadFile(filepath.Join("..", "..", "LICENSE.md"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, jsonMode := range []bool{false, true} {
		args := []string{"--license"}
		if jsonMode {
			args = append(args, "--json")
		}
		out, diagnostic, code := runCLI(t, binary, dir, args...)
		if code != 0 || diagnostic != "" {
			t.Fatalf("license failed: %d %s %s", code, out, diagnostic)
		}
		if jsonMode {
			var response struct {
				OK    bool `json:"ok"`
				Value struct {
					Text string `json:"text"`
				} `json:"value"`
			}
			if err := json.Unmarshal([]byte(out), &response); err != nil || !response.OK {
				t.Fatalf("invalid license response: %v %s", err, out)
			}
			out = response.Value.Text
		}
		if out != string(want) {
			t.Fatalf("embedded license differs from LICENSE.md: %q", out)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("license command modified working directory", entries, err)
	}
}
