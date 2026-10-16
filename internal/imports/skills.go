// Find, choose, and read the skills a source installs, all from the one revision the source's plan names.

package imports

import (
	"context"
	"encoding/json"
	"errors"
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
	for _, id := range source.Skills {
		if !slices.Contains(available, id) {
			return nil, skillNotInRevision(source, plan, "sources."+source.Name+".skills.include", id)
		}
	}
	return slices.Clone(source.Skills), nil
}

// skillNotInRevision reports that field, a configuration field of source, names skill id, which the plan's revision
// lacks: with skill-not-in-revision when the revision is the one the source record locks, since sync never moves a
// source, and otherwise as invalid configuration at field.
func skillNotInRevision(source rules.Source, plan sourcePlan, field, id string) error {
	revision := revisionName(plan.release, plan.commit)
	if plan.locked != "" && plan.locked == plan.commit {
		next := fmt.Sprintf("Run code-rules project update %s, which moves the source to its newest library release, and takes the skill from it if it has it, or check the skill ID.", source.Name)
		if !source.Ref.IsZero() {
			next = fmt.Sprintf("To take the skill, set sources.%s.ref to a tag or commit that has it, then run code-rules project sync, or check the skill ID.", source.Name)
		}
		return fail("skill-not-in-revision", fmt.Sprintf("%s names %s, which isn't in %s, the revision vendor/%s/_source.json records for the source. Sync never moves a source, so it takes every skill from that revision. %s", field, id, revision, source.Name, next), nil)
	}
	return &ValidationError{Location: field, Problem: fmt.Sprintf("the source has no skill %s at %s; check the skill ID. A library's skills are the folders in its .code-rules/skills/, such as skills/build-feature; another repository's are the folders that hold a SKILL.md, named by their path from the repository root", id, revision)}
}

// requireConfiguredUses checks that every skill the source's skills.uses lists is one of available, the IDs of the
// skills in the plan's revision, failing as skillNotInRevision does. Whether each key is installed is
// installedSkills' to check, once it knows what the source installs.
func requireConfiguredUses(source rules.Source, plan sourcePlan, available []string) error {
	for _, id := range slices.Sorted(maps.Keys(source.SkillUses)) {
		for _, used := range source.SkillUses[id] {
			if !slices.Contains(available, used) {
				return skillNotInRevision(source, plan, "sources."+source.Name+".skills.uses."+id, used)
			}
		}
	}
	return nil
}

// installedSkills returns every skill the source installs, by ID, with why: chosen, the skills skills.include
// selects, and every skill they use, transitively. uses maps each skill chosen reaches that the plan's revision has
// to the skills it uses, by its syllabus-skills-used and the source's skills.uses. A skill that uses one the revision lacks,
// skills that use each other in a cycle, and a skills.uses key the source doesn't install fail as invalid
// configuration of the source's skills, naming the skills involved.
func installedSkills(source rules.Source, plan sourcePlan, chosen []string, uses map[string][]string) (map[string]skills.Installed, error) {
	location := "sources." + source.Name + ".skills"
	installed, err := skills.ResolveUses(chosen, uses)
	var missing *skills.MissingUseError
	var cycle *skills.UseCycleError
	switch {
	case errors.As(err, &missing):
		installing := missing.Skill
		if missing.Via != missing.Skill {
			installing = fmt.Sprintf("%s, which %s uses", missing.Skill, missing.Via)
		}
		return nil, &ValidationError{Location: location, Problem: fmt.Sprintf("%s uses %s, which isn't a skill in %s: %s names it in syllabus-skills-used. Code Rules installs every skill an installed skill uses, so it can't install %s. Ask the source's maintainer to correct syllabus-skills-used, or stop including %s", missing.Skill, missing.Missing, revisionName(plan.release, plan.commit), missing.Skill, installing, missing.Via)}
	case errors.As(err, &cycle):
		return nil, &ValidationError{Location: location, Problem: fmt.Sprintf("the skills the source installs use each other in a cycle: %s. Code Rules installs every skill an installed skill uses, so uses can't form a cycle. Ask the source's maintainer to correct syllabus-skills-used, or remove the use from %s.uses", strings.Join(cycle.Cycle, " -> "), location)}
	case err != nil:
		return nil, err
	}
	for _, id := range slices.Sorted(maps.Keys(source.SkillUses)) {
		if _, ok := installed[id]; !ok {
			return nil, &ValidationError{Location: location + ".uses." + id, Problem: id + " isn't a skill the source installs, so it has no uses to declare; a key of skills.uses names a skill that skills.include selects, or one those skills use. Include the skill, or remove the entry"}
		}
	}
	return installed, nil
}

