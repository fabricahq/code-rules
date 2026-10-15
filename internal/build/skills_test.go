// Test the installed skill folders, SKILLS.md, and installed-skills.json that generation renders for authored skills.

package build_test

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/library"
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

func TestGenerateRejectsASourceSkillAndAnAuthoredSkillWithOneInstallName(t *testing.T) {
	commit := strings.Repeat("a", 40)
	config, err := rules.ParseConfiguration([]byte(`{"schemaVersion":2,"skillTargets":[],"sources":{"tools":{"repository":"https://example.com/tools.git","ref":"` + commit + `","skills":{"include":["*"]}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	skill := []byte("---\nname: shared\n---\n")
	tools := build.Library{
		Catalog:  library.Catalog{Rules: []string{}, Groups: []library.Group{}, SupportingFiles: map[string][]byte{}},
		Snapshot: library.Snapshot{NonLibrary: true, Commit: commit, Skills: map[string]library.ImportedSkill{"tools/shared": {}}, Files: map[string][]byte{"tools/shared/SKILL.md": skill}},
	}
	_, err = build.Generate(config, map[string]build.Library{"tools": tools}, nil, build.Options{ToolVersion: "test", LocalSkills: map[string]build.SkillFiles{"skills/shared": {Files: map[string][]byte{"SKILL.md": skill}}}})
	var clash *build.ClashError
	if !errors.As(err, &clash) || clash.Name != "shared" || clash.IDs != [2]string{"skills/shared", "tools:tools/shared"} {
		t.Fatalf("got %v; want a clash of skills/shared and tools:tools/shared", err)
	}
}

func TestInstalledSkillIDsNamesTheSkillInEachFolder(t *testing.T) {
	record := `{"formatVersion":1,"skills":[{"id":"skills/tidy","targets":[".agents/skills/tidy",".claude/skills/tidy"]},{"id":"team:skills/review","targets":[".claude/skills/review"]}]}`
	want := map[string]string{".agents/skills/tidy": "skills/tidy", ".claude/skills/tidy": "skills/tidy", ".claude/skills/review": "team:skills/review"}
	if got := build.InstalledSkillIDs([]byte(record)); !maps.Equal(got, want) {
		t.Fatalf("installed skills = %v, want %v", got, want)
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
