// Apply the skills each source excludes, and pair each authored replacement with the skill it replaces.

package build

import (
	"fmt"
	"maps"
	"slices"

	"github.com/fabricahq/code-rules/internal/rules"
)

// skillReplacement is what an authored skill replaces: a source's skill, which the project excludes and installs
// the authored skill in place of.
type skillReplacement struct {
	// Upstream is the replaced skill as its source supplies it, without its files.
	Upstream resolvedSkill
	// Reason is the exclusion's reason, and BasedOn the commit the replacement was copied from, or the zero GitRef.
	Reason  string
	BasedOn rules.GitRef
}

// excludedSkill is a source's skill that the project leaves out without a replacement.
type excludedSkill struct {
	// ID is the skill's project ID, source:folder.
	ID, Reason string
}

// skillReplacements returns, by authored skill ID, the skill each authored skill in local replaces. It fails when an
// exclusion names a skill its source's snapshot doesn't hold, when a replacement isn't an authored skill, and when
// one authored skill replaces two skills.
func skillReplacements(config rules.Configuration, libraries map[string]Library, local map[string]SkillFiles) (map[string]skillReplacement, error) {
	result := map[string]skillReplacement{}
	// replaces maps each authored skill that replaces one to the configuration entry that names it.
	replaces := map[string]string{}
	for _, source := range config.Sources {
		snapshot := libraries[source.Name].Snapshot
		for _, id := range slices.Sorted(maps.Keys(source.SkillExclude)) {
			exclusion := source.SkillExclude[id]
			where := "sources." + source.Name + ".skills.exclude." + id
			if _, installs := snapshot.Skills[id]; !installs {
				return nil, invalid(where, fmt.Sprintf("the source doesn't install skill %s, so there's nothing to exclude; name a skill its skills.include selects, or remove the entry", id))
			}
			if exclusion.ReplacedBy == "" {
				continue
			}
			if _, authored := local[exclusion.ReplacedBy]; !authored {
				return nil, invalid(where+".replacedBy", fmt.Sprintf("the project has no authored skill %s; add .syllabus/%s/SKILL.md, or remove replacedBy to exclude the skill without a replacement", exclusion.ReplacedBy, exclusion.ReplacedBy))
			}
			if other, taken := replaces[exclusion.ReplacedBy]; taken {
				return nil, invalid(where+".replacedBy", fmt.Sprintf("%s already replaces the skill %s names; each authored skill replaces one skill", exclusion.ReplacedBy, other))
			}
			replaces[exclusion.ReplacedBy] = where
			upstream := resolvedSkill{ID: source.Name + ":" + id, Source: source.Name, Folder: id, Repository: source.Repository, Ref: source.Ref.String(), Commit: snapshot.Commit, Release: snapshot.Release, NonLibrary: snapshot.NonLibrary}
			result[exclusion.ReplacedBy] = skillReplacement{Upstream: upstream, Reason: exclusion.Reason, BasedOn: exclusion.BasedOn}
		}
	}
	return result, nil
}

// excludedSkills returns, in source order and then by ID, the skills of sources that the project excludes without a
// replacement.
func excludedSkills(sources []resolvedSource) []excludedSkill {
	result := []excludedSkill{}
	for _, source := range sources {
		for _, id := range source.Skills {
			if exclusion, excluded := source.Configured.SkillExclude[id]; excluded && exclusion.ReplacedBy == "" {
				result = append(result, excludedSkill{ID: source.Name + ":" + id, Reason: exclusion.Reason})
			}
		}
	}
	return result
}