// skillUses reads the SKILL.md of every skill chosen reaches in folders, the skills of the plan's commit, and returns
// what each uses: its syllabus-skills-used lines, then the skills the source's skills.uses adds. It fetches the
// SKILL.md files one level of uses at a time, so it never downloads those of skills nothing installed uses. A used
// skill folders lacks is absent from the result, for installedSkills to report. It fails when a SKILL.md it reads
// has unreadable frontmatter.
func (r *repository) skillUses(ctx context.Context, source rules.Source, folders map[string]map[string]treeEntry, chosen []string) (map[string][]string, error) {
	uses := map[string][]string{}
	for pending := chosen; len(pending) > 0; {
		entries := map[string]treeEntry{}
		for _, id := range pending {
			if _, seen := uses[id]; seen {
				continue
			}
			if entry, ok := folders[id][id+"/"+skills.File]; ok {
				entries[id] = entry
			}
		}
		if err := r.prefetch(ctx, slices.Collect(maps.Values(entries))); err != nil {
			return nil, err
		}
		files := map[string]treeEntry{}
		for _, entry := range entries {
			files[entry.name] = entry
		}
		input := newGitFiles(ctx, r, files)
		pending = nil
		for _, id := range slices.Sorted(maps.Keys(entries)) {
			file := entries[id].name
			data, err := input.ReadFile(file)
			if err != nil {
				return nil, err
			}
			metadata, err := skills.ReadMetadata(data, "vendor/"+source.Name+"/"+file)
			if err != nil {
				return nil, err
			}
			uses[id] = slices.Concat(metadata.SkillsUsed, source.SkillUses[id])
			pending = append(pending, uses[id]...)
		}
	}
	return uses, nil
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
// set, and of every skill they use, transitively, recording why each is installed. retained is the bytes the source's
// import already read, which count toward the same retained limit. It fails for a skill file that is a symbolic link
// or submodule, over the import byte limits, and as installedSkills does for the skills' uses.
func (r *repository) readSkills(ctx context.Context, source rules.Source, plan sourcePlan, isLibrary bool, retained int) (importedSkills, error) {
	result := importedSkills{files: map[string][]byte{}, executable: []string{}, skills: map[string]library.ImportedSkill{}}
	if len(source.Skills) == 0 {
		return result, nil
	}
	tree, err := r.repositoryTree(ctx, plan.commit)
	if err != nil {
		return importedSkills{}, err
	}
	folders := skillFolders(tree, isLibrary)
	available := slices.Sorted(maps.Keys(folders))
	chosen, err := chooseSkills(source, plan, available)
	if err != nil {
		return importedSkills{}, err
	}
	if err := requireConfiguredUses(source, plan, available); err != nil {
		return importedSkills{}, err
	}
	uses, err := r.skillUses(ctx, source, folders, chosen)
	if err != nil {
		return importedSkills{}, err
	}
	installed, err := installedSkills(source, plan, chosen, uses)
	if err != nil {
		return importedSkills{}, err
	}
	entries := map[string]treeEntry{}
	prefetch := []treeEntry{}
	for _, id := range slices.Sorted(maps.Keys(installed)) {
		level := ""
		if isLibrary {
			level = offered
		}
		result.skills[id] = library.ImportedSkill{Level: level, Included: installed[id].Included, UsedBy: installed[id].UsedBy}
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
	input.total = retained
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
