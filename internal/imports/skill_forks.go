// Check the commits that skill replacements are based on, and preview the changes a source made to replaced skills.

package imports

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// ReplacedSkillUpdate is a row of the update preview for a skill the project replaces with an authored skill, listed
// when the source's copy of the skill at the revision the update takes skills from differs from the copy the
// replacement is based on.
type ReplacedSkillUpdate struct {
	ID string `json:"id"`
	// LocalSkill is the authored skill that replaces it, as the exclusion's replacedBy names it, which is also its
	// folder relative to the Syllabus directory.
	LocalSkill string `json:"localSkill"`
	// BasedOn is the commit the replacement was copied from, as the exclusion records it, or empty when it records
	// none; the row then compares with From.
	BasedOn string `json:"basedOn,omitempty"`
	// From is the commit the source record locks, and To the commit the update takes the source's skills from.
	From string `json:"from"`
	To   string `json:"to"`
	// Files lists, sorted and relative to the skill's folder, the files that differ between the compared copies: in
	// contents, presence, or executable bit.
	Files []string `json:"files"`
}

// requireSkillBasedOn checks that the repository has the commit each of the source's skill replacements is based
// on, failing with code version-not-found when it doesn't, and that the skill is in that commit, failing validation
// when it isn't.
func (r *repository) requireSkillBasedOn(ctx context.Context, source rules.Source) error {
	for _, id := range slices.Sorted(maps.Keys(source.SkillExclude)) {
		based := source.SkillExclude[id].BasedOn
		if based.IsZero() {
			continue
		}
		if _, err := r.skillEntries(ctx, source, id, based.Canonical()); err != nil {
			return err
		}
	}
	return nil
}

// skillEntries returns the tree entries of skill id at commit, by path in the skill's folder, fetching the commit
// when it isn't present. id's exclusion bases its replacement on commit, which it names in failures: code
// version-not-found when the repository doesn't have the commit, and a validation error when the commit doesn't
// have the skill.
func (r *repository) skillEntries(ctx context.Context, source rules.Source, id, commit string) (map[string]treeEntry, error) {
	where := "sources." + source.Name + ".skills.exclude." + id + ".basedOn"
	if err := r.fetchCommits(ctx, []string{commit}, fmt.Sprintf("%s: the repository has no commit %s; check basedOn, the full commit SHA that the replacement was copied from.", where, commit)); err != nil {
		return nil, err
	}
	entries, found, err := r.skillAt(ctx, id, commit)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &ValidationError{Location: where, Problem: fmt.Sprintf("commit %s has no skill %s; check basedOn, the full commit SHA that the replacement was copied from", commit, id)}
	}
	return entries, nil
}

// skillAt returns the tree entries of skill id at commit, which must be present, by path in the skill's folder,
// and whether the commit has the skill.
func (r *repository) skillAt(ctx context.Context, id, commit string) (map[string]treeEntry, bool, error) {
	isLibrary, err := r.isLibrary(ctx, commit)
	if err != nil {
		return nil, false, err
	}
	tree, err := r.repositoryTree(ctx, commit)
	if err != nil {
		return nil, false, err
	}
	folder, found := skillFolders(tree, isLibrary)[id]
	if !found {
		return nil, false, nil
	}
	entries := map[string]treeEntry{}
	for name, entry := range folder {
		entries[strings.TrimPrefix(name, id+"/")] = entry
	}
	return entries, true, nil
}

// replacedSkills returns a preview row, sorted by ID, for each skill the source replaces whose copy at commit, the
// revision the update takes the source's skills from, differs from the copy its replacement is based on: the
// exclusion's basedOn, or else the commit recorded locks. A replacement without basedOn in a source with no record
// has nothing to compare with, and a skill commit lacks is left to the import, which fails for it.
func (r *repository) replacedSkills(ctx context.Context, source rules.Source, recorded *library.Snapshot, commit string) ([]ReplacedSkillUpdate, error) {
	rows := []ReplacedSkillUpdate{}
	from := ""
	if recorded != nil {
		from = recorded.Commit
	}
	for _, id := range slices.Sorted(maps.Keys(source.SkillExclude)) {
		exclusion := source.SkillExclude[id]
		if exclusion.ReplacedBy == "" {
			continue
		}
		before, baseline, err := r.replacementBaseline(ctx, source, id, from)
		if err != nil {
			return nil, err
		}
		if baseline == "" || baseline == commit {
			continue
		}
		if err := r.fetchCommits(ctx, []string{commit}, fmt.Sprintf("The repository no longer has commit %s, which the update takes source %s's skills from. Run syllabus project update again.", commit, source.Name)); err != nil {
			return nil, err
		}
		after, found, err := r.skillAt(ctx, id, commit)
		if err != nil {
			return nil, err
		}
		if files := differentEntries(before, after); found && len(files) > 0 {
			rows = append(rows, ReplacedSkillUpdate{ID: id, LocalSkill: exclusion.ReplacedBy, BasedOn: exclusion.BasedOn.Canonical(), From: from, To: commit, Files: files})
		}
	}
	return rows, nil
}

// replacementBaseline returns the copy of skill id that the source's replacement of it is based on, by path in the
// skill's folder, and its commit: the exclusion's basedOn, checked as requireSkillBasedOn checks it, or else from,
// the commit the source record locks, or "" when there is neither. A skill from lacks has no files.
func (r *repository) replacementBaseline(ctx context.Context, source rules.Source, id, from string) (map[string]treeEntry, string, error) {
	if based := source.SkillExclude[id].BasedOn; !based.IsZero() {
		entries, err := r.skillEntries(ctx, source, id, based.Canonical())
		return entries, based.Canonical(), err
	}
	if from == "" {
		return nil, "", nil
	}
	if err := r.fetchCommits(ctx, []string{from}, fmt.Sprintf("The repository no longer has commit %s, which vendor/%s/_source.json records. Run syllabus project sync to check the source.", from, source.Name)); err != nil {
		return nil, "", err
	}
	entries, _, err := r.skillAt(ctx, id, from)
	return entries, from, err
}

// differentEntries returns, sorted, the paths whose entries differ between before and after: present in only one,
// or with another object or mode, so a file that only becomes executable differs too.
func differentEntries(before, after map[string]treeEntry) []string {
	names := map[string]bool{}
	for _, entries := range []map[string]treeEntry{before, after} {
		for name := range entries {
			names[name] = true
		}
	}
	files := []string{}
	for _, name := range slices.Sorted(maps.Keys(names)) {
		old, had := before[name]
		current, has := after[name]
		if !had || !has || old.object != current.object || old.mode != current.mode {
			files = append(files, name)
		}
	}
	return files
}
