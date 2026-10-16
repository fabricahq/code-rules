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
	// planned maps the name of a library source to the borrowed revisions an update planned for it, by the commit
	// that supplies the library's skills, then by borrowed source name, which the import takes as they are.
	planned map[string]map[string]map[string]library.BorrowedSource
}

// borrowedSkill is a borrowed skill a project takes from a library source: its ID in the library, such as
// mattpocock:skills/to-spec, and the level the library lists it at.
type borrowedSkill struct {
	id, level string
}

// borrowedSkills returns, by borrowed source name, the borrowed skills source takes from manifest, its library's:
// every required one, the recommended ones when skills.include names recommended, and each offered one that it
// names or includes with "*", which, as for a source's own skills, leaves out one in a folder whose name starts with
// a dot. It fails when skills.include or skills.exclude names a borrowed skill the library
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
		_, folder, _ := library.SplitBorrowed(id)
		switch {
		case level == skills.LevelRequired:
		case slices.Contains(source.Skills, id):
		case slices.Contains(source.Skills, skills.Wildcard) && !library.HiddenSkill(folder):
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
		planned, isPlanned := borrow.planned[source.Name][imported.Snapshot.Commit][name]
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
	retained := 0
	for _, data := range imported.Snapshot.Files {
		retained += len(data)
	}
	read, err := l.read(ctx, repo, revision, chosen, kept, retained)
	if err != nil {
		return err
	}
	for id, skill := range read.skills {
		usedBy := []string{}
		for _, user := range skill.UsedBy {
			usedBy = append(usedBy, l.name+":"+user)
		}
		imported.Snapshot.Skills[l.name+":"+id] = library.ImportedSkill{Level: skill.Level, Included: skill.Included, UsedBy: usedBy}
	}
	prefix := library.BorrowedDirectory + "/" + l.name + "/"
	for name, data := range read.files {
		imported.Snapshot.Files[prefix+name] = data
	}
	for _, name := range read.executable {
		imported.Snapshot.Executable = append(imported.Snapshot.Executable, prefix+name)
	}
	if imported.Snapshot.Borrowed == nil {
		imported.Snapshot.Borrowed = map[string]library.BorrowedSource{}
	}
	imported.Snapshot.Borrowed[l.name] = revision
	return nil
}

