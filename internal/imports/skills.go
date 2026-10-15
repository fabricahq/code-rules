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

// offered is the level of every skill a library publishes until libraries can set levels.
const offered = "offered"

// skillFolders returns the skills in files, a commit's tree by repository path, keyed by skill ID, each with its
// files keyed by their path in the vendor snapshot. A library's skills are the folders directly in its
// .code-rules/skills/ that hold a SKILL.md, with IDs skills/<name> and snapshot paths in skills/. Any other
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

// chooseSkills returns, sorted, the IDs of the skills the source's skills.include selects from available, the IDs
// of the skills in the plan's revision. "*" selects every skill except those in a folder whose name starts with a
// dot, such as another tool's installed copies in .claude/skills/. A named skill the revision lacks fails: with
// skill-not-in-revision when the revision is the one the source record locks, since sync never moves a source, and
// otherwise as invalid configuration.
func chooseSkills(source rules.Source, plan sourcePlan, available []string) ([]string, error) {
	if slices.Equal(source.Skills, []string{skills.Wildcard}) {
		chosen := []string{}
		for _, id := range available {
			if !slices.ContainsFunc(strings.Split(id, "/"), func(part string) bool { return strings.HasPrefix(part, ".") }) {
				chosen = append(chosen, id)
			}
		}
		return chosen, nil
	}
	revision := revisionName(plan.release, plan.commit)
	for _, id := range source.Skills {
		if slices.Contains(available, id) {
			continue
		}
		if plan.locked != "" && plan.locked == plan.commit {
			next := fmt.Sprintf("Run code-rules project update %s, which moves the source to its newest library release, and takes the skill from it if it has it, or check the skill ID.", source.Name)
			if !source.Ref.IsZero() {
				next = fmt.Sprintf("To take the skill, set sources.%s.ref to a tag or commit that has it, then run code-rules project sync, or check the skill ID.", source.Name)
			}
			return nil, fail("skill-not-in-revision", fmt.Sprintf("sources.%s.skills.include names %s, which isn't in %s, the revision vendor/%s/_source.json records for the source. Sync never moves a source, so it takes every skill from that revision. %s", source.Name, id, revision, source.Name, next), nil)
		}
		return nil, &ValidationError{Location: "sources." + source.Name + ".skills.include", Problem: fmt.Sprintf("the source has no skill %s at %s; check the skill ID. A library's skills are the folders in its .code-rules/skills/, such as skills/build-feature; another repository's are the folders that hold a SKILL.md, named by their path from the repository root", id, revision)}
	}
	return slices.Clone(source.Skills), nil
}

// isLibrary reports whether a fetched commit holds a library: a .code-rules/config.yaml with a library section, or a
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
	file := librarytree.RepositoryPath(librarytree.Manifest)
	entry, ok := tree[file]
	if !ok {
		return false, nil
	}
	data, err := newGitFiles(ctx, r, map[string]treeEntry{file: entry}).ReadFile(file)
	if err != nil {
		return false, err
	}
	_, encoded, err := decode.YAML(data, file)
	var fields map[string]json.RawMessage
	if err != nil || json.Unmarshal(encoded, &fields) != nil {
		return true, nil
	}
	_, library := fields["library"]
	return library, nil
}

// importedSkills is what a source installs of its skills: their files and executable files by snapshot path, and
// each skill's record by ID.
type importedSkills struct {
	files      map[string][]byte
	executable []string
	skills     map[string]library.ImportedSkill
}

// readSkills reads the files of the skills the source selects at the plan's commit, a library's when isLibrary is
// set. It fails for a skill file that is a symbolic link or submodule, and over the import byte limits.
func (r *repository) readSkills(ctx context.Context, source rules.Source, plan sourcePlan, isLibrary bool) (importedSkills, error) {
	result := importedSkills{files: map[string][]byte{}, executable: []string{}, skills: map[string]library.ImportedSkill{}}
	if len(source.Skills) == 0 {
		return result, nil
	}
	tree, err := r.repositoryTree(ctx, plan.commit)
	if err != nil {
		return importedSkills{}, err
	}
	folders := skillFolders(tree, isLibrary)
	chosen, err := chooseSkills(source, plan, slices.Sorted(maps.Keys(folders)))
	if err != nil {
		return importedSkills{}, err
	}
	entries := map[string]treeEntry{}
	prefetch := []treeEntry{}
	for _, id := range chosen {
		level := ""
		if isLibrary {
			level = offered
		}
		result.skills[id] = library.ImportedSkill{Level: level}
		for name, entry := range folders[id] {
			if !entry.mode.IsRegular() {
				return importedSkills{}, fail("unsupported-content", fmt.Sprintf("sources.%s: skill %s has a symbolic link or submodule at %s, which Code Rules doesn't install. Ask the source's maintainer to replace it with a file.", source.Name, id, name), nil)
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
