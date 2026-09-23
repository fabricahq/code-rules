// Exercise the YAML-only project lifecycle through the installed command surface.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestYAMLProjectLifecycle(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if out, diagnostic, code := runCLI(t, binary, directory, args...); code != 0 {
			t.Fatal(args, code, out, diagnostic)
		}
	}
	run("project", "init")
	name := filepath.Join(directory, ".code-rules/config.yaml")
	original, err := os.ReadFile(name)
	if err != nil || !strings.HasPrefix(string(original), "schemaVersion: 1\n") {
		t.Fatal(string(original), err)
	}
	if _, err := os.Stat(filepath.Join(directory, ".code-rules/config.json")); !os.IsNotExist(err) {
		t.Fatal("created legacy config", err)
	}
	run("project", "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance", "--when-to-read", "When writing Go", "--non-interactive")
	body := filepath.Join(directory, "body.md")
	if err := os.WriteFile(body, []byte("Return errors to the caller.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("project", "add", "rule", "techs/go/errors", "--title", "Return errors", "--when-to-read", "When calling functions", "--impact", "HIGH", "--impact-description", "Preserve failures", "--body-file", body, "--non-interactive")
	run("project", "build")
	run("project", "check")
	run("project", "add", "library", "team", "--repository", "https://example.invalid/team.git", "--ref", "v1.2.3", "--groups", "*", "--non-interactive")
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	// Preserve a user's comments, quoted scalar, and exception policy when another library is added.
	text := "# Company rule selections\n" + strings.Replace(string(data), "exclude: {}", "exclude:\n      techs/go/old: 'Keep our local policy' # retained reason", 1)
	if err := os.WriteFile(name, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	run("project", "add", "library", "security", "--repository", "https://example.invalid/security.git", "--ref", ">= 1.0.0, < 2.0.0", "--groups", "practices/*", "--non-interactive")
	data, err = os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	config, err := rules.ParseConfigurationYAML(data)
	if err != nil || len(config.Sources) != 2 {
		t.Fatal(string(data), err)
	}
	for _, marker := range []string{"# Company rule selections", "'Keep our local policy'", "# retained reason"} {
		if !strings.Contains(string(data), marker) {
			t.Fatal("lost authored YAML", marker, string(data))
		}
	}
	if config.Sources[0].Version != ">= 1.0.0, < 2.0.0" || config.Sources[1].Exclude["techs/go/old"] != "Keep our local policy" {
		t.Fatal(config)
	}
	before := projectFileContents(t, directory)
	out, diagnostic, code := runCLI(t, binary, directory, "project", "add", "library", "team", "--non-interactive")
	if code == 0 || !strings.Contains(diagnostic, "already exists") {
		t.Fatal(code, out, diagnostic)
	}
	after := projectFileContents(t, directory)
	if len(before) != len(after) {
		t.Fatal("duplicate changed files")
	}
	for name, value := range before {
		if after[name] != value {
			t.Fatal("duplicate changed", name)
		}
	}
	run("project", "init")
	unchanged, err := os.ReadFile(name)
	if err != nil || string(unchanged) != string(data) {
		t.Fatal("init rewrote authored YAML", err)
	}
}

func TestLegacyConfigIsNotLoaded(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	root := filepath.Join(directory, ".code-rules")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte(`{"schemaVersion":1,"sources":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code := runCLI(t, binary, directory, "project", "build")
	if code == 0 || !strings.Contains(diagnostic, "config.yaml: missing configuration") {
		t.Fatal(code, out, diagnostic)
	}
}