// read reads, from repo at revision, the lending source's skills that chosen names and every skill they use, as a
// source's own skills bring the skills they use, by their IDs in the borrowed repository, with their files by path
// in that repository. kept and retained are as missing and readFiles take them.
func (l lendingSource) read(ctx context.Context, repo *repository, revision library.BorrowedSource, chosen []borrowedSkill, kept bool, retained int) (importedSkills, error) {
	tree, err := repo.repositoryTree(ctx, revision.Commit)
	if err != nil {
		return importedSkills{}, err
	}
	folders := skillFolders(tree, !revision.NonLibrary)
	ids, levels := []string{}, map[string]string{}
	for _, skill := range chosen {
		_, id, _ := library.SplitBorrowed(skill.id)
		if _, ok := folders[id]; !ok {
			return importedSkills{}, l.missing(skill.id, revision, kept)
		}
		ids, levels[id] = append(ids, id), skill.level
	}
	lender := rules.Source{Name: l.library}
	plan := sourcePlan{release: revision.Release, commit: revision.Commit}
	uses, err := repo.skillUses(ctx, lender, folders, ids)
	if err != nil {
		return importedSkills{}, err
	}
	installed, err := installedSkills(lender, plan, ids, uses)
	if err != nil {
		return importedSkills{}, err
	}
	result := importedSkills{skills: map[string]library.ImportedSkill{}}
	files := map[string]treeEntry{}
	for _, id := range slices.Sorted(maps.Keys(installed)) {
		level, ok := levels[id]
		if !ok {
			level = skills.LevelOffered
		}
		result.skills[id] = library.ImportedSkill{Level: level, Included: installed[id].Included, UsedBy: installed[id].UsedBy}
		maps.Copy(files, folders[id])
	}
	result.files, result.executable, err = repo.readFiles(ctx, l.library, files, retained)
	return result, err
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

// planBorrowed plans the revision of each source a library borrows skills from, given manifest, the library's
// .syllabus/config.yaml at a revision the update can take source's skills from, and recorded, the source's snapshot,
// or nil. It resolves the ref the library names again, as an update of the whole source does, unless keep is set,
// which keeps the recorded revision while the library still names its repository and ref, as sync does. A borrowed repository that
// a project source declares too comes from overridden, that source's revision once the update is applied, by source
// name. A non-nil only limits the plan to the borrowed sources it reports true for. It returns the revisions it
// resolved, by borrowed source name, leaving out overridden ones, and a preview row for each borrowed source whose
// commit or skills differ from those recorded, including one the project no longer takes any skills from.
func planBorrowed(ctx context.Context, configuration rules.Configuration, source rules.Source, manifest []byte, recorded *library.Snapshot, keep bool, overridden map[string]library.Snapshot, only func(name string) bool, options Options) (map[string]library.BorrowedSource, []BorrowedUpdate, error) {
	parsed, err := rules.ParseLibraryManifest(manifest, source.Name+"/"+librarytree.RepositoryPath(librarytree.Manifest))
	if err != nil {
		return nil, nil, err
	}
	selected, err := borrowedSkills(parsed, source, 0)
	if err != nil {
		return nil, nil, err
	}
	before := library.Snapshot{}
	if recorded != nil {
		before = *recorded
	}
	revisions, rows := map[string]library.BorrowedSource{}, []BorrowedUpdate{}
	names := maps.Clone(selected)
	for name := range before.Borrowed {
		if _, ok := names[name]; !ok {
			names[name] = nil
		}
	}
	for _, name := range slices.Sorted(maps.Keys(names)) {
		if only != nil && !only(name) {
			continue
		}
		was := before.BorrowedFrom(name)
		if len(selected[name]) == 0 {
			// The project takes no skills from the source any more, so the update removes those it installed.
			gone := before.Borrowed[name]
			row := BorrowedUpdate{Name: name, Repository: gone.Repository, Ref: gone.Ref.String(), From: gone.Commit, Skills: []string{}, Changes: skillChanges(&was, importedSkills{})}
			for i := range row.Changes {
				row.Changes[i].ID = name + ":" + row.Changes[i].ID
			}
			rows = append(rows, row)
			continue
		}
		lender := lendingSource{library: source.Name, name: name, declared: parsed.Borrowed[name]}
		lender.override = overridingSource(configuration, source.Name, lender.declared.Repository)
		if lender.override != nil {
			lender.overridden = overridden[lender.override.Name]
		}
		revision, row, err := lender.plan(ctx, selected[name], was, before.Borrowed[name], keep, options)
		if err != nil {
			return nil, nil, err
		}
		if lender.override == nil {
			revisions[name] = revision
		}
		if row != nil {
			rows = append(rows, *row)
		}
	}
	return revisions, rows, nil
}

// plan fetches the lending source's revision in a repository of its own, within one import deadline, and returns it,
// and the preview row for it when its commit or the skills the project takes from it differ from was's, which holds
// the skills the project recorded from the source, as Snapshot.BorrowedFrom gives them. The revision is the
// overriding source's, when there is one; else recorded, when keep is set and it records the ref the library names;
// else the library's ref, resolved now. The row compares the files of chosen, the skills the project takes, and of
// the skills they use, at the two commits, as skillChanges compares a source's own skills.
func (l lendingSource) plan(ctx context.Context, chosen []borrowedSkill, was library.Snapshot, recorded library.BorrowedSource, keep bool, options Options) (_ library.BorrowedSource, _ *BorrowedUpdate, err error) {
	ctx, cancel, err := withTimeout(ctx, options)
	if err != nil {
		return library.BorrowedSource{}, nil, err
	}
	defer cancel()
	address := l.declared.Repository
	if l.override != nil {
		address = l.override.Repository
	}
	repo, err := openRepository(ctx, rules.Source{Name: l.name, Repository: address}, options)
	if err != nil {
		return library.BorrowedSource{}, nil, err
	}
	defer func() {
		if closeErr := repo.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	if !keep {
		recorded = library.BorrowedSource{}
	}
	revision, kept, err := l.revision(ctx, repo, recorded, library.BorrowedSource{}, false)
	if err != nil {
		return library.BorrowedSource{}, nil, err
	}
	read, err := l.read(ctx, repo, revision, chosen, kept, 0)
	if err != nil {
		return library.BorrowedSource{}, nil, err
	}
	row := BorrowedUpdate{Name: l.name, Repository: revision.Repository, Ref: revision.Ref.String(), From: was.Commit, To: revision.Commit, Skills: []string{}, Changes: skillChanges(&was, read)}
	if was.Commit == revision.Commit && len(row.Changes) == 0 {
		return revision, nil, nil
	}
	for _, skill := range chosen {
		row.Skills = append(row.Skills, skill.id)
	}
	for i := range row.Changes {
		row.Changes[i].ID = l.name + ":" + row.Changes[i].ID
	}
	return revision, &row, nil
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
