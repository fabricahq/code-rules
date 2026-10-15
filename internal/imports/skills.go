// Find, choose, and read the skills a source installs, all from the one revision the source's plan names.

package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/decode"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/librarytree"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/skills"
)

// librarySkillsFolder is where a library keeps the skills it publishes, relative to its repository root.
const librarySkillsFolder = librarytree.Directory + "/" + skills.Directory + "/"

// skillFolders returns the skills in files, a commit's tree by repository path, keyed by skill ID, each with its
// files keyed by their path in the vendor snapshot. A library's skills are the folders directly in its
// .syllabus/skills/ that hold a SKILL.md, with IDs skills/<name> and snapshot paths in skills/. Any other
// repository's are the folders that hold a SKILL.md, other than the repository root and a folder inside another
// skill's folder, which belongs to that skill; their IDs and snapshot paths are their repository paths.
func skillFolders(files map[string]treeEntry, isLibrary bool) map[string]map[string]treeEntry {
	ids := []string{}
	for name := range files {
		folder, found := strings.CutSuffix(name, "/"+skills.File)
		if !found {
			continue
		}
		if isLibrary {
			if skill, inside := strings.CutPrefix(folder, librarySkillsFolder); inside && !strings.Contains(skill, "/") {
				ids = append(ids, skills.AuthoredID(skill))
			}
			continue
		}
		ids = append(ids, folder)
	}
	slices.Sort(ids)
	// Sorting puts each folder before the folders inside it, so an outer skill is kept before its inner ones are seen.
	kept := []string{}
	for _, id := range ids {
		if !slices.ContainsFunc(kept, func(outer string) bool { return strings.HasPrefix(id, outer+"/") }) {
			kept = append(kept, id)
		}
	}
	result := map[string]map[string]treeEntry{}
	for _, id := range kept {
		result[id] = map[string]treeEntry{}
	}
	for name, entry := range files {
		stored := name
		if isLibrary {
			inside, found := strings.CutPrefix(name, librarytree.Directory+"/")
			if !found {
				continue
			}
			stored = inside
		}
		for id := range result {
			if strings.HasPrefix(stored, id+"/") {
				entry.name = stored
				result[id][stored] = entry
				break
			}
		}
	}
	return result
}

// chooseSkills returns, sorted, the IDs of the skills the source installs from available, the IDs of the skills in
// the plan's revision, given levels, the levels its library gives them, or nil for a repository that isn't a
// library: a library's required skills, its recommended ones when skills.include has recommended, for "*" every
// skill except those in a folder whose name starts with a dot, such as another tool's installed copies in
// .claude/skills/, and each skill it names. A named skill the revision lacks fails: with skill-not-in-revision when
// the revision is the one the source record locks, since sync never moves a source, and otherwise as invalid
// configuration. Excluding a required skill fails too, since a project that uses the library can't diverge from it.
func chooseSkills(source rules.Source, plan sourcePlan, available []string, levels *rules.SkillLevels) ([]string, error) {
	chosen := []string{}
	revision := revisionName(plan.release, plan.commit)
	if levels != nil {
		for _, id := range levels.Listed() {
			if !slices.Contains(available, id) {
				return nil, fail("unsupported-content", fmt.Sprintf("sources.%s: the library's manifest at %s gives skill %s a level, but the library has no such skill there. Ask the library's maintainer to fix library.skills; syllabus library check catches this.", source.Name, revision, id), nil)
			}
		}
		chosen = append(chosen, levels.Required...)
		for _, id := range slices.Sorted(maps.Keys(source.SkillExclude)) {
			if slices.Contains(levels.Required, id) {
				return nil, &ValidationError{Location: "sources." + source.Name + ".skills.exclude." + id, Problem: rules.RequiredSkillProblem(source.Name, id, plan.release)}
			}
		}
	}
	for _, id := range source.Skills {
		switch {
		case id == skills.Wildcard:
			chosen = append(chosen, slices.DeleteFunc(slices.Clone(available), library.HiddenSkill)...)
		case id == skills.Recommended && levels == nil:
			return nil, &ValidationError{Location: "sources." + source.Name + ".skills.include", Problem: fmt.Sprintf("the repository isn't a library at %s, so it recommends no skills; name its skills by ID, or include \"*\" for every skill it has", revision)}
		case id == skills.Recommended:
			chosen = append(chosen, levels.Recommended...)
		case slices.Contains(available, id):
			chosen = append(chosen, id)
		case plan.locked != "" && plan.locked == plan.commit:
			next := fmt.Sprintf("Run syllabus project update %s, which moves the source to its newest library release, and takes the skill from it if it has it, or check the skill ID.", source.Name)
			if !source.Ref.IsZero() {
				next = fmt.Sprintf("To take the skill, set sources.%s.ref to a tag or commit that has it, then run syllabus project sync, or check the skill ID.", source.Name)
			}
			return nil, fail("skill-not-in-revision", fmt.Sprintf("sources.%s.skills.include names %s, which isn't in %s, the revision vendor/%s/_source.json records for the source. Sync never moves a source, so it takes every skill from that revision. %s", source.Name, id, revision, source.Name, next), nil)
		default:
			return nil, &ValidationError{Location: "sources." + source.Name + ".skills.include", Problem: fmt.Sprintf("the source has no skill %s at %s; check the skill ID. A library's skills are the folders in its .syllabus/skills/, such as skills/build-feature; another repository's are the folders that hold a SKILL.md, named by their path from the repository root", id, revision)}
		}
	}
	slices.Sort(chosen)
	return slices.Compact(chosen), nil
}

