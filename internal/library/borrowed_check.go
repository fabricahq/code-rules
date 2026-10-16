// Check the skills a library borrows against the repositories it borrows them from, at the refs it declares.

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

// BorrowedReader reads the source a library borrows skills from, as its manifest declares it under name: it
// resolves the source's ref, or its newest library release without one, and returns the IDs of the skills the
// repository has there, sorted, and the revision's name for messages, such as main or release/3.
type BorrowedReader func(ctx context.Context, name string, source rules.BorrowedSource) ([]string, string, error)

// requireBorrowedSkills fails when manifest, the library's configuration, lists a borrowed skill that its source
// doesn't have at the ref the library declares, or when two skills it offers projects, its own in authored, its
// skills/ folder, and the borrowed ones it lists, have the same install name, which agents find skills by. read
// reads each borrowed source; nil skips the check.
func requireBorrowedSkills(ctx context.Context, manifest []byte, authored *filetxn.Tree, read BorrowedReader) error {
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
	listed := declaration.Skills.Borrowed()
	sources := map[string][]string{}
	for _, id := range listed {
		source, _, _ := SplitBorrowed(id)
		sources[source] = nil
	}
	revisions := map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(sources)) {
		if sources[name], revisions[name], err = read(ctx, name, declaration.Borrowed[name]); err != nil {
			return err
		}
	}
	for _, id := range listed {
		source, skill, _ := SplitBorrowed(id)
		if !slices.Contains(sources[source], skill) {
			field := "library.skills." + declaration.Skills.Level(id)
			return &ValidationError{Location: manifestPath + ": " + field, Problem: field + " names " + id + ", which " + source + " doesn't have at " + revisions[source] + "; check the skill ID, or set sources." + source + ".ref to a revision that has it"}
		}
	}
	ids := slices.Concat(slices.Sorted(maps.Values(names)), listed)
	seen := map[string]string{}
	for _, id := range ids {
		_, skill, _ := SplitBorrowed(id)
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
