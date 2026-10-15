// Resolve the skills a project installs, and render their installed copies, SKILLS.md, and installed-skills.json.

package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/skills"
)

// SkillFiles is one skill folder's files, by path in the folder, and the paths of its executable files, sorted.
type SkillFiles struct {
	Files      map[string][]byte `json:"files"`
	Executable []string          `json:"executable"`
}

// ClashError reports two skills the project installs with the same install name, which names their installed
// folders, so both can't be installed.
type ClashError struct {
	Name string
	// IDs holds the two skills' project IDs, sorted.
	IDs [2]string
}

// Error names the skills and the next step.
func (e *ClashError) Error() string {
	return fmt.Sprintf("skills %s and %s both have the install name %s, and installed folders are named by install name, so both can't be installed; stop including one of them", e.IDs[0], e.IDs[1], e.Name)
}

// resolvedSkill is one skill the project installs.
type resolvedSkill struct {
	// ID is the project ID: source:folder for a source's skill, or skills/<name> for an authored one.
	ID string
	// Source is the source's name, or empty for an authored skill; Folder is the skill's ID in its source or in the
	// repository's skills/ folder.
	Source, Folder string
	// Name is the install name; Description and Type are as SKILL.md declares them.
	Name, Description, Type string
	Files                   SkillFiles
	// Repository, Ref, Commit, and Release identify the revision that supplied a source's skill; Release is 0 for a
	// revision other than a library release, and NonLibrary marks a source whose repository isn't a library.
	Repository, Ref, Commit string
	Release                 int
	NonLibrary              bool
	// Level is the skill's level in its library, or empty for an authored skill or one from another repository.
	Level string
	// Targets lists, sorted, the folders the skill installs into, by path from the repository root.
	Targets []string
	// RulesUsed lists the lines of the skill's syllabus-rules-used, and Cited how the project resolves each of them, in
	// the same order; both are empty for a skill that cites no rules.
	RulesUsed []string
	Cited     []citedRule
}

