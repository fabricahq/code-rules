// Choose each imported rule's version from a source's configuration, its recorded snapshot, and its library releases.

package imports

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// sourcePlan is what one source imports: the revision that supplies its library-wide files, and each imported
// rule's version and the commit that supplies its files.
type sourcePlan struct {
	// release is the library release that supplies the library-wide files, or 0 when ref isn't a library release.
	release int
	commit  string
	rules   map[string]library.ImportedRule
	// individual lists the individually selected rules to load; it omits entries naming retired rules.
	individual []string
	warnings   []string
}

// planner chooses versions for one source, reading the library's release history only when a choice needs it.
type planner struct {
	ctx    context.Context
	repo   *repository
	source rules.Source
	// recorded is the source's snapshot from the last sync, or nil for a new source or a changed repository.
	recorded *library.Snapshot
	// history is nil until first needed.
	history *releaseHistory
}

// planSource chooses what source imports, as project sync does: recorded versions stay, and versions are chosen
// only for new sources, changed repositories, newly selected rules, added or changed pins, and a changed ref.
// Every commit the plan names is fetched when it returns.
func planSource(ctx context.Context, repo *repository, source rules.Source, recorded *library.Snapshot) (sourcePlan, error) {
	p := newPlanner(ctx, repo, source, recorded)
	plan, err := p.choose()
	if err != nil {
		return sourcePlan{}, err
	}
	if err := p.requireUnmoved(plan); err != nil {
		return sourcePlan{}, err
	}
	commits := []string{plan.commit}
	for _, rule := range plan.rules {
		commits = append(commits, rule.Commit)
	}
	if err := repo.fetchCommits(ctx, commits, fmt.Sprintf("A commit that vendor/%s/_source.json records is missing from the library's repository. Delete vendor/%s and run code-rules project sync to choose versions again.", source.Name, source.Name)); err != nil {
		return sourcePlan{}, err
	}
	slices.Sort(plan.individual)
	return plan, nil
}

// newPlanner plans source against recorded, which it ignores when it records another repository.
func newPlanner(ctx context.Context, repo *repository, source rules.Source, recorded *library.Snapshot) *planner {
	if recorded != nil && recorded.Repository != source.Repository {
		recorded = nil
	}
	return &planner{ctx: ctx, repo: repo, source: source, recorded: recorded}
}

// choose plans what the source imports as project sync does, without fetching the commits the plan names that
// choosing didn't need.
func (p *planner) choose() (sourcePlan, error) {
	if p.source.Ref != "" {
		return p.planRevision()
	}
	return p.planVersions()
}

// releases returns the library's release history, reading it on first use; it may be empty.
func (p *planner) releases() (releaseHistory, error) {
	if p.history == nil {
		history, err := p.repo.loadHistory(p.ctx)
		if err != nil {
			return releaseHistory{}, err
		}
		p.history = &history
	}
	return *p.history, nil
}

// versioned returns the release history, failing with code releases-not-found before the first library release.
func (p *planner) versioned() (releaseHistory, error) {
	history, err := p.releases()
	if err != nil {
		return releaseHistory{}, err
	}
	if history.newest() == nil {
		return releaseHistory{}, fail("releases-not-found", fmt.Sprintf("The library that source %s imports has no release/<number> tags, because it hasn't published its first library release. Ask the maintainer to publish a library release, or import a commit with sources.%s.ref.", p.source.Name, p.source.Name), nil)
	}
	return history, nil
}

