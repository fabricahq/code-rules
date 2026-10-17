// Read the skills a library curates from the sources in its manifest, which its skill levels name.

package rules

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/trailguide/internal/decode"
	"github.com/fabricahq/trailguide/internal/skills"
)

// Curated returns, sorted, the IDs of the curated skills the lists name. Only these curated skills reach projects.
func (l SkillLevels) Curated() []string {
	return slices.DeleteFunc(l.Listed(), func(id string) bool { return !strings.Contains(id, ":") })
}

// CuratedSource is a source a library's manifest names that it curates skills from: the repository and the ref
// that each project resolves for itself when it updates.
type CuratedSource struct {
	Repository string `json:"repository"`
	// Ref is the zero GitRef when the library follows the repository's newest library release.
	Ref GitRef `json:"ref,omitzero"`
}

// validateCuratedSkill checks that id, listed at location, names a curated skill: the name of one of sources, a
// colon, and the skill's ID in that source's repository, such as mattpocock:skills/to-spec.
func validateCuratedSkill(id, location string, sources []string) error {
	source, skill, _ := strings.Cut(id, ":")
	if !sourceNamePattern.MatchString(source) {
		return invalid(location, "expected a curated skill as the name of a source in sources, a colon, and the skill's ID in that source's repository, such as mattpocock:skills/to-spec, got "+decode.Quote(id))
	}
	if err := skills.ValidateID(skill, location); err != nil {
		return err
	}
	if !slices.Contains(sources, source) {
		return invalid(location, id+" comes from "+source+", which isn't a source in sources; add sources."+source+" with the repository the skill comes from, and a ref unless the repository publishes library releases")
	}
	return nil
}

// curatedSources returns, by name, the sources of sources that the library's skills curate from, or nil when they
// curate from none.
func curatedSources(levels SkillLevels, sources []Source) map[string]CuratedSource {
	if len(levels.Curated()) == 0 {
		return nil
	}
	result := map[string]CuratedSource{}
	for _, id := range levels.Curated() {
		name, _, _ := strings.Cut(id, ":")
		index := slices.IndexFunc(sources, func(source Source) bool { return source.Name == name })
		result[name] = CuratedSource{Repository: sources[index].Repository, Ref: sources[index].Ref}
	}
	return result
}

// curatedNames returns, sorted, the names of the sources the raw library section's skills curate from, without
// validating anything else, so the published manifest can carry those sources.
func curatedNames(section json.RawMessage) []string {
	var library struct {
		Skills map[string][]string `json:"skills"`
	}
	if json.Unmarshal(section, &library) != nil {
		return []string{}
	}
	names := map[string]bool{}
	for _, list := range library.Skills {
		for _, id := range list {
			if name, _, curated := strings.Cut(id, ":"); curated {
				names[name] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(names))
}
