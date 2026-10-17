package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/trailguide/internal/test/gitfixture"
)

// forkingLibrary returns a library that curates pocock's deploy skill at tag stable, the pocock fixture, and the
// environment that routes to it.
func forkingLibrary(t *testing.T) (string, *gitfixture.Fixture, []string) {
	t.Helper()
	ctx := context.Background()
	pocock, err := gitfixture.New(ctx, map[string][]byte{
		"skills/ops/deploy/SKILL.md":          []byte("---\nname: deploy\ndescription: Deploy a service.\n---\n# Deploy\n"),
		"skills/ops/deploy/scripts/deploy.sh": []byte("#!/bin/sh\necho deploy\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pocock.Close() })
	if _, err := pocock.Command(ctx, "tag", "stable", pocock.FirstCommit); err != nil {
		t.Fatal(err)
	}
	environment, err := pocock.Route(map[string]*gitfixture.Fixture{"pocock": pocock})
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	config := "schemaVersion: 2\nsources:\n  pocock:\n    repository: git@fixture.invalid:pocock\n    ref: stable\nlibrary:\n  skills:\n    required: [pocock:skills/ops/deploy]\n"
	file := filepath.Join(directory, ".trailguide", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	return directory, pocock, environment
}

func TestLibraryAddSkill_ForksACuratedSkillAndCheckListsTheSourcesChanges(t *testing.T) {
	ctx := context.Background()
	directory, pocock, environment := forkingLibrary(t)
	binary := buildCLI(t)
	out, diagnostic, code := runCLIWithEnvironment(t, binary, directory, environment, "library", "add", "skill", "--from", "pocock:skills/ops/deploy", "--reason", "Deploy with our pipeline.")
	if code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out, diagnostic)
	}
	skill, err := os.ReadFile(filepath.Join(directory, ".trailguide", "skills", "deploy", "SKILL.md"))
	if err != nil || !strings.Contains(string(skill), "# Deploy") {
		t.Fatalf("forked SKILL.md: %q, %v", skill, err)
	}
	if _, err := os.Stat(filepath.Join(directory, ".trailguide", "skills", "deploy", "scripts", "deploy.sh")); err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile(filepath.Join(directory, ".trailguide", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"replacedBy: skills/deploy", "basedOn: \"" + pocock.FirstCommit + "\"", "reason: Deploy with our pipeline.", "required: [skills/deploy]"} {
		if !strings.Contains(string(config), want) {
			t.Fatalf("config lacks %q:\n%s", want, config)
		}
	}

	out, diagnostic, code = runCLIWithEnvironment(t, binary, directory, environment, "library", "check")
	if code != 0 || strings.Contains(out, "Forked skills") {
		t.Fatalf("exit %d before the source changed:\n%s%s", code, out, diagnostic)
	}

	changed, err := pocock.Commit(ctx, pocock.Worktree(), "Change deploy", map[string][]byte{"skills/ops/deploy/scripts/deploy.sh": []byte("#!/bin/sh\necho deploy v2\n")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pocock.Command(ctx, "tag", "--force", "stable", changed); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code = runCLIWithEnvironment(t, binary, directory, environment, "library", "check")
	if code != 0 || !strings.Contains(out, "Forked skills that their sources changed:") || !strings.Contains(out, "scripts/deploy.sh") {
		t.Fatalf("exit %d after the source changed:\n%s%s", code, out, diagnostic)
	}
	out, diagnostic, code = runCLIWithEnvironment(t, binary, directory, environment, "library", "check", "--json")
	for _, want := range []string{`"forkedSkills": [`, `"skill": "skills/deploy"`, `"basedOn": "` + pocock.FirstCommit + `"`, `"commit": "` + changed + `"`} {
		if code != 0 || !strings.Contains(out, want) {
			t.Fatalf("exit %d, want %s:\n%s%s", code, want, out, diagnostic)
		}
	}
}

func TestLibraryAddSkill_RefusesASkillTheSourceLacks(t *testing.T) {
	directory, _, environment := forkingLibrary(t)
	out, diagnostic, code := runCLIWithEnvironment(t, buildCLI(t), directory, environment, "library", "add", "skill", "--from", "pocock:skills/ops/grill", "--reason", "Ours.")
	if code == 0 || !strings.Contains(out+diagnostic, "source pocock has no skill skills/ops/grill at stable") {
		t.Fatalf("exit %d:\n%s%s", code, out, diagnostic)
	}
}