// isLibrary reports whether a fetched commit holds a library: a .syllabus/config.yaml with a library section, or a
// rule-library.yaml in the layout before version 2. A configuration it can't read counts as a library's, so the
// catalog loader reports what is wrong with it.
func (r *repository) isLibrary(ctx context.Context, commit string) (bool, error) {
	tree, err := r.repositoryTree(ctx, commit)
	if err != nil {
		return false, err
	}
	if _, legacy := tree[librarytree.LegacyManifest]; legacy {
		return true, nil
	}
	data, found, err := r.manifest(ctx, tree)
	if err != nil || !found {
		return false, err
	}
	_, encoded, err := decode.YAML(data, librarytree.RepositoryPath(librarytree.Manifest))
	var fields map[string]json.RawMessage
	if err != nil || json.Unmarshal(encoded, &fields) != nil {
		return true, nil
	}
	_, library := fields["library"]
	return library, nil
}

// skillLevels returns the levels the library at a fetched commit gives its skills, or nil when the commit holds no
// library. A library in the layout before version 2 gives none, so each of its skills is offered.
func (r *repository) skillLevels(ctx context.Context, commit string) (*rules.SkillLevels, error) {
	tree, err := r.repositoryTree(ctx, commit)
	if err != nil {
		return nil, err
	}
	if _, legacy := tree[librarytree.LegacyManifest]; legacy {
		return &rules.SkillLevels{Required: []string{}, Recommended: []string{}, Offered: []string{}}, nil
	}
	data, found, err := r.manifest(ctx, tree)
	if err != nil || !found {
		return nil, err
	}
	declaration, err := rules.ParseLibraryManifest(data, librarytree.RepositoryPath(librarytree.Manifest))
	if err != nil || declaration == nil {
		return nil, err
	}
	return &declaration.Skills, nil
}

// manifest reads .syllabus/config.yaml from tree, a commit's tree by repository path, reporting false when it has
// none.
func (r *repository) manifest(ctx context.Context, tree map[string]treeEntry) ([]byte, bool, error) {
	file := librarytree.RepositoryPath(librarytree.Manifest)
	entry, ok := tree[file]
	if !ok {
		return nil, false, nil
	}
	data, err := newGitFiles(ctx, r, map[string]treeEntry{file: entry}).ReadFile(file)
	return data, err == nil, err
}

// importedSkills is what a source installs of its skills: their files and executable files by snapshot path, each
// skill's record by ID, and the level of every skill the source offers at the revision.
type importedSkills struct {
	files      map[string][]byte
	executable []string
	skills     map[string]library.ImportedSkill
	available  map[string]library.ImportedSkill
}

