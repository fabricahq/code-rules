// Read the skills a library borrows from the sources in its manifest, which its skill levels name.

package rules

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/trailguide/internal/decode"
	"github.com/fabricahq/trailguide/internal/skills"
)

// Borrowed returns, sorted, the IDs of the borrowed skills the lists name. Only these borrowed skills reach projects.
func (l SkillLevels) Borrowed() []string {
	return slices.DeleteFunc(l.Listed(), func(id string) bool { return !strings.Contains(id, ":") })
}

// BorrowedSource is a source a library's manifest names that it borrows skills from: the repository and the ref
// that each project resolves for itself when it updates.
type BorrowedSource struct {
	Repository string `json:"repository"`
	// Ref is the zero GitRef when the library follows the repository's newest library release.
	Ref GitRef `json:"ref,omitzero"`
}

// validateBorrowedSkill checks that id, listed at location, names a borrowed skill: the name of one of sources, a
// colon, and the skill's ID in that source's repository, such as mattpocock:skills/to-spec.
func validateBorrowedSkill(id, location string, sources []string) error {
	source, skill, _ := strings.Cut(id, ":")
	if !sourceNamePattern.MatchString(source) {
		return invalid(location, "expected a borrowed skill as the name of a source in sources, a colon, and the skill's ID in that source's repository, such as mattpocock:skills/to-spec, got "+decode.Quote(id))
	}
	if err := skills.ValidateID(skill, location); err != nil {
		return err
	}
	if !slices.Contains(sources, source) {
		return invalid(location, id+" borrows from "+source+", which isn't a source in sources; add sources."+source+" with the repository the skill comes from, and a ref unless the repository publishes library releases")
	}
	return nil
}

// borrowedSources returns, by name, the sources of sources that the library's skills borrow from, or nil when they
// borrow from none.
func borrowedSources(levels SkillLevels, sources []Source) map[string]BorrowedSource {
	if len(levels.Borrowed()) == 0 {
		return nil
	}
	result := map[string]BorrowedSource{}
	for _, id := range levels.Borrowed() {
		name, _, _ := strings.Cut(id, ":")
		index := slices.IndexFunc(sources, func(source Source) bool { return source.Name == name })
		result[name] = BorrowedSource{Repository: sources[index].Repository, Ref: sources[index].Ref}
	}
	return result
}

// borrowedNames returns, sorted, the names of the sources the raw library section's skills borrow from, without
// validating anything else, so the published manifest can carry those sources.
func borrowedNames(section json.RawMessage) []string {
	var library struct {
		Skills map[string][]string `json:"skills"`
	}
	if json.Unmarshal(section, &library) != nil {
		return []string{}
	}
	names := map[string]bool{}
	for _, list := range library.Skills {
		for _, id := range list {
			if name, _, borrowed := strings.Cut(id, ":"); borrowed {
				names[name] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(names))
}
