// Fetch the skills a library borrows from other repositories, from those repositories, at revisions each project
// resolves for itself.

package imports

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/syllabus/internal/library"
	"github.com/fabricahq/syllabus/internal/librarytree"
	"github.com/fabricahq/syllabus/internal/rules"
	"github.com/fabricahq/syllabus/internal/skills"
)

// borrowing says which revision of each source a library borrows from an import takes skills from.
type borrowing struct {
	// recorded holds each source's snapshot from the last sync, keyed by source name. Its borrowed revisions stay
	// while the library still names the same repository and ref for them, since sync never moves a source.
	recorded map[string]library.Snapshot
	// planned maps the name of a library source to the borrowed revisions an update planned for it, by borrowed
	// source name, which the import takes as they are.
	planned map[string]map[string]library.BorrowedSource
}

// borrowedSkill is a borrowed skill a project takes from a library source: its ID in the library, such as
// mattpocock:skills/to-spec, and the level the library lists it at.
type borrowedSkill struct {
	id, level string
}

// borrowedSkills returns, by borrowed source name, the borrowed skills source takes from manifest, its library's:
// every required one, the recommended ones when skills.include names recommended, and each offered one that it
// names or includes with "*". It fails when skills.include or skills.exclude names a borrowed skill the library
// doesn't list, since a borrowed skill reaches projects only through the library's lists, and when skills.exclude
// names a required one, since a project that uses the library can't diverge from it.
func borrowedSkills(manifest *rules.Library, source rules.Source, release int) (map[string][]borrowedSkill, error) {
	listed := rules.SkillLevels{}
	if manifest != nil {
		listed = manifest.Skills
	}
	borrowed := listed.Borrowed()
	for _, field := range []struct {
		name string
		ids  []string
	}{{"include", source.Skills}, {"exclude", slices.Sorted(maps.Keys(source.SkillExclude))}} {
		for _, id := range field.ids {
			if _, _, isBorrowed := library.SplitBorrowed(id); isBorrowed && !slices.Contains(borrowed, id) {
				return nil, &ValidationError{Location: "sources." + source.Name + ".skills." + field.name, Problem: fmt.Sprintf("the library doesn't list %s at any level, and a library's borrowed skills reach projects only through its lists; check the skill ID, or ask the library's maintainer to list it", id)}
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(source.SkillExclude)) {
		if slices.Contains(listed.Required, id) {
			return nil, &ValidationError{Location: "sources." + source.Name + ".skills.exclude." + id, Problem: rules.RequiredSkillProblem(source.Name, id, release)}
		}
	}
	result := map[string][]borrowedSkill{}
	for _, id := range borrowed {
		level := listed.Level(id)
		switch {
		case level == skills.LevelRequired:
		case slices.Contains(source.Skills, skills.Wildcard) || slices.Contains(source.Skills, id):
		case level == skills.LevelRecommended && slices.Contains(source.Skills, skills.Recommended):
		default:
			continue
		}
		name, _, _ := library.SplitBorrowed(id)
		result[name] = append(result[name], borrowedSkill{id: id, level: level})
	}
	return result, nil
}

// borrowSkills adds to imported, source's import, the borrowed skills the source takes from its library, with their
// files from the repositories the library borrows from, and records each borrowed source's revision. all holds every
// source's import, so a project source that declares a borrowed repository supplies its revision instead of the
// library's ref.
func borrowSkills(ctx context.Context, configuration rules.Configuration, source rules.Source, imported *Library, all map[string]Library, borrow borrowing, options Options) error {
	if imported.Snapshot.NonLibrary {
		return nil
	}
	manifest, err := rules.ParseLibraryManifest(imported.Snapshot.Files[librarytree.Manifest], source.Name+"/"+librarytree.RepositoryPath(librarytree.Manifest))
	if err != nil {
		return err
	}
	selected, err := borrowedSkills(manifest, source, imported.Snapshot.Release)
	if err != nil {
		return err
	}
	// The skills the library lists are what it offers of each borrowed repository, whether the project takes them or not.
	for _, id := range manifest.Skills.Borrowed() {
		imported.Snapshot.AvailableSkills[id] = library.ImportedSkill{Level: manifest.Skills.Level(id)}
	}
	for _, name := range slices.Sorted(maps.Keys(selected)) {
		lender := lendingSource{library: source.Name, name: name, declared: manifest.Borrowed[name]}
		lender.override = overridingSource(configuration, source.Name, lender.declared.Repository)
		if lender.override != nil {
			lender.overridden = all[lender.override.Name].Snapshot
		}
		recorded := borrow.recorded[source.Name].Borrowed[name]
		planned, isPlanned := borrow.planned[source.Name][name]
		if err := lender.borrow(ctx, imported, selected[name], recorded, planned, isPlanned, options); err != nil {
			return err
		}
	}
	slices.Sort(imported.Snapshot.Executable)
	return nil
}

// overridingSource returns the project's source, other than the library source named library, that declares
// repository, or nil when none does. Repository identity, not the source's name, decides the match.
func overridingSource(configuration rules.Configuration, library, repository string) *rules.Source {
	identity := repositoryIdentity(repository)
	for i, source := range configuration.Sources {
		if source.Name != library && identity != "" && repositoryIdentity(source.Repository) == identity {
			return &configuration.Sources[i]
		}
	}
	return nil
}

// repositoryIdentity returns the identity of a repository address, or "" for one that doesn't parse.
func repositoryIdentity(address string) string {
	parsed, err := rules.ParseRepository(quote(address), "repository")
	if err != nil {
		return ""
	}
	return parsed.Identity
}

// lendingSource is one source a library borrows skills from, as one project resolves it.
type lendingSource struct {
	// library names the project's library source, and name the borrowed source in the library's manifest, which
	// declares it as declared.
	library, name string
	declared      rules.BorrowedSource
	// override is the project's own source that declares the same repository, and overridden its import; override
	// is nil when no source does.
	override   *rules.Source
	overridden library.Snapshot
}

// borrow fetches the lending source's revision and adds the files of chosen, its skills the project takes, to
// imported. The revision is the overriding source's, when there is one; else planned, when an update planned one;
// else recorded's, when it records the ref the library names; else the library's ref, resolved now.
func (l lendingSource) borrow(ctx context.Context, imported *Library, chosen []borrowedSkill, recorded library.BorrowedSource, planned library.BorrowedSource, isPlanned bool, options Options) (err error) {
	ctx, cancel, err := withTimeout(ctx, options)
	if err != nil {
		return err
	}
	defer cancel()
	address := l.declared.Repository
	if l.override != nil {
		address = l.override.Repository
	}
	repo, err := openRepository(ctx, rules.Source{Name: l.name, Repository: address}, options)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := repo.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	revision, kept, err := l.revision(ctx, repo, recorded, planned, isPlanned)
	if err != nil {
		return err
	}
	tree, err := repo.repositoryTree(ctx, revision.Commit)
	if err != nil {
		return err
	}
	folders := skillFolders(tree, !revision.NonLibrary)
	ids, levels := []string{}, map[string]string{}
	for _, skill := range chosen {
		_, id, _ := library.SplitBorrowed(skill.id)
		if _, ok := folders[id]; !ok {
			return l.missing(skill.id, revision, kept)
		}
		ids, levels[id] = append(ids, id), skill.level
	}
	// A borrowed skill brings the skills it uses from its repository, as a source's own skills do.
	lender := rules.Source{Name: l.library}
	plan := sourcePlan{release: revision.Release, commit: revision.Commit}
	uses, err := repo.skillUses(ctx, lender, folders, ids)
	if err != nil {
		return err
	}
	installed, err := installedSkills(lender, plan, ids, uses)
	if err != nil {
		return err
	}
	files := map[string]treeEntry{}
	for _, id := range slices.Sorted(maps.Keys(installed)) {
		level, ok := levels[id]
		if !ok {
			level = skills.LevelOffered
		}
		usedBy := []string{}
		for _, user := range installed[id].UsedBy {
			usedBy = append(usedBy, l.name+":"+user)
		}
		imported.Snapshot.Skills[l.name+":"+id] = library.ImportedSkill{Level: level, Included: installed[id].Included, UsedBy: usedBy}
		maps.Copy(files, folders[id])
	}
	retained := 0
	for _, data := range imported.Snapshot.Files {
		retained += len(data)
	}
	read, executable, err := repo.readFiles(ctx, l.library, files, retained)
	if err != nil {
		return err
	}
	prefix := library.BorrowedDirectory + "/" + l.name + "/"
	for name, data := range read {
		imported.Snapshot.Files[prefix+name] = data
	}
	for _, name := range executable {
		imported.Snapshot.Executable = append(imported.Snapshot.Executable, prefix+name)
	}
	if imported.Snapshot.Borrowed == nil {
		imported.Snapshot.Borrowed = map[string]library.BorrowedSource{}
	}
	imported.Snapshot.Borrowed[l.name] = revision
	return nil
}

// revision returns the revision the lending source's skills come from, fetched, and reports whether a source record
// locked it, which sync never moves.
func (l lendingSource) revision(ctx context.Context, repo *repository, recorded, planned library.BorrowedSource, isPlanned bool) (library.BorrowedSource, bool, error) {
	if l.override != nil {
		revision := library.BorrowedSource{Repository: l.override.Repository, Ref: l.override.Ref, Release: l.overridden.Release, Commit: l.overridden.Commit, Source: l.override.Name, NonLibrary: l.overridden.NonLibrary}
		return revision, true, repo.fetchCommits(ctx, []string{revision.Commit}, fmt.Sprintf("The commit that vendor/%s/_source.json records for sources.%s is missing from its repository, so the repository may have rewritten its history. Run syllabus project sync again, or delete vendor/%s and run syllabus project sync to resolve sources.%s again.", l.override.Name, l.override.Name, l.override.Name, l.override.Name))
	}
	if isPlanned && l.declares(planned) {
		return planned, false, repo.fetchCommits(ctx, []string{planned.Commit}, l.vanished("syllabus project update "+l.library))
	}
	if recorded.Source == "" && l.declares(recorded) {
		missing := fmt.Sprintf("The commit that vendor/%s/_source.json records for the skills library %s borrows from %s is missing from that repository, so it may have rewritten its history. Run syllabus project update %s to resolve the library's ref for %s again.", l.library, l.library, l.name, l.library, l.name)
		if err := repo.fetchCommits(ctx, []string{recorded.Commit}, missing); err != nil {
			return library.BorrowedSource{}, false, err
		}
		isLibrary, err := repo.isLibrary(ctx, recorded.Commit)
		recorded.NonLibrary = !isLibrary
		return recorded, true, err
	}
	revision, err := l.resolve(ctx, repo)
	return revision, false, err
}

// vanished explains a commit of the lending source that its repository no longer has, although Syllabus just read it,
// and names command, what to run again.
func (l lendingSource) vanished(command string) string {
	return fmt.Sprintf("A commit of the repository library %s borrows skills from as %s disappeared while Syllabus read it, so the repository's history changed. Run %s again.", l.library, l.name, command)
}

// declares reports whether revision records the repository and ref the library declares for the lending source.
func (l lendingSource) declares(revision library.BorrowedSource) bool {
	return revision.Commit != "" && revision.Repository == l.declared.Repository && revision.Ref.Equal(l.declared.Ref)
}

// resolve resolves the ref the library declares for the lending source, or, without one, the repository's newest
// library release, and fetches the revision.
func (l lendingSource) resolve(ctx context.Context, repo *repository) (library.BorrowedSource, error) {
	p := &planner{ctx: ctx, repo: repo, source: rules.Source{Name: l.name, Repository: l.declared.Repository, Ref: l.declared.Ref}}
	revision := library.BorrowedSource{Repository: l.declared.Repository, Ref: l.declared.Ref}
	var err error
	if l.declared.Ref.IsZero() {
		history, historyErr := p.releases()
		if historyErr != nil {
			return library.BorrowedSource{}, historyErr
		}
		newest := history.newest()
		if newest == nil {
			return library.BorrowedSource{}, fail("ref-required", fmt.Sprintf("Library %s borrows skills from its source %s, whose repository has no release/<number> tags, so the library must name a ref for it. Ask the library's maintainer to set sources.%s.ref in its .syllabus/config.yaml, or declare the repository as a source of your own with a ref, which overrides the library's.", l.library, l.name, l.name), nil)
		}
		revision.Release, revision.Commit = newest.number, newest.commit
		if err := repo.fetchCommits(ctx, []string{revision.Commit}, l.vanished("the command")); err != nil {
			return library.BorrowedSource{}, err
		}
	} else if revision.Release, revision.Commit, err = p.resolveRef(); err != nil {
		return library.BorrowedSource{}, l.libraryRef(err)
	} else if err := repo.fetchCommits(ctx, []string{revision.Commit}, l.vanished("the command")); err != nil {
		return library.BorrowedSource{}, err
	}
	isLibrary, err := repo.isLibrary(ctx, revision.Commit)
	revision.NonLibrary = !isLibrary
	return revision, err
}

// libraryRef rewords a failure resolving the ref the library declares, which names sources.<name>.ref as if it
// were the project's, to name the library's configuration.
func (l lendingSource) libraryRef(err error) error {
	var failure *Error
	if !errors.As(err, &failure) || !strings.HasPrefix(failure.Problem, "sources."+l.name+".ref") {
		return err
	}
	return fail(failure.Code, fmt.Sprintf("Library %s borrows skills from its source %s at ref %s, which the repository doesn't resolve: %s Ask the library's maintainer to fix sources.%s.ref in its .syllabus/config.yaml, or declare the repository as a source of your own with a ref, which overrides the library's.", l.library, l.name, l.declared.Ref, strings.TrimSpace(strings.TrimPrefix(failure.Problem, "sources."+l.name+".ref:")), l.name), nil)
}

// missing explains a borrowed skill, id, that the lending source's revision doesn't have: a revision a record or an
// overriding source locks needs an update or another ref, and any other means the library lists a skill its
// borrowed repository lacks.
func (l lendingSource) missing(id string, revision library.BorrowedSource, kept bool) error {
	where := revisionName(revision.Release, revision.Commit)
	switch {
	case l.override != nil:
		return fail("skill-not-in-revision", fmt.Sprintf("Library %s lists %s, which isn't in %s, the revision of sources.%s, which declares the same repository and so supplies the skills the library borrows from it. Run syllabus project update %s, or set sources.%s.ref to a revision that has the skill, then run syllabus project sync.", l.library, id, where, l.override.Name, l.override.Name, l.override.Name), nil)
	case kept:
		return fail("skill-not-in-revision", fmt.Sprintf("Library %s lists %s, which isn't in %s, the revision vendor/%s/_source.json records for the skills it borrows from %s. Sync never moves that revision; run syllabus project update %s, which resolves the library's ref for %s again.", l.library, id, where, l.library, l.name, l.library, l.name), nil)
	}
	return &ValidationError{Location: "sources." + l.library, Problem: fmt.Sprintf("library %s lists %s, which its repository %s doesn't have at %s; ask the library's maintainer to fix library.skills or sources.%s.ref", l.library, id, l.declared.Repository, where, l.name)}
}

// planBorrowed resolves again, as an update of the whole library source does, the ref the library names for each
// source it borrows skills that the project takes from, given manifest, the library's .syllabus/config.yaml at the
// revision the update moves to. It returns the revisions by borrowed source name, and a preview row for each whose
// commit differs from the one recorded, the source's snapshot, holds. A borrowed repository that a project source
// declares too is left out, since that source's revision supplies its skills.
func planBorrowed(ctx context.Context, configuration rules.Configuration, source rules.Source, manifest []byte, recorded *library.Snapshot, options Options) (map[string]library.BorrowedSource, []BorrowedUpdate, error) {
	parsed, err := rules.ParseLibraryManifest(manifest, source.Name+"/"+librarytree.RepositoryPath(librarytree.Manifest))
	if err != nil {
		return nil, nil, err
	}
	selected, err := borrowedSkills(parsed, source, 0)
	if err != nil {
		return nil, nil, err
	}
	revisions, rows := map[string]library.BorrowedSource{}, []BorrowedUpdate{}
	for _, name := range slices.Sorted(maps.Keys(selected)) {
		lender := lendingSource{library: source.Name, name: name, declared: parsed.Borrowed[name]}
		if overridingSource(configuration, source.Name, lender.declared.Repository) != nil {
			continue
		}
		revision, err := lender.resolveOnce(ctx, options)
		if err != nil {
			return nil, nil, err
		}
		revisions[name] = revision
		from := ""
		if recorded != nil {
			from = recorded.Borrowed[name].Commit
		}
		if from == revision.Commit {
			continue
		}
		ids := []string{}
		for _, skill := range selected[name] {
			ids = append(ids, skill.id)
		}
		rows = append(rows, BorrowedUpdate{Name: name, Repository: revision.Repository, Ref: revision.Ref.String(), From: from, To: revision.Commit, Skills: ids})
	}
	return revisions, rows, nil
}

// resolveOnce resolves the lending source's ref in a repository of its own, within one import deadline.
func (l lendingSource) resolveOnce(ctx context.Context, options Options) (_ library.BorrowedSource, err error) {
	ctx, cancel, err := withTimeout(ctx, options)
	if err != nil {
		return library.BorrowedSource{}, err
	}
	defer cancel()
	repo, err := openRepository(ctx, rules.Source{Name: l.name, Repository: l.declared.Repository}, options)
	if err != nil {
		return library.BorrowedSource{}, err
	}
	defer func() {
		if closeErr := repo.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	return l.resolve(ctx, repo)
}

// manifestAt returns the .syllabus/config.yaml of a fetched commit, or nil when it has none.
func (r *repository) manifestAt(ctx context.Context, commit string) ([]byte, error) {
	tree, err := r.repositoryTree(ctx, commit)
	if err != nil {
		return nil, err
	}
	data, _, err := r.manifest(ctx, tree)
	return data, err
}
