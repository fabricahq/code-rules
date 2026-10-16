// Parse a library's skill levels: which of its skills projects that use it must, should, or may install.

package rules

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/fabricahq/syllabus/internal/decode"
	"github.com/fabricahq/syllabus/internal/skills"
)

// SkillLevels lists, sorted, the IDs of the skills at each level the manifest names: the library's own skills,
// skills/<name>, and the skills it borrows, the name of a source in its sources, a colon, and the skill's ID in that
// source's repository, such as mattpocock:skills/to-spec. Each list is empty, never nil, when the manifest names
// none, and a skill is in at most one list. Whether each own skill exists is checked against the library's skills/
// folder, and whether each borrowed one exists when a project imports it.
type SkillLevels struct {
	Required    []string `json:"required"`
	Recommended []string `json:"recommended"`
	Offered     []string `json:"offered"`
}

// Level returns the level of the library's skill id: the level whose list names it, or offered, the level of every
// own skill no list names. A borrowed skill no list names doesn't reach projects; see Borrowed.
func (l SkillLevels) Level(id string) string {
	switch {
	case slices.Contains(l.Required, id):
		return skills.LevelRequired
	case slices.Contains(l.Recommended, id):
		return skills.LevelRecommended
	}
	return skills.LevelOffered
}

// Listed returns, sorted, the IDs of every skill a list names.
func (l SkillLevels) Listed() []string {
	listed := slices.Concat(l.Required, l.Recommended, l.Offered)
	slices.Sort(listed)
	return listed
}

// parseSkillLevels validates library.skills at location: an object of required, recommended, and offered lists of
// skill IDs, each listed once, each borrowed one from one of sources, the names the configuration's sources declare.
// A missing object lists none.
func parseSkillLevels(input json.RawMessage, location string, sources []string) (SkillLevels, error) {
	result := SkillLevels{Required: []string{}, Recommended: []string{}, Offered: []string{}}
	if input == nil {
		return result, nil
	}
	fields, err := decode.Object(input, location)
	if err != nil {
		return SkillLevels{}, err
	}
	if err := decode.KnownFields(fields, skills.Levels, location); err != nil {
		return SkillLevels{}, err
	}
	seen := map[string]string{}
	for _, level := range skills.Levels {
		raw, ok := fields[level]
		if !ok {
			continue
		}
		where := location + "." + level
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return SkillLevels{}, invalid(where, "expected a list of the library's skill IDs, such as [skills/build-feature]")
		}
		ids := []string{}
		for i, item := range items {
			at := fmt.Sprintf("%s[%d]", where, i)
			id, err := decode.Text(item, at)
			if err != nil {
				return SkillLevels{}, err
			}
			if _, _, borrowed := strings.Cut(id, ":"); borrowed {
				if err := validateBorrowedSkill(id, at, sources); err != nil {
					return SkillLevels{}, err
				}
			} else if name, authored := strings.CutPrefix(id, skills.Directory+"/"); !authored || !skills.ValidName(name) {
				return SkillLevels{}, invalid(at, "expected the ID of one of the library's skills, skills/<name>, the folder's path in .syllabus/, such as skills/build-feature, got "+decode.Quote(id))
			}
			if other, listed := seen[id]; listed {
				if other == level {
					return SkillLevels{}, invalid(at, id+" is listed more than once; list each skill once")
				}
				return SkillLevels{}, invalid(at, id+" is already "+other+"; give each skill one level")
			}
			seen[id] = level
			ids = append(ids, id)
		}
		slices.Sort(ids)
		switch level {
		case skills.LevelRequired:
			result.Required = ids
		case skills.LevelRecommended:
			result.Recommended = ids
		default:
			result.Offered = ids
		}
	}
	return result, nil
}

// RequiredSkillProblem explains why a project can't exclude or fork skill id, which the library that source names
// requires in release, or 0 for a revision other than a library release.
func RequiredSkillProblem(source, id string, release int) string {
	at := ""
	if release != 0 {
		at = fmt.Sprintf(" in release %d", release)
	}
	return fmt.Sprintf("%s's library requires %s%s, so every project that uses the library installs it, and the project can't exclude or fork it. To diverge from it, stop using sources.%s, or ask the library's maintainer to make the skill recommended or offered", source, id, at, source)
}
