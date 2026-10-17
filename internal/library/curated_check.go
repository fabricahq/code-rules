// Check the skills a library curates against the repositories it curates them from, at the refs it declares.

package library

import (
	"context"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/trailguide/internal/filetxn"
	"github.com/fabricahq/trailguide/internal/rules"
	"github.com/fabricahq/trailguide/internal/skills"
)

// CuratedReader reads the source a library curates skills from, as its manifest declares it under name: it
// resolves the source's ref, or its newest library release without one, and returns the IDs of the skills the
// repository has there, sorted, and the revision's name for messages, such as main or release/3.
type CuratedReader func(ctx context.Context, name string, source rules.CuratedSource) ([]string, string, error)

// requireCuratedSkills fails when manifest, the library's configuration, lists a curated skill that its source
// doesn't have at the ref the library declares, or when two skills it offers projects, its own in authored, its
// skills/ folder, and the curated ones it lists, have the same install name, which agents find skills by. read
// reads each curated source; nil skips the check.
func requireCuratedSkills(ctx context.Context, manifest []byte, authored *filetxn.Tree, read CuratedReader) error {
	declaration, err := rules.ParseLibraryManifest(manifest, manifestPath)
	if err != nil || declaration == nil || read == nil {
		return err
	}
	// Every skill the library authors reaches projects, since one no level lists is offered.
	names := map[string]string{}
	if authored != nil {
		for file := range authored.Files {
			if name, found := strings.CutSuffix(file, "/"+skills.File); found && !strings.Contains(name, "/") {
				names[name] = skills.AuthoredID(name)
			}
		}
	}
	listed := declaration.Skills.Curated()
	sources := map[string][]string{}
	for _, id := range listed {
		source, _, _ := SplitCurated(id)
		sources[source] = nil
	}
	revisions := map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(sources)) {
		if sources[name], revisions[name], err = read(ctx, name, declaration.Curated[name]); err != nil {
			return err
		}
	}
	for _, id := range listed {
		source, skill, _ := SplitCurated(id)
		if !slices.Contains(sources[source], skill) {
			field := "library.skills." + declaration.Skills.Level(id)
			return &ValidationError{Location: manifestPath + ": " + field, Problem: field + " names " + id + ", which " + source + " doesn't have at " + revisions[source] + "; check the skill ID, or set sources." + source + ".ref to a revision that has it"}
		}
	}
	ids := slices.Concat(slices.Sorted(maps.Values(names)), listed)
	seen := map[string]string{}
	for _, id := range ids {
		_, skill, _ := SplitCurated(id)
		if !strings.Contains(id, ":") {
			skill = id
		}
		name := path.Base(skill)
		if other, clash := seen[name]; clash {
			return &ValidationError{Location: manifestPath + ": library.skills", Problem: "library.skills lists " + other + " and " + id + ", which both install as " + name + "; list one of them, or rename the folder of your own skill"}
		}
		seen[name] = id
	}
	return nil
}
