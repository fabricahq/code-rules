// Test the installed skill folders, SKILLS.md, and installed-skills.json that generation renders for authored skills.

package build_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/rules"
)

// skillConfig parses a configuration with no sources and the skill targets in targets, a JSON list.
func skillConfig(t *testing.T, targets string) rules.Configuration {
	t.Helper()
	config, err := rules.ParseConfiguration([]byte(`{"schemaVersion":2,"skillTargets":` + targets + `,"sources":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestGenerateInstallsAuthoredSkillsIntoEachTarget(t *testing.T) {
	local := map[string]build.SkillFiles{"skills/tidy": {Files: map[string][]byte{"SKILL.md": []byte("---\nname: tidy\ndescription: Tidy | code.\n---\n# Tidy\n"), "run.sh": []byte("#!/bin/sh\n")}, Executable: []string{"run.sh"}}}
	output, err := build.Generate(skillConfig(t, `["claude","agents"]`), nil, nil, build.Options{ToolVersion: "test", LocalSkills: local})
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{".claude/skills/tidy", ".agents/skills/tidy"} {
		installed, ok := output.Skills[folder]
		if !ok || !strings.Contains(string(installed.Files["SKILL.md"]), "---\n<!-- code-rules:installed skills/tidy. Do not edit; edit .code-rules/skills/tidy/ instead. -->\n# Tidy") || !slices.Equal(installed.Executable, []string{"run.sh"}) {
			t.Errorf("%s = %+v", folder, installed)
		}
	}
	if !slices.Equal(output.SkillIDs, []string{"skills/tidy"}) {
		t.Errorf("skill IDs = %v", output.SkillIDs)
	}
	if got := build.InstalledFolders(output.Files[build.InstalledSkillsFile]); !slices.Equal(got, []string{".agents/skills/tidy", ".claude/skills/tidy"}) {
		t.Errorf("installed folders = %v", got)
	}
	if index := string(output.Files["SKILLS.md"]); !strings.Contains(index, "### tidy\n\nTidy \\| code.") {
		t.Errorf("SKILLS.md:\n%s", index)
	}
	if !strings.Contains(string(output.Files["RULES.md"]), "This project also installs agent skills.") {
		t.Errorf("RULES.md doesn't point to SKILLS.md:\n%s", output.Files["RULES.md"])
	}
}

func TestGenerateWithoutSkillsAddsNoSkillFiles(t *testing.T) {
	output, err := build.Generate(skillConfig(t, `["claude"]`), nil, nil, build.Options{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Skills) != 0 || output.Files["SKILLS.md"] != nil || output.Files[build.InstalledSkillsFile] != nil || strings.Contains(string(output.Files["RULES.md"]), "SKILLS.md") {
		t.Fatalf("output without skills has skill files: %v", output.Skills)
	}
}

func TestGenerateRejectsTwoSkillsWithOneInstallName(t *testing.T) {
	skill := build.SkillFiles{Files: map[string][]byte{"SKILL.md": []byte("---\nname: shared\n---\n")}}
	_, err := build.Generate(skillConfig(t, `[]`), nil, nil, build.Options{ToolVersion: "test", LocalSkills: map[string]build.SkillFiles{"skills/a": skill, "skills/b": skill}})
	var clash *build.ClashError
	if !errors.As(err, &clash) || clash.Name != "shared" || clash.IDs != [2]string{"skills/a", "skills/b"} {
		t.Fatalf("got %v; want a clash of skills/a and skills/b", err)
	}
}

func TestInstalledFoldersListsOnlySkillTargetFolders(t *testing.T) {
	record := `{"formatVersion":1,"skills":[{"targets":[".claude/skills/tidy","../outside/tidy",".claude/skills/../../x",".github/skills/tidy",".agents/skills/Bad_Name",".agents/skills/tidy"]}]}`
	if got := build.InstalledFolders([]byte(record)); !slices.Equal(got, []string{".agents/skills/tidy", ".claude/skills/tidy"}) {
		t.Fatalf("installed folders = %v", got)
	}
	for _, unreadable := range []string{"", "{", `{"formatVersion":2,"skills":[{"targets":[".claude/skills/tidy"]}]}`} {
		if got := build.InstalledFolders([]byte(unreadable)); len(got) != 0 {
			t.Errorf("InstalledFolders(%q) = %v, want none", unreadable, got)
		}
	}
}
