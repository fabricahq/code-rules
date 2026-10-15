// Find personal skills, in the known folders of a person's home directory, with the name of a skill the project
// installs, and say which copy each agent that reads both uses. Nothing here writes to the home directory.

package project

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/skills"
)

// personalSkillFolders are the folders, relative to a home directory, where agents load a person's own skills, in
// the order warnings name them.
var personalSkillFolders = []string{".claude/skills", ".agents/skills", ".codex/skills"}

// personalSkillAgents are the agents a warning can name, each with the personal folder, relative to the home
// directory, and the skill target folder, relative to the repository, in which it reads skills, and the line saying
// which copy it uses when it reads both. Only agents whose folders are known are listed, so a warning never claims
// what an agent does with copies it may not read.
var personalSkillAgents = []struct{ personal, project, line string }{
	{".claude/skills", ".claude/skills", "Claude Code uses your personal skill."},
}

// personalSkill is a personal skill with the name of a skill the project installs.
type personalSkill struct {
	// folder is the personal skill's folder relative to the home directory, such as .claude/skills/review.
	folder    string
	installed build.InstalledSkill
}

// personalSkillWarnings returns one warning for each personal skill in home with the install name of a skill that
// record, an installed-skills.json, lists with at least one installed folder, in the record's order and then
// personalSkillFolders order; repository is the repository root. It returns an empty list when home is empty or
// isn't absolute, and skips every folder it can't read, so a missing or unreadable home warns about nothing. It
// reads only home and the skills' installed folders in repository, and writes nothing.
func personalSkillWarnings(home, repository string, record []byte) []string {
	warnings := []string{}
	if home == "" || !filepath.IsAbs(home) {
		return warnings
	}
	for _, personal := range findPersonalSkills(home, repository, build.InstalledSkills(record)) {
		warnings = append(warnings, personalSkillWarning(personal))
	}
	return warnings
}

// findPersonalSkills returns each folder in personalSkillFolders of home that holds a SKILL.md and is named for a
// skill in installed with at least one installed folder. It compares folder names only, never contents. It skips a
// folder that is one it already found, as when ~/.agents/skills links to ~/.claude/skills, and one that is the
// skill's own installed folder in repository, as when the repository is the home directory.
func findPersonalSkills(home, repository string, installed []build.InstalledSkill) []personalSkill {
	found := []personalSkill{}
	for _, skill := range installed {
		if len(skill.Targets) == 0 || !skills.ValidName(skill.Name) {
			continue
		}
		seen := []os.FileInfo{}
		for _, target := range skill.Targets {
			if info, err := os.Stat(filepath.Join(repository, filepath.FromSlash(target))); err == nil {
				seen = append(seen, info)
			}
		}
		for _, parent := range personalSkillFolders {
			folder := parent + "/" + skill.Name
			info, ok := skillFolder(filepath.Join(home, filepath.FromSlash(folder)))
			if !ok || sameAsAny(info, seen) {
				continue
			}
			seen = append(seen, info)
			found = append(found, personalSkill{folder: folder, installed: skill})
		}
	}
	return found
}

// skillFolder returns the folder at name, following links, when it is a folder holding a regular SKILL.md file. It
// reports false for anything it can't read.
func skillFolder(name string) (os.FileInfo, bool) {
	info, err := os.Stat(name)
	if err != nil || !info.IsDir() {
		return nil, false
	}
	file, err := os.Stat(filepath.Join(name, skills.File))
	if err != nil || !file.Mode().IsRegular() {
		return nil, false
	}
	return info, true
}

// sameAsAny reports whether info describes the same file as any of others.
func sameAsAny(info os.FileInfo, others []os.FileInfo) bool {
	for _, other := range others {
		if os.SameFile(info, other) {
			return true
		}
	}
	return false
}

// personalSkillWarning names both of personal's paths, the personal one from ~, and, in personalSkillAgents order,
// which copy each agent uses that reads both personal's folder and one of the skill's installed folders.
func personalSkillWarning(personal personalSkill) string {
	skill := personal.installed
	var agents strings.Builder
	for _, agent := range personalSkillAgents {
		if personal.folder == agent.personal+"/"+skill.Name && slices.Contains(skill.Targets, agent.project+"/"+skill.Name) {
			agents.WriteString("  " + agent.line + "\n")
		}
	}
	return fmt.Sprintf("your personal skill ~/%s has the same name as this project's %s (%s, installed at %s).\n%s"+
		"Rename or remove your personal skill to use the project's.", personal.folder, skill.Name, skill.ID, strings.Join(skill.Targets, " and "), agents.String())
}
