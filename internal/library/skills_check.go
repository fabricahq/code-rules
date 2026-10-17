// Check the frontmatter of the skills a library authors: each folder's name, and the metadata keys Trail Guide defines.

package library

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/trailguide/internal/filetxn"
	"github.com/fabricahq/trailguide/internal/rules"
	"github.com/fabricahq/trailguide/internal/skills"
)

// skillsDirectory is where a library's skills live in its repository.
const skillsDirectory = ".trailguide/" + skills.Directory

// skillProblems returns every problem in the skills tree holds, by path in the library's skills/ folder, in path
// order, each starting with the file or folder it's about: a file directly in skills/, a folder whose name breaks the
// skill name grammar or that has no SKILL.md, frontmatter skills.ReadMetadata or skills.Check rejects, and a cited
// rule that isn't one of current, the library's rules, naming the replacement when the library's release history or
// a pending change note retired it, and a skill a skill uses that isn't a folder of skills/ with a SKILL.md or, when
// the line names a source, one of curated, the curated skills the library's levels list. A nil tree has no skills.
func skillProblems(tree *filetxn.Tree, current, curated []string, changes libraryChanges) []string {
	problems := []string{}
	if tree == nil {
		return problems
	}
	folders := map[string]bool{}
	for _, name := range slices.Sorted(maps.Keys(tree.Files)) {
		folder, file, inFolder := strings.Cut(name, "/")
		if !inFolder {
			problems = append(problems, skills.Directory+"/"+name+": skills/ holds only skill folders")
			continue
		}
		folders[folder] = folders[folder] || file == skills.File
	}
	for _, directory := range tree.Directories {
		if _, listed := folders[directory]; !listed && !strings.Contains(directory, "/") {
			folders[directory] = false
		}
	}
	_, retiring := changes.namedRules()
	for _, folder := range slices.Sorted(maps.Keys(folders)) {
		location := skills.AuthoredID(folder)
		switch {
		case !skills.ValidName(folder):
			problems = append(problems, location+": a skill name uses 1 to 64 lowercase letters, digits, and hyphens, without a leading, trailing, or doubled hyphen")
			continue
		case !folders[folder]:
			problems = append(problems, location+": expected SKILL.md in the skill folder")
			continue
		}
		location += "/" + skills.File
		metadata, err := skills.ReadMetadata(tree.Files[folder+"/"+skills.File], location)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		for _, problem := range skills.Check(folder, metadata, false) {
			problems = append(problems, location+": "+problem)
		}
		for _, id := range metadata.RulesUsed {
			if problem := citedRuleProblem(id, current, changes.history, retiring); problem != "" {
				problems = append(problems, location+": "+problem)
			}
		}
		missing := []string{}
		for _, id := range metadata.SkillsUsed {
			// A skill in the library is a folder with a SKILL.md, and a curated one is one its levels list, which
			// requireCuratedSkills checks its source has; skills.Check reports a use that isn't a skill ID.
			if !skills.ValidSkillCitation(id) || slices.Contains(missing, id) {
				continue
			}
			if citation := skills.ParseCitation(id); citation.Source != "" && !slices.Contains(curated, id) {
				missing = append(missing, id)
				problems = append(problems, fmt.Sprintf("%s: %s names %s, which isn't a skill the library curates; list it in library.skills, at any level, so projects that install the skill can install it too", location, skills.SkillsUsedKey, id))
			} else if name, _ := strings.CutPrefix(id, skills.Directory+"/"); citation.Source == "" && !folders[name] {
				missing = append(missing, id)
				problems = append(problems, fmt.Sprintf("%s: %s names %s, which isn't a skill in this library", location, skills.SkillsUsedKey, id))
			}
		}
	}
	return problems
}

// citedRuleProblem returns why id, a line of trailguide-rules-used, doesn't name a current rule of the library, or empty when
// it does or isn't a rule ID, which skills.Check reports instead. retiring holds the pending change notes'
// retirements by rule.
func citedRuleProblem(id string, current []string, history releaseHistory, retiring map[string][]rules.NoteChange) string {
	if slices.Contains(current, id) || !skills.ValidRuleCitation(id, false) {
		return ""
	}
	replacement, retired := "", false
	if _, ok := history.retired[id]; ok {
		replacement, retired = history.replacedBy[id], true
	}
	for _, change := range retiring[id] {
		replacement, retired = change.ReplacedBy, true
	}
	switch {
	case retired && replacement != "":
		return fmt.Sprintf("%s names %s, which is retired and replaced by %s; cite the replacement", skills.RulesUsedKey, id, replacement)
	case retired:
		return fmt.Sprintf("%s names %s, which is retired; remove it", skills.RulesUsedKey, id)
	}
	return fmt.Sprintf("%s names %s, which isn't a rule in this library", skills.RulesUsedKey, id)
}
