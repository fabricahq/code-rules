// Exercise the YAML-only project lifecycle through the installed command surface.

package cli

import (
	"os"
	"path/filepath"
	"reflect"
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
	if err != nil || !strings.HasPrefix(string(original), "schemaVersion: 2\n") {
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
	if !strings.Contains(string(data), "    ref: v1.2.3\n") || strings.Contains(string(data), "exclude") {
		t.Fatal("unexpected source declaration", string(data))
	}
	text := "# Company rule selections\n" + strings.Replace(string(data), "      include:\n", "      exclude:\n        - id: techs/go/old\n          reason: 'Keep our local policy' # retained reason\n      include:\n", 1)
	if err := os.WriteFile(name, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	run("project", "add", "library", "security", "--repository", "https://example.invalid/security.git", "--ref", "v2.0.0", "--groups", "practices/*", "--non-interactive")
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
	if config.Sources[0].Ref.String() != "v2.0.0" || config.Sources[1].Exclude["techs/go/old"].Reason != "Keep our local policy" {
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

func TestAddingLibrariesPreservesFoldedExclusion(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if out, diagnostic, code := runCLI(t, binary, directory, args...); code != 0 {
			t.Fatal(args, code, out, diagnostic)
		}
	}
	run("project", "init")
	run("project", "add", "library", "existing", "--repository", "https://example.invalid/existing.git", "--ref", "v1.0.0", "--groups", "*", "--non-interactive")
	name := filepath.Join(directory, ".code-rules/config.yaml")
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "      include:\n", "      exclude:\n        - id: techs/go/old\n          reason: > # keep this reason\n            Heading:\n\n              * first item\n              * second item\n      include:\n", 1))
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := rules.ParseConfigurationYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	want := before.Sources[0].Exclude["techs/go/old"].Reason
	for _, alias := range []string{"second", "third"} {
		run("project", "add", "library", alias, "--repository", "https://example.invalid/"+alias+".git", "--ref", "v1.0.0", "--groups", "*", "--non-interactive")
		data, err = os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		config, err := rules.ParseConfigurationYAML(data)
		if err != nil {
			t.Fatal(err)
		}
		var got string
		for _, source := range config.Sources {
			if source.Name == "existing" {
				got = source.Exclude["techs/go/old"].Reason
			}
		}
		if got != want {
			t.Fatalf("%s changed existing exclusion from %q to %q:\n%s", alias, want, got, data)
		}
		if !strings.Contains(string(data), "# keep this reason") {
			t.Fatalf("%s lost exclusion comment:\n%s", alias, data)
		}
	}
}

func TestLegacyConfigIsNotLoaded(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	root := filepath.Join(directory, ".code-rules")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte(`{"schemaVersion":2,"sources":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code := runCLI(t, binary, directory, "project", "build")
	if code == 0 || !strings.Contains(diagnostic, "config.yaml: missing configuration") {
		t.Fatal(code, out, diagnostic)
	}
}

// TestProjectCommands_RejectAVersionOneConfiguration fails each command that reads the configuration with the
// version 2 example, writing nothing.
func TestProjectCommands_RejectAVersionOneConfiguration(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	if out, diagnostic, code := runCLI(t, binary, directory, "project", "init"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	config := "schemaVersion: 1\nsources:\n  team:\n    repository: https://example.invalid/team.git\n    groups: [techs/go]\n"
	if err := os.WriteFile(filepath.Join(directory, ".code-rules/config.yaml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	before := projectFileContents(t, directory)
	for _, args := range [][]string{{"project", "check"}, {"project", "sync"}, {"project", "build"}, {"project", "add", "library", "other", "--repository", "https://example.invalid/other.git", "--groups", "techs/go", "--non-interactive"}} {
		out, diagnostic, code := runCLI(t, binary, directory, args...)
		if code != 1 || out != "" || !strings.HasPrefix(diagnostic, "Error: version 1 configuration is no longer supported") || !strings.Contains(diagnostic, "\n\nschemaVersion: 2\nsources:\n") || !strings.Contains(diagnostic, "\nLocation: schemaVersion\n") {
			t.Errorf("%v: exit %d\n%s%s", args, code, out, diagnostic)
		}
	}
	if !reflect.DeepEqual(before, projectFileContents(t, directory)) {
		t.Fatal("a command changed the project")
	}
}
