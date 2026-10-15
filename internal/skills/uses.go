// Resolve which skills installing a selection brings in: the selected skills, and every skill they use, transitively.

package skills

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Installed says why one skill is installed: a selection includes it, other installed skills use it, or both.
type Installed struct {
	Included bool
	// UsedBy lists, sorted, the installed skills that use this one; it is empty, never nil, when none does, which
	// happens only for an included skill.
	UsedBy []string
}

// MissingUseError reports a skill that uses a skill the repository doesn't have.
type MissingUseError struct {
	// Skill uses Missing.
	Skill, Missing string
	// Via is the included skill whose uses reach Skill: Skill itself when the selection includes it.
	Via string
}

// Error names the skill and the skill it uses.
func (e *MissingUseError) Error() string {
	return fmt.Sprintf("%s uses %s, which isn't a skill", e.Skill, e.Missing)
}

// UseCycleError reports skills that use each other in a cycle, directly or through other skills.
type UseCycleError struct {
	// Cycle lists the skills in the order each uses the next, starting and ending with the same skill, such as
	// [skills/a skills/b skills/a]; a skill that uses itself appears twice.
	Cycle []string
}

// Error names the skills in the cycle.
func (e *UseCycleError) Error() string {
	return "skills use each other in a cycle: " + strings.Join(e.Cycle, " -> ")
}

// ResolveUses returns every skill that installing included brings in, by ID, with why each is installed: the
// included skills, every skill they use, and every skill those use in turn. uses maps the ID of every skill the
// repository has, and no other, to the IDs of the skills it uses; a repeated ID counts once. Every included ID must be
// a key of uses. It visits included skills, and the skills each uses, in sorted order, and fails with the first
// *MissingUseError, for a used ID that isn't a key of uses, or *UseCycleError, for skills that use each other, it
// meets. Only skills the selection reaches are checked. It leaves its inputs unchanged.
func ResolveUses(included []string, uses map[string][]string) (map[string]Installed, error) {
	result := map[string]Installed{}
	for _, id := range included {
		if _, ok := uses[id]; !ok {
			return nil, fmt.Errorf("resolve skill uses: included skill %s isn't in uses", id)
		}
		result[id] = Installed{Included: true, UsedBy: []string{}}
	}
	// done holds each skill whose uses, transitively, are resolved; a skill on path is being resolved.
	done := map[string]bool{}
	var visit func(id, via string, path []string) error
	visit = func(id, via string, path []string) error {
		if done[id] {
			return nil
		}
		if at := slices.Index(path, id); at >= 0 {
			return &UseCycleError{Cycle: append(slices.Clone(path[at:]), id)}
		}
		path = append(path, id)
		used := slices.Clone(uses[id])
		slices.Sort(used)
		for _, next := range slices.Compact(used) {
			if _, ok := uses[next]; !ok {
				return &MissingUseError{Skill: id, Missing: next, Via: via}
			}
			entry, ok := result[next]
			if !ok {
				entry = Installed{UsedBy: []string{}}
			}
			entry.UsedBy = append(entry.UsedBy, id)
			result[next] = entry
			if err := visit(next, via, path); err != nil {
				return err
			}
		}
		done[id] = true
		return nil
	}
	for _, id := range slices.Sorted(maps.Keys(result)) {
		if result[id].Included {
			if err := visit(id, id, nil); err != nil {
				return nil, err
			}
		}
	}
	for id, entry := range result {
		slices.Sort(entry.UsedBy)
		result[id] = entry
	}
	return result, nil
}
