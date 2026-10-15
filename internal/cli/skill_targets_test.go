// Verify project init writes, keeps, and replaces the configuration's skill targets, and sync installs a source's
// skills into them.

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

func TestInitSetsSkillTargets(t *testing.T) {
	binary := buildCLI(t)
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"default", nil, "skillTargets: [claude, agents]\n"},
		{"one target", []string{"--skill-targets", "claude"}, "skillTargets: [claude]\n"},
		{"none", []string{"--skill-targets", "none"}, "skillTargets: []\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			if out, diagnostic, code := runCLI(t, binary, directory, append([]string{"project", "init"}, test.args...)...); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
			data, err := os.ReadFile(filepath.Join(directory, ".code-rules/config.yaml"))
			if err != nil || string(data) != "schemaVersion: 2\n"+test.want+"sources: {}\n" {
				t.Fatalf("config.yaml:\n%s %v", data, err)
			}
		})
	}
}

func TestInitReplacesSkillTargetsOnlyWhenAsked(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	config := filepath.Join(directory, ".code-rules/config.yaml")
	for _, step := range []struct {
		args []string
		want string
	}{
		{nil, "skillTargets: [claude, agents]"},
		{[]string{"--skill-targets", "agents"}, "skillTargets: [agents]"},
		{nil, "skillTargets: [agents]"},
		{[]string{"--skill-targets", "none"}, "skillTargets: []"},
	} {
		if out, diagnostic, code := runCLI(t, binary, directory, append([]string{"project", "init"}, step.args...)...); code != 0 {
			t.Fatal(code, out, diagnostic)
		}
		data, err := os.ReadFile(config)
		if err != nil || !strings.Contains(string(data), step.want+"\n") {
			t.Fatalf("after init %v, config.yaml:\n%s %v", step.args, data, err)
		}
	}
}

func TestInitRejectsInvalidSkillTargets(t *testing.T) {
	binary := buildCLI(t)
	for name, args := range map[string][]string{
		"unknown":         {"--skill-targets", "cursor"},
		"none and claude": {"--skill-targets", "none", "--skill-targets", "claude"},
		"repeated":        {"--skill-targets", "claude", "--skill-targets", "claude"},
	} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			out, diagnostic, code := runCLI(t, binary, directory, append([]string{"project", "init"}, args...)...)
			if code != 2 || !strings.Contains(diagnostic, "--skill-targets") {
				t.Fatal(code, out, diagnostic)
			}
			if entries, err := os.ReadDir(directory); err != nil || len(entries) != 0 {
				t.Fatal("rejected init wrote files", entries, err)
			}
		})
	}
}

func TestSync_InstallsTheSkillsOfAnAddedSource(t *testing.T) {
	binary := buildCLI(t)
	f, err := gitfixture.New(context.Background(), map[string][]byte{"tools/review/SKILL.md": []byte("---\nname: review\ndescription: Review a change.\n---\n# Review\n")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	for _, test := range []struct {
		targets, warning string
		added            []string
	}{
		{"claude", "", []string{"../.claude/skills/review/SKILL.md", "generated/SKILLS.md", "vendor/tools/tools/review/SKILL.md"}},
		{"none", "The project has no skill targets, so no copy of its 1 skill(s) was installed.", []string{"generated/SKILLS.md", "vendor/tools/tools/review/SKILL.md"}},
	} {
		t.Run(test.targets, func(t *testing.T) {
			directory := t.TempDir()
			for _, args := range [][]string{{"project", "init", "--skill-targets", test.targets}, {"project", "add", "source", "tools", "--repository", f.Repository, "--ref", "v1.0.0", "--skills", "*", "--non-interactive"}} {
				if out, diagnostic, code := runCLI(t, binary, directory, args...); code != 0 {
					t.Fatal(args, code, out, diagnostic)
				}
			}
			out, diagnostic, code := runCLIWithEnvironment(t, binary, directory, f.Environment, "project", "sync", "--json")
			var result struct {
				OK    bool
				Value struct{ Added, Warnings []string }
			}
			if code != 0 || json.Unmarshal([]byte(out), &result) != nil || !result.OK {
				t.Fatal(code, out, diagnostic)
			}
			for _, name := range test.added {
				if !slices.Contains(result.Value.Added, name) {
					t.Errorf("added %v lacks %s", result.Value.Added, name)
				}
			}
			if test.warning == "" && len(result.Value.Warnings) != 0 || test.warning != "" && (len(result.Value.Warnings) != 1 || !strings.HasPrefix(result.Value.Warnings[0], test.warning)) {
				t.Errorf("warnings = %v", result.Value.Warnings)
			}
			_, err := os.Stat(filepath.Join(directory, ".claude/skills/review/SKILL.md"))
			if installed := err == nil; installed != (test.targets == "claude") {
				t.Errorf("installed copy exists: %v", installed)
			}
		})
	}
}