// resolveSkills returns, sorted by project ID, the skills of every configured source's snapshot and the authored
// skills in local, keyed by ID, skills/<name>. It fails when a SKILL.md is missing or its frontmatter unreadable,
// when an install name isn't valid, and with *ClashError when two skills share an install name.
func resolveSkills(config rules.Configuration, libraries map[string]Library, local map[string]SkillFiles) ([]resolvedSkill, error) {
	result := []resolvedSkill{}
	for _, source := range config.Sources {
		snapshot := libraries[source.Name].Snapshot
		for _, id := range slices.Sorted(maps.Keys(snapshot.Skills)) {
			skill := resolvedSkill{ID: source.Name + ":" + id, Source: source.Name, Folder: id, Files: SkillFiles{snapshot.SkillFiles(id), snapshot.SkillExecutable(id)}, Repository: source.Repository, Ref: source.Ref.String(), Commit: snapshot.Commit, Release: snapshot.Release, NonLibrary: snapshot.NonLibrary, Level: snapshot.Skills[id].Level}
			if err := describeSkill(&skill, "vendor/"+source.Name+"/"+id+"/"+skills.File); err != nil {
				return nil, err
			}
			result = append(result, skill)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(local)) {
		skill := resolvedSkill{ID: id, Folder: id, Files: local[id]}
		if err := describeSkill(&skill, id+"/"+skills.File); err != nil {
			return nil, err
		}
		result = append(result, skill)
	}
	slices.SortFunc(result, func(a, b resolvedSkill) int { return strings.Compare(a.ID, b.ID) })
	names := map[string]string{}
	for i, skill := range result {
		result[i].Targets = []string{}
		for _, folder := range config.SkillFolders() {
			result[i].Targets = append(result[i].Targets, folder+"/"+skill.Name)
		}
		if other, taken := names[skill.Name]; taken {
			return nil, &ClashError{Name: skill.Name, IDs: [2]string{other, skill.ID}}
		}
		names[skill.Name] = skill.ID
	}
	return result, nil
}

// describeSkill reads skill's SKILL.md, at location for diagnostics, for its install name, description, type, and
// cited rules. An authored skill's frontmatter must also pass skills.Check, which reports its first problem; a
// source's skills were checked by their library, or, from a repository that isn't a library, install as they are.
func describeSkill(skill *resolvedSkill, location string) error {
	data, ok := skill.Files.Files[skills.File]
	if !ok {
		return invalid(location, "missing the skill's SKILL.md")
	}
	metadata, err := skills.ReadMetadata(data, location)
	if err != nil {
		return err
	}
	if skill.Source == "" {
		if problems := skills.Check(path.Base(skill.Folder), metadata, true); len(problems) > 0 {
			return invalid(location, problems[0])
		}
	}
	if skill.Name, err = skills.InstallName(skill.Folder, metadata, location); err != nil {
		return err
	}
	skill.Description, skill.Type, skill.RulesUsed = metadata.Description, metadata.Type, metadata.RulesUsed
	return nil
}

// marker returns the line an installed copy of skill's SKILL.md carries: the project ID and the commit of a source's
// skill, and, for an authored skill, where to edit it.
func (skill resolvedSkill) marker() string {
	if skill.Source == "" {
		return fmt.Sprintf("<!-- code-rules:installed %s. Do not edit; edit .code-rules/%s/ instead. -->", skill.ID, skill.Folder)
	}
	return fmt.Sprintf("<!-- code-rules:installed %s %s. Do not edit; code-rules project sync replaces this copy. -->", skill.ID, skill.Commit)
}

// installed returns the files of skill's installed copy: its files with the marker line in SKILL.md, and, when the
// skill cites rules, the section that links to the project's version of each one at its end. Every skill target
// folder is two levels below the repository root, so one copy's links work in every target.
func (skill resolvedSkill) installed() SkillFiles {
	files := maps.Clone(skill.Files.Files)
	from := skills.File
	if len(skill.Targets) > 0 {
		from = skill.Targets[0] + "/" + skills.File
	}
	files[skills.File] = appendSection(skills.Mark(files[skills.File], skill.marker()), citedSection(skill, from))
	return SkillFiles{Files: files, Executable: slices.Clone(skill.Files.Executable)}
}

// renderSkills returns the installed skill folders by path from the repository root, and, when the project installs
// any skill, SKILLS.md and installed-skills.json by path in generated/.
func renderSkills(resolved []resolvedSkill) (map[string]SkillFiles, map[string][]byte, error) {
	folders := map[string]SkillFiles{}
	files := map[string][]byte{}
	if len(resolved) == 0 {
		return folders, files, nil
	}
	type record struct {
		Name    string   `json:"name"`
		ID      string   `json:"id"`
		Version *string  `json:"version"`
		Commit  *string  `json:"commit"`
		Targets []string `json:"targets"`
		Digest  string   `json:"digest"`
	}
	records := []record{}
	for _, skill := range resolved {
		installed := skill.installed()
		for _, target := range skill.Targets {
			folders[target] = installed
		}
		entry := record{Name: skill.Name, ID: skill.ID, Targets: skill.Targets, Digest: folderDigest(installed)}
		if skill.NonLibrary {
			entry.Commit = &skill.Commit
		}
		records = append(records, entry)
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(struct {
		FormatVersion   int      `json:"formatVersion"`
		GeneratedNotice string   `json:"generatedNotice"`
		Skills          []record `json:"skills"`
	}{1, generatedNotice + " " + strings.ReplaceAll(regenerationNotice, "`", ""), records}); err != nil {
		return nil, nil, fmt.Errorf("encode installed skills: %w", err)
	}
	files[InstalledSkillsFile] = encoded.Bytes()
	files["SKILLS.md"] = []byte(generatedMarkdown(skillsIndex(resolved)))
	return folders, files, nil
}

// InstalledSkillsFile is the record, in generated/, of the skill folders Code Rules installed, which is how it knows
// which folders in the skill targets it owns.
const InstalledSkillsFile = "installed-skills.json"

// InstalledFolders reads the folders an installed-skills.json record lists, by path from the repository root,
// sorted. A record it can't read lists none, so no folder counts as installed by Code Rules.
func InstalledFolders(data []byte) []string {
	return slices.Sorted(maps.Keys(InstalledSkills(data)))
}

// InstalledSkills reads the folders an installed-skills.json record lists, by path from the repository root, each
// with the project ID of the skill installed there. It keeps only folders in a skill target folder whose name
// follows the skill name grammar. A record it can't read lists none, so no folder counts as installed by Code Rules.
func InstalledSkills(data []byte) map[string]string {
	var record struct {
		FormatVersion int `json:"formatVersion"`
		Skills        []struct {
			ID      string   `json:"id"`
			Targets []string `json:"targets"`
		} `json:"skills"`
	}
	folders := map[string]string{}
	if json.Unmarshal(data, &record) != nil || record.FormatVersion != 1 {
		return folders
	}
	for _, skill := range record.Skills {
		for _, target := range skill.Targets {
			if _, listed := folders[target]; !listed && slices.Contains(slices.Collect(maps.Values(rules.SkillTargetFolders)), path.Dir(target)) && skills.ValidName(path.Base(target)) {
				folders[target] = skill.ID
			}
		}
	}
	return folders
}

// folderDigest returns the SHA-256 of an installed folder: each file's path, executable bit, and contents.
func folderDigest(folder SkillFiles) string {
	entries := [][3]string{}
	for _, name := range slices.Sorted(maps.Keys(folder.Files)) {
		mode := "100644"
		if slices.Contains(folder.Executable, name) {
			mode = "100755"
		}
		sum := sha256.Sum256(folder.Files[name])
		entries = append(entries, [3]string{name, mode, hex.EncodeToString(sum[:])})
	}
	encoded, _ := json.Marshal(entries)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// skillsIndex renders SKILLS.md: one entry per installed skill, saying what it does, where it comes from, and where
// agents find it.
func skillsIndex(resolved []resolvedSkill) string {
	sections := []string{"# Skills", "This project installs the agent skills below. Agents that load skills from a skill target folder find each one there. Any other agent can read a skill's `SKILL.md` at the path given and follow it when its description matches the task. These files are generated; edit the configuration or the authored skills in `.code-rules/skills/` and rebuild."}
	for _, skill := range resolved {
		lines := []string{"### " + skill.Name}
		if description := strings.TrimSpace(skill.Description); description != "" {
			lines = append(lines, escapeText(description))
		}
		facts := []string{}
		switch {
		case skill.Source == "":
			facts = append(facts, "- **Skill:** `"+skill.ID+"`, authored in this project")
		case skill.Release != 0:
			facts = append(facts, fmt.Sprintf("- **Skill:** `%s`, from release/%d, commit `%s`", skill.ID, skill.Release, skill.Commit))
		default:
			facts = append(facts, fmt.Sprintf("- **Skill:** `%s`, commit `%s`", skill.ID, skill.Commit))
		}
		if skill.Level != "" {
			setBy := "`" + skill.Source + "`"
			if skill.Release != 0 {
				setBy += fmt.Sprintf(", release %d", skill.Release)
			}
			facts = append(facts, "- **Level:** "+skill.Level+" by "+setBy)
		}
		if skill.Type != "" {
			facts = append(facts, "- **Type:** "+escapeText(skill.Type))
		}
		if len(skill.Cited) > 0 && len(skill.Targets) > 0 {
			facts = append(facts, fmt.Sprintf("- **Applies:** %d rule(s), listed at the end of its installed SKILL.md", len(skill.Cited)))
		} else if len(skill.Cited) > 0 {
			lines := []string{}
			for _, cited := range skill.Cited {
				lines = append(lines, "  - "+citedLine(cited, ".code-rules/generated/SKILLS.md"))
			}
			facts = append(facts, "- **Applies:**\n"+strings.Join(lines, "\n"))
		}
		if len(skill.Targets) == 0 {
			source := ".code-rules/" + skill.Folder + "/" + skills.File
			if skill.Source != "" {
				source = ".code-rules/vendor/" + skill.Source + "/" + skill.Folder + "/" + skills.File
			}
			facts = append(facts, "- **Installed at:** nowhere, because the project has no skill targets. Read it at ["+source+"]("+rules.RelativeLink(".code-rules/generated/SKILLS.md", source)+").")
		} else {
			links := []string{}
			for _, target := range skill.Targets {
				file := target + "/" + skills.File
				links = append(links, "["+file+"]("+rules.RelativeLink(".code-rules/generated/SKILLS.md", file)+")")
			}
			facts = append(facts, "- **Installed at:** "+strings.Join(links, ", "))
		}
		lines = append(lines, strings.Join(facts, "\n"))
		sections = append(sections, strings.Join(lines, "\n\n"))
	}
	return strings.Join(sections, "\n\n") + "\n"
}