// planVersions plans a source that follows rule versions: recorded rules keep their versions unless a pin moves
// them or they have none, and newly selected rules get their pinned or newest version.
func (p *planner) planVersions() (sourcePlan, error) {
	plan := sourcePlan{rules: map[string]library.ImportedRule{}, individual: []string{}, warnings: []string{}}
	recorded := p.recorded
	if recorded != nil {
		for _, id := range slices.Sorted(maps.Keys(recorded.Rules)) {
			if p.selected(id) {
				plan.rules[id] = recorded.Rules[id]
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(plan.rules)) {
		rule := plan.rules[id]
		if pin, pinned := p.source.Pins[id]; pinned && (rule.Version == nil || *rule.Version != pin.Version) {
			chosen, err := p.pinnedVersion(id, pin)
			if err != nil {
				return sourcePlan{}, err
			}
			plan.rules[id] = chosen
		} else if rule.Version == nil {
			// Only a removed ref leaves unreleased rules; each gets its newest version, if it has one.
			history, err := p.versioned()
			if err != nil {
				return sourcePlan{}, err
			}
			if _, published := history.newest().record.Rules[id]; !published {
				delete(plan.rules, id)
				continue
			}
			if plan.rules[id], err = p.newestVersion(id); err != nil {
				return sourcePlan{}, err
			}
		}
	}
	chosen := []string{}
	if recorded == nil || !sameGroupSelection(recorded.Selection, p.source.Groups) {
		history, err := p.versioned()
		if err != nil {
			return sourcePlan{}, err
		}
		for _, id := range slices.Sorted(maps.Keys(history.newest().record.Rules)) {
			group := ruleGroup(id)
			if _, imported := plan.rules[id]; !imported && p.source.Groups.Includes(group) && (recorded == nil || !recorded.Selection.Includes(group)) {
				chosen = append(chosen, id)
			}
		}
	}
	for _, id := range p.source.Rules {
		if _, imported := plan.rules[id]; imported || slices.Contains(chosen, id) {
			plan.individual = append(plan.individual, id)
			continue
		}
		if recorded != nil && slices.Contains(recorded.RuleSelection, id) && !hasRule(recorded, id) {
			plan.warnings = append(plan.warnings, p.retiredEntry("rules", id))
			continue
		}
		history, err := p.versioned()
		if err != nil {
			return sourcePlan{}, err
		}
		if _, current := history.newest().record.Rules[id]; current {
			chosen = append(chosen, id)
			plan.individual = append(plan.individual, id)
			continue
		}
		if err := p.unimported("rules", id, &plan); err != nil {
			return sourcePlan{}, err
		}
	}
	for _, id := range chosen {
		var err error
		if pin, pinned := p.source.Pins[id]; pinned {
			plan.rules[id], err = p.pinnedVersion(id, pin)
		} else {
			plan.rules[id], err = p.newestVersion(id)
		}
		if err != nil {
			return sourcePlan{}, err
		}
	}
	if err := p.requireEntries(&plan); err != nil {
		return sourcePlan{}, err
	}
	return plan, p.libraryWideRelease(&plan)
}

// libraryWideRelease sets the plan's library release to the newest one among its rule versions. A source that
// imports no rules keeps its recorded library release while its selection is unchanged, and otherwise gets the
// newest library release.
func (p *planner) libraryWideRelease(plan *sourcePlan) error {
	for _, rule := range plan.rules {
		if rule.Release > plan.release {
			plan.release, plan.commit = rule.Release, rule.Commit
		}
	}
	if plan.release != 0 {
		if p.history != nil && p.history.release(plan.release) != nil {
			plan.commit = p.history.release(plan.release).commit
		}
		return nil
	}
	recorded := p.recorded
	if recorded != nil && recorded.Ref == "" && recorded.Release != 0 && sameGroupSelection(recorded.Selection, p.source.Groups) && slices.Equal(recorded.RuleSelection, p.source.Rules) {
		plan.release, plan.commit = recorded.Release, recorded.Commit
		return nil
	}
	history, err := p.versioned()
	if err != nil {
		return err
	}
	plan.release, plan.commit = history.newest().number, history.newest().commit
	return nil
}

// planRevision plans a source that imports one revision with ref. A recorded snapshot of the same ref keeps its
// commit and rule versions; otherwise the ref is resolved again. Rules newly selected at the revision record the
// version its library release record lists, or, for any other revision, the newest published version whose files
// match, or none.
func (p *planner) planRevision() (sourcePlan, error) {
	plan := sourcePlan{rules: map[string]library.ImportedRule{}, individual: []string{}, warnings: []string{}}
	recorded := p.recorded
	if recorded != nil && recorded.Ref == p.source.Ref {
		plan.release, plan.commit = recorded.Release, recorded.Commit
		if err := p.repo.fetchCommits(p.ctx, []string{plan.commit}, fmt.Sprintf("The commit that vendor/%s/_source.json records for sources.%s.ref is missing from the library's repository. Delete vendor/%s and run code-rules project sync to resolve the ref again.", p.source.Name, p.source.Name, p.source.Name)); err != nil {
			return sourcePlan{}, err
		}
	} else {
		recorded = nil
		var err error
		if plan.release, plan.commit, err = p.resolveRef(); err != nil {
			return sourcePlan{}, err
		}
	}
	tree, err := p.repo.tree(p.ctx, plan.commit)
	if err != nil {
		return sourcePlan{}, err
	}
	terms, err := p.repo.terms(p.ctx, p.source.Name, tree)
	if err != nil {
		return sourcePlan{}, err
	}
	present := rulesInTree(tree, terms)
	imported := []string{}
	for _, id := range present {
		if p.source.Groups.Includes(ruleGroup(id)) {
			imported = append(imported, id)
		}
	}
	for _, id := range p.source.Rules {
		switch {
		case slices.Contains(present, id):
			plan.individual = append(plan.individual, id)
			if !slices.Contains(imported, id) {
				imported = append(imported, id)
			}
		case recorded != nil && slices.Contains(recorded.RuleSelection, id) && !hasRule(recorded, id):
			plan.warnings = append(plan.warnings, p.retiredEntry("rules", id))
		default:
			if err := p.unimported("rules", id, &plan); err != nil {
				return sourcePlan{}, err
			}
		}
	}
	slices.Sort(imported)
	for _, id := range imported {
		if recorded != nil && hasRule(recorded, id) {
			plan.rules[id] = recorded.Rules[id]
			continue
		}
		if plan.rules[id], err = p.revisionVersion(id, plan); err != nil {
			return sourcePlan{}, err
		}
	}
	if err := p.requireEntries(&plan); err != nil {
		return sourcePlan{}, err
	}
	if plan.release == 0 {
		plan.warnings = append(plan.warnings, unreleasedWarning(p.source, plan.rules))
	}
	return plan, nil
}

// resolveRef returns the commit the source's ref names and its library release number, or 0 when it isn't a
// library release tag. A missing tag or commit fails with code version-not-found.
func (p *planner) resolveRef() (int, string, error) {
	ref, err := rules.ParseGitRef(p.source.Ref, "sources."+p.source.Name+".ref")
	if err != nil {
		return 0, "", err
	}
	if ref.Kind == rules.GitRefTag {
		if number, err := rules.ParseReleaseTag(strings.TrimPrefix(ref.Name, "refs/tags/")); err == nil {
			history, err := p.releases()
			if err != nil {
				return 0, "", err
			}
			release := history.release(number)
			if release == nil {
				return 0, "", fail("version-not-found", fmt.Sprintf("sources.%s.ref: the library has no library release %s; check the ref.", p.source.Name, p.source.Ref), nil)
			}
			return number, release.commit, nil
		}
	}
	commit, err := p.repo.fetchRef(p.ctx, p.source)
	return 0, commit, err
}

// revisionVersion chooses the version of rule id imported at the plan's revision.
func (p *planner) revisionVersion(id string, plan sourcePlan) (library.ImportedRule, error) {
	history, err := p.releases()
	if err != nil {
		return library.ImportedRule{}, err
	}
	if plan.release != 0 {
		version, listed := history.release(plan.release).record.Rules[id]
		if !listed {
			return library.ImportedRule{}, fail("invalid-release-tag", fmt.Sprintf("Library release release/%d contains rule %s, but its release record doesn't list it. Don't create or move release tags by hand.", plan.release, id), nil)
		}
		return p.publishedVersion(id, version)
	}
	files, err := p.repo.ruleFiles(p.ctx, plan.commit, id)
	if err != nil {
		return library.ImportedRule{}, err
	}
	for _, release := range history.published(id) {
		published, err := p.repo.ruleFiles(p.ctx, release.commit, id)
		if err != nil {
			return library.ImportedRule{}, err
		}
		if maps.Equal(files, published) {
			version := release.record.Rules[id]
			return library.ImportedRule{Version: &version, Release: release.number, Commit: release.commit}, nil
		}
	}
	return library.ImportedRule{Commit: plan.commit}, nil
}

// pinnedVersion returns the pinned version of rule id, failing with code version-not-found when the rule never
// published it.
func (p *planner) pinnedVersion(id string, pin rules.Pin) (library.ImportedRule, error) {
	history, err := p.versioned()
	if err != nil {
		return library.ImportedRule{}, err
	}
	if history.publisher(id, pin.Version) == nil {
		return library.ImportedRule{}, fail("version-not-found", fmt.Sprintf("sources.%s.pins.%s: the rule never published version %s; check the pin.", p.source.Name, id, pin.Version), nil)
	}
	return p.publishedVersion(id, pin.Version)
}

// newestVersion returns the version of rule id in the newest library release.
func (p *planner) newestVersion(id string) (library.ImportedRule, error) {
	history, err := p.versioned()
	if err != nil {
		return library.ImportedRule{}, err
	}
	return p.publishedVersion(id, history.newest().record.Rules[id])
}

// publishedVersion returns version of rule id with the library release that published it.
func (p *planner) publishedVersion(id string, version rules.RuleVersion) (library.ImportedRule, error) {
	history, err := p.releases()
	if err != nil {
		return library.ImportedRule{}, err
	}
	release := history.publisher(id, version)
	if release == nil {
		return library.ImportedRule{}, fail("invalid-release-tag", fmt.Sprintf("No library release's record publishes version %s of rule %s, although a later record lists it. Don't create or move release tags by hand.", version, id), nil)
	}
	return library.ImportedRule{Version: &version, Release: release.number, Commit: release.commit}, nil
}

// requireEntries checks that each pin and exclusion names an imported rule. An entry naming a retired rule adds a
// warning; any other fails.
func (p *planner) requireEntries(plan *sourcePlan) error {
	for _, id := range slices.Sorted(maps.Keys(p.source.Pins)) {
		if _, imported := plan.rules[id]; imported {
			continue
		}
		// A pin that the last sync recorded without importing its rule already named a retired rule.
		if recorded := p.recorded; recorded != nil && !hasRule(recorded, id) && hasPin(recorded, id, p.source.Pins[id].Version) {
			plan.warnings = append(plan.warnings, p.retiredEntry("pins", id))
			continue
		}
		if err := p.unimported("pins", id, plan); err != nil {
			return err
		}
	}
	for _, id := range slices.Sorted(maps.Keys(p.source.Exclude)) {
		if _, imported := plan.rules[id]; !imported {
			if err := p.unimported("exclude", id, plan); err != nil {
				return err
			}
		}
	}
	return nil
}

// unimported handles a configuration entry naming a rule the source doesn't import: a warning for a retired
// rule, and a validation error for any other.
func (p *planner) unimported(field, id string, plan *sourcePlan) error {
	history, err := p.releases()
	if err != nil {
		return err
	}
	if history.retired(id) {
		plan.warnings = append(plan.warnings, p.retiredEntry(field, id))
		return nil
	}
	location := "sources." + p.source.Name + "." + field + "." + id
	if field == "rules" {
		return &rules.ValidationError{Location: "sources." + p.source.Name + ".rules", Problem: "the library has no rule " + id + " to import; check the rule ID"}
	}
	return &rules.ValidationError{Location: location, Problem: "rule is not imported by this source; name a rule that its groups or rules select"}
}

// retiredEntry warns that a configuration entry names a retired rule.
func (p *planner) retiredEntry(field, id string) string {
	return fmt.Sprintf("sources.%s.%s names %s, a rule the library retired, so the entry no longer does anything; delete it.", p.source.Name, field, id)
}

// requireUnmoved fails when a rule's recorded library release tag now names a different commit than the record.
func (p *planner) requireUnmoved(plan sourcePlan) error {
	if p.history == nil {
		return nil
	}
	for _, id := range slices.Sorted(maps.Keys(plan.rules)) {
		rule := plan.rules[id]
		if release := p.history.release(rule.Release); release != nil && release.commit != rule.Commit {
			return fail("invalid-release-tag", fmt.Sprintf("Library release tag release/%d now names a different commit than vendor/%s/_source.json records for %s. Library release tags must not move; ask the library's maintainer, or delete vendor/%s and run code-rules project sync to use the tag's current commit.", rule.Release, p.source.Name, id, p.source.Name), nil)
		}
	}
	return nil
}

// selected reports whether the source's configuration selects rule id, through its groups or its rules list.
func (p *planner) selected(id string) bool {
	return p.source.Groups.Includes(ruleGroup(id)) || slices.Contains(p.source.Rules, id)
}

// unreleasedWarning names a source that imports a revision other than a library release, and its unreleased rules.
func unreleasedWarning(source rules.Source, imported map[string]library.ImportedRule) string {
	unreleased := []string{}
	for _, id := range slices.Sorted(maps.Keys(imported)) {
		if imported[id].Version == nil {
			unreleased = append(unreleased, id)
		}
	}
	warning := fmt.Sprintf("Source %s imports %s, which isn't a library release, so rules with unreleased changes have no version.", source.Name, source.Ref)
	if len(unreleased) > 0 {
		warning += " Unreleased rules: " + strings.Join(unreleased, ", ") + "."
	}
	return warning
}

// ruleGroup returns the group of a valid library rule ID.
func ruleGroup(id string) string {
	parts := strings.SplitN(id, "/", 3)
	return parts[0] + "/" + parts[1]
}

// hasRule reports whether the snapshot imported rule id.
func hasRule(snapshot *library.Snapshot, id string) bool {
	_, ok := snapshot.Rules[id]
	return ok
}

// hasPin reports whether the snapshot recorded a pin of rule id to version.
func hasPin(snapshot *library.Snapshot, id string, version rules.RuleVersion) bool {
	pin, ok := snapshot.Pins[id]
	return ok && pin.Version == version
}

// sameGroupSelection reports whether two group selections request the same groups.
func sameGroupSelection(a, b rules.GroupSelection) bool {
	return a.Pattern == b.Pattern && slices.Equal(a.Groups, b.Groups)
}

// rulesInTree returns, sorted, the IDs of the rules whose Markdown files tree holds, skipping declared terms.
func rulesInTree(tree map[string]treeEntry, terms []string) []string {
	ids := []string{}
	for file := range tree {
		if id, ok := rules.VersionedRule(file); ok && file == id+".md" && !slices.Contains(terms, file) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}