// readSkills reads the files of the skills the source installs at the plan's commit, as chooseSkills chooses them,
// a library's when isLibrary is set. It fails for a skill file that is a symbolic link or submodule, over the import
// byte limits, and as requireSkillBasedOn does for a replacement's basedOn.
func (r *repository) readSkills(ctx context.Context, source rules.Source, plan sourcePlan, isLibrary bool) (importedSkills, error) {
	result := importedSkills{files: map[string][]byte{}, executable: []string{}, skills: map[string]library.ImportedSkill{}, available: map[string]library.ImportedSkill{}}
	if err := r.requireSkillBasedOn(ctx, source); err != nil {
		return importedSkills{}, err
	}
	tree, err := r.repositoryTree(ctx, plan.commit)
	if err != nil {
		return importedSkills{}, err
	}
	var levels *rules.SkillLevels
	if isLibrary {
		if levels, err = r.skillLevels(ctx, plan.commit); err != nil {
			return importedSkills{}, err
		}
	}
	folders := skillFolders(tree, isLibrary)
	available := slices.Sorted(maps.Keys(folders))
	for _, id := range available {
		result.available[id] = importedSkill(id, levels)
	}
	chosen, err := chooseSkills(source, plan, available, levels)
	if err != nil {
		return importedSkills{}, err
	}
	entries := map[string]treeEntry{}
	prefetch := []treeEntry{}
	for _, id := range chosen {
		result.skills[id] = importedSkill(id, levels)
		for name, entry := range folders[id] {
			if !entry.mode.IsRegular() {
				return importedSkills{}, fail("unsupported-content", fmt.Sprintf("sources.%s: skill %s has a symbolic link or submodule at %s, which Syllabus doesn't install. Ask the source's maintainer to replace it with a file.", source.Name, id, name), nil)
			}
			entries[name] = entry
			prefetch = append(prefetch, entry)
		}
	}
	if err := r.prefetch(ctx, prefetch); err != nil {
		return importedSkills{}, err
	}
	input := newGitFiles(ctx, r, entries)
	for _, name := range slices.Sorted(maps.Keys(entries)) {
		data, err := input.ReadFile(name)
		if err != nil {
			return importedSkills{}, err
		}
		result.files[name] = data
		if entries[name].mode.Perm()&0o100 != 0 {
			result.executable = append(result.executable, name)
		}
	}
	return result, nil
}

// importedSkill returns the record of skill id given levels, the levels its library gives its skills, or nil for a
// repository that isn't a library, whose skills have no level.
func importedSkill(id string, levels *rules.SkillLevels) library.ImportedSkill {
	if levels == nil {
		return library.ImportedSkill{}
	}
	return library.ImportedSkill{Level: levels.Level(id)}
}

// skillLevelChanges returns, sorted by ID, the update preview's rows for the skills whose level changes between the
// skills recorded imported and the library at commit, the revision an update takes them from. A recorded source
// that isn't a library, or a commit that holds none, has none.
func (r *repository) skillLevelChanges(ctx context.Context, recorded *library.Snapshot, commit string) ([]SkillLevelUpdate, error) {
	rows := []SkillLevelUpdate{}
	if recorded == nil || recorded.NonLibrary {
		return rows, nil
	}
	levels, err := r.skillLevels(ctx, commit)
	if err != nil || levels == nil {
		return rows, err
	}
	tree, err := r.repositoryTree(ctx, commit)
	if err != nil {
		return nil, err
	}
	after := map[string]string{}
	for id := range skillFolders(tree, true) {
		after[id] = levels.Level(id)
	}
	ids := slices.Sorted(maps.Keys(after))
	for id := range recorded.AvailableSkills {
		if _, kept := after[id]; !kept {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		from, to := recorded.AvailableSkills[id].Level, after[id]
		if from != to && (from != "" && to != "" || from == skills.LevelRequired || to == skills.LevelRequired) {
			rows = append(rows, SkillLevelUpdate{ID: id, From: from, To: to})
		}
	}
	return rows, nil
}
